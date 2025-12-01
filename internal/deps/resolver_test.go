package deps

import (
	"fmt"
	"testing"
)

// mockProvider is a mock implementation of ModInfoProvider for testing.
type mockProvider struct {
	mods map[string]map[string]*ModInfo // modID -> version -> ModInfo
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		mods: make(map[string]map[string]*ModInfo),
	}
}

func (m *mockProvider) addMod(info *ModInfo) {
	if m.mods[info.ID] == nil {
		m.mods[info.ID] = make(map[string]*ModInfo)
	}
	m.mods[info.ID][info.Version] = info
}

func (m *mockProvider) GetModInfo(modID, version string) (*ModInfo, error) {
	versions, ok := m.mods[modID]
	if !ok {
		return nil, fmt.Errorf("mod not found: %s", modID)
	}
	info, ok := versions[version]
	if !ok {
		return nil, fmt.Errorf("version not found: %s@%s", modID, version)
	}
	return info, nil
}

func (m *mockProvider) GetLatestVersion(modID, constraint string) (*ModInfo, error) {
	versions, ok := m.mods[modID]
	if !ok {
		return nil, fmt.Errorf("mod not found: %s", modID)
	}

	var constraints *VersionConstraints
	var err error
	if constraint == "" || constraint == "*" {
		constraints = &VersionConstraints{
			Constraints: []*Constraint{{Op: "*", Raw: "*"}},
			Raw:         "*",
		}
	} else {
		constraints, err = ParseVersionConstraints(constraint)
		if err != nil {
			return nil, err
		}
	}

	var latest *ModInfo
	var latestVer *Version

	for v, info := range versions {
		ver, err := ParseVersion(v)
		if err != nil {
			continue
		}
		if !constraints.Matches(ver) {
			continue
		}
		if latest == nil || ver.Compare(latestVer) > 0 {
			latest = info
			latestVer = ver
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no version of %s matches constraint %s", modID, constraint)
	}

	return latest, nil
}

func (m *mockProvider) GetAllVersions(modID string) ([]*ModInfo, error) {
	versions, ok := m.mods[modID]
	if !ok {
		return nil, fmt.Errorf("mod not found: %s", modID)
	}

	var result []*ModInfo
	for _, info := range versions {
		result = append(result, info)
	}
	return result, nil
}

func TestResolver_SimpleDependency(t *testing.T) {
	provider := newMockProvider()

	// Add mod-a with a dependency on mod-b
	provider.addMod(&ModInfo{
		ID:      "mod-a",
		Name:    "Mod A",
		Version: "1.0.0",
		Dependencies: []*Dependency{
			{ID: "mod-b", VersionConstraint: ">=1.0.0", Type: Required},
		},
	})

	// Add mod-b
	provider.addMod(&ModInfo{
		ID:      "mod-b",
		Name:    "Mod B",
		Version: "1.2.0",
	})

	resolver := NewResolver(provider, nil)
	graph, err := resolver.Resolve("mod-a", "1.0.0")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if graph.Root == nil {
		t.Fatal("Expected root to be set")
	}

	if graph.Root.ID != "mod-a" {
		t.Errorf("Expected root ID 'mod-a', got '%s'", graph.Root.ID)
	}

	if len(graph.Root.Dependencies) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(graph.Root.Dependencies))
	}

	if graph.Root.Dependencies[0].ID != "mod-b" {
		t.Errorf("Expected dependency ID 'mod-b', got '%s'", graph.Root.Dependencies[0].ID)
	}
}

func TestResolver_TransitiveDependencies(t *testing.T) {
	provider := newMockProvider()

	// mod-a -> mod-b -> mod-c
	provider.addMod(&ModInfo{
		ID:      "mod-a",
		Name:    "Mod A",
		Version: "1.0.0",
		Dependencies: []*Dependency{
			{ID: "mod-b", VersionConstraint: ">=1.0.0", Type: Required},
		},
	})

	provider.addMod(&ModInfo{
		ID:      "mod-b",
		Name:    "Mod B",
		Version: "1.0.0",
		Dependencies: []*Dependency{
			{ID: "mod-c", VersionConstraint: ">=2.0.0", Type: Required},
		},
	})

	provider.addMod(&ModInfo{
		ID:      "mod-c",
		Name:    "Mod C",
		Version: "2.5.0",
	})

	resolver := NewResolver(provider, nil)
	graph, err := resolver.Resolve("mod-a", "1.0.0")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// Should have 3 mods in total
	if len(graph.AllMods) != 3 {
		t.Errorf("Expected 3 mods, got %d", len(graph.AllMods))
	}
}

