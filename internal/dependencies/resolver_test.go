package dependencies

import (
	"fmt"
	"testing"
)

// MockRegistry implements ModRegistry for testing.
type MockRegistry struct {
	mods map[string][]*ModInfo
}

func NewMockRegistry() *MockRegistry {
	return &MockRegistry{
		mods: make(map[string][]*ModInfo),
	}
}

func (r *MockRegistry) AddMod(mod *ModInfo) {
	r.mods[mod.ID] = append(r.mods[mod.ID], mod)
}

func (r *MockRegistry) GetMod(modID string, versionExpr string) (*ModInfo, error) {
	versions, ok := r.mods[modID]
	if !ok || len(versions) == 0 {
		return nil, fmt.Errorf("mod not found: %s", modID)
	}

	vr, err := ParseVersionRange(versionExpr)
	if err != nil {
		return nil, err
	}

	// Find best matching version
	for _, mod := range versions {
		v, err := ParseVersion(mod.Version)
		if err != nil {
			continue
		}
		if vr.Matches(v) {
			return mod, nil
		}
	}

	return nil, fmt.Errorf("no matching version for %s %s", modID, versionExpr)
}

func (r *MockRegistry) GetAvailableVersions(modID string) ([]*ModInfo, error) {
	versions, ok := r.mods[modID]
	if !ok {
		return nil, fmt.Errorf("mod not found: %s", modID)
	}
	return versions, nil
}

func (r *MockRegistry) GetLatestVersion(modID string) (*ModInfo, error) {
	versions, ok := r.mods[modID]
	if !ok || len(versions) == 0 {
		return nil, fmt.Errorf("mod not found: %s", modID)
	}
	return versions[0], nil
}

func TestNewResolver(t *testing.T) {
	registry := NewMockRegistry()
	resolver := NewResolver(registry)

	if resolver == nil {
		t.Fatal("NewResolver returned nil")
	}
	if resolver.registry == nil {
		t.Error("registry is nil")
	}
}

func TestResolveSimpleMod(t *testing.T) {
	registry := NewMockRegistry()
	registry.AddMod(&ModInfo{
		ID:      "simple-mod",
		Name:    "Simple Mod",
		Version: "1.0.0",
	})

	resolver := NewResolver(registry)
	result, err := resolver.Resolve([]*ModInfo{
		{ID: "simple-mod", Name: "Simple Mod", Version: "1.0.0"},
	})

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(result.ResolvedMods) != 1 {
		t.Errorf("Expected 1 resolved mod, got %d", len(result.ResolvedMods))
	}

	if len(result.Conflicts) != 0 {
		t.Errorf("Expected 0 conflicts, got %d", len(result.Conflicts))
	}

	if len(result.InstallOrder) != 1 {
		t.Errorf("Expected install order with 1 item, got %d", len(result.InstallOrder))
	}
}

func TestResolveDependencyChain(t *testing.T) {
	registry := NewMockRegistry()

	// mod-a depends on mod-b, mod-b depends on mod-c
	registry.AddMod(&ModInfo{
		ID:      "mod-c",
		Name:    "Mod C",
		Version: "1.0.0",
	})
	registry.AddMod(&ModInfo{
		ID:      "mod-b",
		Name:    "Mod B",
		Version: "1.0.0",
		Dependencies: []Dependency{
			{ModID: "mod-c", VersionExpr: ">=1.0.0", Required: true},
		},
	})
	registry.AddMod(&ModInfo{
		ID:      "mod-a",
		Name:    "Mod A",
		Version: "1.0.0",
		Dependencies: []Dependency{
			{ModID: "mod-b", VersionExpr: ">=1.0.0", Required: true},
		},
	})

	resolver := NewResolver(registry)
	result, err := resolver.Resolve([]*ModInfo{
		{
			ID:      "mod-a",
			Name:    "Mod A",
			Version: "1.0.0",
			Dependencies: []Dependency{
				{ModID: "mod-b", VersionExpr: ">=1.0.0", Required: true},
			},
		},
	})

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Should have resolved all 3 mods
	if len(result.ResolvedMods) != 3 {
		t.Errorf("Expected 3 resolved mods, got %d", len(result.ResolvedMods))
	}

	// Install order should have dependencies first
	if len(result.InstallOrder) != 3 {
		t.Errorf("Expected install order with 3 items, got %d", len(result.InstallOrder))
	}

	// mod-c should be before mod-b, mod-b before mod-a
	orderMap := make(map[string]int)
	for i, id := range result.InstallOrder {
		orderMap[id] = i
	}

	if orderMap["mod-c"] >= orderMap["mod-b"] {
		t.Error("mod-c should be installed before mod-b")
	}
	if orderMap["mod-b"] >= orderMap["mod-a"] {
		t.Error("mod-b should be installed before mod-a")
	}
}

func TestResolveMissingDependency(t *testing.T) {
	registry := NewMockRegistry()

	resolver := NewResolver(registry)
	result, err := resolver.Resolve([]*ModInfo{
		{
			ID:      "mod-a",
			Name:    "Mod A",
			Version: "1.0.0",
			Dependencies: []Dependency{
				{ModID: "missing-mod", VersionExpr: ">=1.0.0", Required: true},
			},
		},
	})

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(result.MissingDeps) != 1 {
		t.Errorf("Expected 1 missing dependency, got %d", len(result.MissingDeps))
	}

	if result.MissingDeps[0].ModID != "missing-mod" {
		t.Errorf("Expected missing dep 'missing-mod', got %q", result.MissingDeps[0].ModID)
	}
}