func TestResolver_CircularDependency(t *testing.T) {
	provider := newMockProvider()

	// mod-a -> mod-b -> mod-a (circular)
	provider.addMod(&ModInfo{
		ID:      "mod-a",
		Name:    "Mod A",
		Version: "1.0.0",
		Dependencies: []*Dependency{
			{ID: "mod-b", VersionConstraint: ">=1.0.0", Type: Required},
		},
	})

	provider.addMod(&ModInfo{
		ID:      "mod-b",
		Name:    "Mod B",
		Version: "1.0.0",
		Dependencies: []*Dependency{
			{ID: "mod-a", VersionConstraint: ">=1.0.0", Type: Required},
		},
	})

	resolver := NewResolver(provider, nil)
	_, err := resolver.Resolve("mod-a", "1.0.0")
	if err == nil {
		t.Fatal("Expected circular dependency error")
	}

	resErr, ok := err.(*ResolutionError)
	if !ok {
		t.Fatalf("Expected ResolutionError, got %T", err)
	}

	if resErr.Type != ErrCircularDependency {
		t.Errorf("Expected ErrCircularDependency, got %s", resErr.Type)
	}
}

func TestResolver_OptionalDependency(t *testing.T) {
	provider := newMockProvider()

	provider.addMod(&ModInfo{
		ID:      "mod-a",
		Name:    "Mod A",
		Version: "1.0.0",
		Dependencies: []*Dependency{
			{ID: "mod-b", VersionConstraint: ">=1.0.0", Type: Required},
			{ID: "mod-c", VersionConstraint: ">=1.0.0", Type: Optional},
		},
	})

	provider.addMod(&ModInfo{
		ID:      "mod-b",
		Name:    "Mod B",
		Version: "1.0.0",
	})

	// mod-c is not available

	// Test with optional dependencies included
	resolver := NewResolver(provider, &ResolutionOptions{
		Strategy:        StrategyLatest,
		IncludeOptional: true,
	})
	graph, err := resolver.Resolve("mod-a", "1.0.0")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// Should have 2 mods (mod-a and mod-b, mod-c is missing but optional)
	if len(graph.AllMods) != 2 {
		t.Errorf("Expected 2 mods, got %d", len(graph.AllMods))
	}

	// Test with optional dependencies excluded
	resolver = NewResolver(provider, &ResolutionOptions{
		Strategy:        StrategyLatest,
		IncludeOptional: false,
	})
	graph, err = resolver.Resolve("mod-a", "1.0.0")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// Should have 2 mods
	if len(graph.AllMods) != 2 {
		t.Errorf("Expected 2 mods, got %d", len(graph.AllMods))
	}
}

func TestResolver_EmbeddedDependency(t *testing.T) {
	provider := newMockProvider()

	provider.addMod(&ModInfo{
		ID:      "mod-a",
		Name:    "Mod A",
		Version: "1.0.0",
		Dependencies: []*Dependency{
			{ID: "mod-embedded", VersionConstraint: "1.0.0", Type: Embedded},
		},
	})

	resolver := NewResolver(provider, nil)
	graph, err := resolver.Resolve("mod-a", "1.0.0")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// Check that embedded dependency is present
	if len(graph.Root.Dependencies) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(graph.Root.Dependencies))
	}

	if graph.Root.Dependencies[0].Type != Embedded {
		t.Errorf("Expected Embedded type, got %s", graph.Root.Dependencies[0].Type)
	}
}

func TestResolver_IncompatibleMods(t *testing.T) {
	provider := newMockProvider()

	provider.addMod(&ModInfo{
		ID:      "mod-a",
		Name:    "Mod A",
		Version: "1.0.0",
		Dependencies: []*Dependency{
			{ID: "mod-b", Type: Incompatible},
		},
	})

	provider.addMod(&ModInfo{
		ID:      "mod-b",
		Name:    "Mod B",
		Version: "1.0.0",
	})

	resolver := NewResolver(provider, nil)
	graph, err := resolver.Resolve("mod-a", "1.0.0")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// The graph should be created but have no incompatibles initially
	// (since mod-b wasn't actually resolved)
	if graph.HasErrors() {
		t.Errorf("Expected no errors (mod-b not installed)")
	}
}

func TestResolver_NotFound(t *testing.T) {
	provider := newMockProvider()

	provider.addMod(&ModInfo{
		ID:      "mod-a",
		Name:    "Mod A",
		Version: "1.0.0",
		Dependencies: []*Dependency{
			{ID: "mod-missing", VersionConstraint: ">=1.0.0", Type: Required},
		},
	})

	resolver := NewResolver(provider, nil)
	_, err := resolver.Resolve("mod-a", "1.0.0")
	if err == nil {
		t.Fatal("Expected not found error")
	}

	resErr, ok := err.(*ResolutionError)
	if !ok {
		t.Fatalf("Expected ResolutionError, got %T", err)
	}

	if resErr.Type != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %s", resErr.Type)
	}
}

func TestResolver_VersionSelection(t *testing.T) {
	provider := newMockProvider()

	provider.addMod(&ModInfo{
		ID:      "mod-a",
		Name:    "Mod A",
		Version: "1.0.0",
		Dependencies: []*Dependency{
			{ID: "mod-b", VersionConstraint: ">=1.0.0 <2.0.0", Type: Required},
		},
	})

	// Multiple versions of mod-b
	provider.addMod(&ModInfo{ID: "mod-b", Name: "Mod B", Version: "1.0.0"})
	provider.addMod(&ModInfo{ID: "mod-b", Name: "Mod B", Version: "1.5.0"})
	provider.addMod(&ModInfo{ID: "mod-b", Name: "Mod B", Version: "1.9.0"})
	provider.addMod(&ModInfo{ID: "mod-b", Name: "Mod B", Version: "2.0.0"})

	// Test latest strategy
	resolver := NewResolver(provider, &ResolutionOptions{
		Strategy:        StrategyLatest,
		IncludeOptional: true,
	})
	graph, err := resolver.Resolve("mod-a", "1.0.0")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if graph.Root.Dependencies[0].Version != "1.9.0" {
		t.Errorf("Expected version 1.9.0 (latest matching), got %s", graph.Root.Dependencies[0].Version)
	}

	// Test minimal strategy
	resolver = NewResolver(provider, &ResolutionOptions{
		Strategy:        StrategyMinimal,
		IncludeOptional: true,
	})
	// Clear cache for new resolution
	resolver.cache = newResolutionCache()
	graph, err = resolver.Resolve("mod-a", "1.0.0")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if graph.Root.Dependencies[0].Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0 (minimal matching), got %s", graph.Root.Dependencies[0].Version)
	}
}

func TestDependencyGraph_GenerateGraph(t *testing.T) {
	graph := &DependencyGraph{
		Root: &ResolvedDependency{
			ID:      "mod-a",
			Version: "1.0.0",
			Dependencies: []*ResolvedDependency{
				{
					ID:         "mod-b",
					Version:    "1.2.0",
					Type:       Required,
					IsOptional: false,
				},
				{
					ID:         "mod-c",
					Version:    "2.0.0",
					Type:       Optional,
					IsOptional: true,
				},
			},
		},
	}

	dot := graph.GenerateGraph()
	if dot == "" {
		t.Fatal("Expected non-empty DOT graph")
	}

	// Check for expected content
	if !contains(dot, "digraph dependencies") {
		t.Error("Expected 'digraph dependencies' in output")
	}
	if !contains(dot, "mod_a") {
		t.Error("Expected 'mod_a' node in output")
	}
	if !contains(dot, "mod_b") {
		t.Error("Expected 'mod_b' node in output")
	}
	if !contains(dot, "mod_c") {
		t.Error("Expected 'mod_c' node in output")
	}
}

func TestDependencyGraph_HasErrors(t *testing.T) {
	// Graph with no errors
	graph := &DependencyGraph{
		Root: &ResolvedDependency{ID: "mod-a", Version: "1.0.0"},
	}
	if graph.HasErrors() {
		t.Error("Expected no errors")
	}

	// Graph with conflict
	graph.Conflicts = []*VersionConflict{
		{ModID: "mod-x", RequiredBy: []string{"mod-a", "mod-b"}},
	}
	if !graph.HasErrors() {
		t.Error("Expected errors due to conflict")
	}

	// Graph with incompatibility
	graph.Conflicts = nil
	graph.Incompatibles = []*IncompatiblePair{
		{ModA: "mod-a", ModB: "mod-b"},
	}
	if !graph.HasErrors() {
		t.Error("Expected errors due to incompatibility")
	}
}

func TestValidateDependencies(t *testing.T) {
	provider := newMockProvider()
	resolver := NewResolver(provider, nil)

	// Test with incompatible constraints (>=2.0.0 and <1.5.0 cannot be satisfied)
	deps := []*Dependency{
		{ID: "mod-a", VersionConstraint: ">=2.0.0", Type: Required},
		{ID: "mod-a", VersionConstraint: "<1.5.0", Type: Required},
	}

	results, err := resolver.ValidateDependencies(deps)
	if err != nil {
		t.Fatalf("ValidateDependencies() error = %v", err)
	}

	// Should detect the conflicting constraints
	if len(results) == 0 {
		t.Error("Expected validation results for conflicting constraints")
	}

	// Test with incompatible mod pair
	deps = []*Dependency{
		{ID: "mod-a", VersionConstraint: ">=1.0.0", Type: Required},
		{ID: "mod-a", Type: Incompatible},
	}

	results, err = resolver.ValidateDependencies(deps)
	if err != nil {
		t.Fatalf("ValidateDependencies() error = %v", err)
	}

	// Should detect the incompatibility
	foundIncompat := false
	for _, r := range results {
		if r.Type == ValidationIncompatible {
			foundIncompat = true
			break
		}
	}
	if !foundIncompat {
		t.Error("Expected incompatibility result")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