func TestResolveVersionConflict(t *testing.T) {
	registry := NewMockRegistry()

	// Both mods depend on shared-dep but with different versions
	registry.AddMod(&ModInfo{
		ID:      "shared-dep",
		Name:    "Shared Dependency",
		Version: "1.0.0",
	})
	registry.AddMod(&ModInfo{
		ID:      "shared-dep",
		Name:    "Shared Dependency",
		Version: "2.0.0",
	})

	resolver := NewResolver(registry)

	// First, resolve with version 1.0.0
	result, err := resolver.Resolve([]*ModInfo{
		{
			ID:      "mod-a",
			Name:    "Mod A",
			Version: "1.0.0",
			Dependencies: []Dependency{
				{ModID: "shared-dep", VersionExpr: "1.0.0", Required: true},
			},
		},
		{
			ID:      "mod-b",
			Name:    "Mod B",
			Version: "1.0.0",
			Dependencies: []Dependency{
				{ModID: "shared-dep", VersionExpr: "2.0.0", Required: true},
			},
		},
	})

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(result.Conflicts) == 0 {
		t.Error("Expected version conflict to be detected")
	}
}

func TestResolveOptionalDependency(t *testing.T) {
	registry := NewMockRegistry()

	resolver := NewResolver(registry)
	result, err := resolver.Resolve([]*ModInfo{
		{
			ID:      "mod-a",
			Name:    "Mod A",
			Version: "1.0.0",
			Dependencies: []Dependency{
				{ModID: "optional-mod", VersionExpr: ">=1.0.0", Required: false},
			},
		},
	})

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Optional dependencies that are missing should not appear in missing deps
	if len(result.MissingDeps) != 0 {
		t.Errorf("Optional missing deps should not be recorded, got %d", len(result.MissingDeps))
	}

	// Only mod-a should be resolved
	if len(result.ResolvedMods) != 1 {
		t.Errorf("Expected 1 resolved mod, got %d", len(result.ResolvedMods))
	}
}

func TestTopologicalSort(t *testing.T) {
	registry := NewMockRegistry()

	// Create a diamond dependency:
	// A depends on B and C
	// B and C both depend on D
	registry.AddMod(&ModInfo{ID: "mod-d", Version: "1.0.0"})
	registry.AddMod(&ModInfo{
		ID:      "mod-b",
		Version: "1.0.0",
		Dependencies: []Dependency{
			{ModID: "mod-d", VersionExpr: "*", Required: true},
		},
	})
	registry.AddMod(&ModInfo{
		ID:      "mod-c",
		Version: "1.0.0",
		Dependencies: []Dependency{
			{ModID: "mod-d", VersionExpr: "*", Required: true},
		},
	})

	resolver := NewResolver(registry)
	result, err := resolver.Resolve([]*ModInfo{
		{
			ID:      "mod-a",
			Version: "1.0.0",
			Dependencies: []Dependency{
				{ModID: "mod-b", VersionExpr: "*", Required: true},
				{ModID: "mod-c", VersionExpr: "*", Required: true},
			},
		},
	})

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// D must be installed before B and C, which must be before A
	orderMap := make(map[string]int)
	for i, id := range result.InstallOrder {
		orderMap[id] = i
	}

	if orderMap["mod-d"] >= orderMap["mod-b"] {
		t.Error("mod-d should be installed before mod-b")
	}
	if orderMap["mod-d"] >= orderMap["mod-c"] {
		t.Error("mod-d should be installed before mod-c")
	}
	if orderMap["mod-b"] >= orderMap["mod-a"] {
		t.Error("mod-b should be installed before mod-a")
	}
	if orderMap["mod-c"] >= orderMap["mod-a"] {
		t.Error("mod-c should be installed before mod-a")
	}
}

func TestHasConflicts(t *testing.T) {
	registry := NewMockRegistry()
	resolver := NewResolver(registry)

	if resolver.HasConflicts() {
		t.Error("New resolver should not have conflicts")
	}
}

func TestHasMissingDeps(t *testing.T) {
	registry := NewMockRegistry()
	resolver := NewResolver(registry)

	if resolver.HasMissingDeps() {
		t.Error("New resolver should not have missing deps")
	}
}

func TestFindBestVersion(t *testing.T) {
	versions := []*ModInfo{
		{ID: "mod", Version: "1.0.0"},
		{ID: "mod", Version: "1.5.0"},
		{ID: "mod", Version: "2.0.0"},
		{ID: "mod", Version: "2.5.0"},
	}

	tests := []struct {
		name        string
		constraints []string
		expect      string
		wantErr     bool
	}{
		{
			name:        "any version gets latest",
			constraints: []string{"*"},
			expect:      "2.5.0",
		},
		{
			name:        "specific version",
			constraints: []string{"1.5.0"},
			expect:      "1.5.0",
		},
		{
			name:        "range constraint",
			constraints: []string{">=1.0.0", "<2.0.0"},
			expect:      "1.5.0",
		},
		{
			name:        "no matching version",
			constraints: []string{">=3.0.0"},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FindBestVersion(versions, tt.constraints)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindBestVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if result.Version != tt.expect {
				t.Errorf("FindBestVersion() = %s, want %s", result.Version, tt.expect)
			}
		})
	}
}

func TestFindBestVersionEmpty(t *testing.T) {
	_, err := FindBestVersion([]*ModInfo{}, []string{"*"})
	if err == nil {
		t.Error("Expected error for empty versions list")
	}
}
