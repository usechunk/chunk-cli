package dependencies

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Version
		wantErr bool
	}{
		{
			name:  "simple version",
			input: "1.0.0",
			want:  &Version{Major: 1, Minor: 0, Patch: 0, Raw: "1.0.0"},
		},
		{
			name:  "version with prefix",
			input: "v1.2.3",
			want:  &Version{Major: 1, Minor: 2, Patch: 3, Raw: "v1.2.3"},
		},
		{
			name:  "version with prerelease",
			input: "1.0.0-beta",
			want:  &Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "-beta", Raw: "1.0.0-beta"},
		},
		{
			name:  "two part version",
			input: "1.20",
			want:  &Version{Major: 1, Minor: 20, Patch: 0, Raw: "1.20"},
		},
		{
			name:    "empty version",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid major",
			input:   "abc.0.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
				t.Errorf("ParseVersion() = %v, want %v", got, tt.want)
			}
			if got.Prerelease != tt.want.Prerelease {
				t.Errorf("ParseVersion() prerelease = %v, want %v", got.Prerelease, tt.want.Prerelease)
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		name   string
		v1     string
		v2     string
		expect int
	}{
		{"equal", "1.0.0", "1.0.0", 0},
		{"major greater", "2.0.0", "1.0.0", 1},
		{"major less", "1.0.0", "2.0.0", -1},
		{"minor greater", "1.2.0", "1.1.0", 1},
		{"minor less", "1.1.0", "1.2.0", -1},
		{"patch greater", "1.0.2", "1.0.1", 1},
		{"patch less", "1.0.1", "1.0.2", -1},
		{"prerelease vs stable", "1.0.0-beta", "1.0.0", -1},
		{"stable vs prerelease", "1.0.0", "1.0.0-alpha", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1, _ := ParseVersion(tt.v1)
			v2, _ := ParseVersion(tt.v2)
			got := v1.Compare(v2)
			if got != tt.expect {
				t.Errorf("(%s).Compare(%s) = %d, want %d", tt.v1, tt.v2, got, tt.expect)
			}
		})
	}
}

func TestParseVersionRange(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"any version", "*", false},
		{"empty", "", false},
		{"exact version", "1.0.0", false},
		{"greater or equal", ">=1.0.0", false},
		{"greater than", ">1.0.0", false},
		{"less or equal", "<=2.0.0", false},
		{"less than", "<2.0.0", false},
		{"range", "1.0.0-2.0.0", false},
		{"combined", ">=1.0.0,<2.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseVersionRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersionRange(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestVersionRangeMatches(t *testing.T) {
	tests := []struct {
		name    string
		range_  string
		version string
		expect  bool
	}{
		{"any matches all", "*", "1.0.0", true},
		{"exact match", "1.0.0", "1.0.0", true},
		{"exact no match", "1.0.0", "1.0.1", false},
		{"gte match equal", ">=1.0.0", "1.0.0", true},
		{"gte match greater", ">=1.0.0", "1.1.0", true},
		{"gte no match", ">=1.0.0", "0.9.0", false},
		{"gt match", ">1.0.0", "1.0.1", true},
		{"gt no match equal", ">1.0.0", "1.0.0", false},
		{"lte match equal", "<=2.0.0", "2.0.0", true},
		{"lte match less", "<=2.0.0", "1.9.0", true},
		{"lte no match", "<=2.0.0", "2.0.1", false},
		{"lt match", "<2.0.0", "1.9.9", true},
		{"lt no match equal", "<2.0.0", "2.0.0", false},
		{"range match", "1.0.0-2.0.0", "1.5.0", true},
		{"range match min", "1.0.0-2.0.0", "1.0.0", true},
		{"range match max", "1.0.0-2.0.0", "2.0.0", true},
		{"range no match below", "1.0.0-2.0.0", "0.9.0", false},
		{"range no match above", "1.0.0-2.0.0", "2.1.0", false},
		{"combined match", ">=1.0.0,<2.0.0", "1.5.0", true},
		{"combined no match", ">=1.0.0,<2.0.0", "2.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vr, err := ParseVersionRange(tt.range_)
			if err != nil {
				t.Fatalf("ParseVersionRange(%q) failed: %v", tt.range_, err)
			}
			v, err := ParseVersion(tt.version)
			if err != nil {
				t.Fatalf("ParseVersion(%q) failed: %v", tt.version, err)
			}
			got := vr.Matches(v)
			if got != tt.expect {
				t.Errorf("VersionRange(%q).Matches(%q) = %v, want %v", tt.range_, tt.version, got, tt.expect)
			}
		})
	}
}

func TestDependencyTypes(t *testing.T) {
	dep := Dependency{
		ModID:       "jei",
		VersionExpr: ">=1.0.0",
		Required:    true,
		Side:        "both",
	}

	if dep.ModID != "jei" {
		t.Errorf("Expected ModID 'jei', got %q", dep.ModID)
	}
	if dep.VersionExpr != ">=1.0.0" {
		t.Errorf("Expected VersionExpr '>=1.0.0', got %q", dep.VersionExpr)
	}
	if !dep.Required {
		t.Error("Expected Required to be true")
	}
	if dep.Side != "both" {
		t.Errorf("Expected Side 'both', got %q", dep.Side)
	}
}

func TestModInfo(t *testing.T) {
	mod := &ModInfo{
		ID:      "test-mod",
		Name:    "Test Mod",
		Version: "1.0.0",
		Dependencies: []Dependency{
			{ModID: "dep1", VersionExpr: ">=1.0.0", Required: true},
			{ModID: "dep2", VersionExpr: "*", Required: false},
		},
		Provides: []string{"api-compat"},
	}

	if mod.ID != "test-mod" {
		t.Errorf("Expected ID 'test-mod', got %q", mod.ID)
	}
	if len(mod.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(mod.Dependencies))
	}
	if len(mod.Provides) != 1 {
		t.Errorf("Expected 1 provides, got %d", len(mod.Provides))
	}
}

func TestConflict(t *testing.T) {
	conflict := Conflict{
		ModID:      "conflicting-mod",
		RequiredBy: []string{"mod-a", "mod-b"},
		Versions:   []string{"1.0.0", "2.0.0"},
	}

	if conflict.ModID != "conflicting-mod" {
		t.Errorf("Expected ModID 'conflicting-mod', got %q", conflict.ModID)
	}
	if len(conflict.RequiredBy) != 2 {
		t.Errorf("Expected 2 RequiredBy entries, got %d", len(conflict.RequiredBy))
	}
	if len(conflict.Versions) != 2 {
		t.Errorf("Expected 2 Versions, got %d", len(conflict.Versions))
	}
}

func TestResolutionResult(t *testing.T) {
	result := &ResolutionResult{
		ResolvedMods: []*ResolvedMod{
			{ModInfo: &ModInfo{ID: "mod1", Version: "1.0.0"}},
			{ModInfo: &ModInfo{ID: "mod2", Version: "1.0.0"}},
		},
		InstallOrder: []string{"mod1", "mod2"},
		Conflicts:    []Conflict{},
		MissingDeps:  []Dependency{},
	}

	if len(result.ResolvedMods) != 2 {
		t.Errorf("Expected 2 resolved mods, got %d", len(result.ResolvedMods))
	}
	if len(result.InstallOrder) != 2 {
		t.Errorf("Expected install order with 2 items, got %d", len(result.InstallOrder))
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		version *Version
		expect  string
	}{
		{&Version{Major: 1, Minor: 2, Patch: 3, Raw: "1.2.3"}, "1.2.3"},
		{&Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "-beta"}, "1.0.0-beta"},
	}

	for _, tt := range tests {
		got := tt.version.String()
		if got != tt.expect {
			t.Errorf("Version.String() = %q, want %q", got, tt.expect)
		}
	}
}

func TestVersionRangeString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"any", "*", "*"},
		{"exact", "1.0.0", "1.0.0"},
		{"gte", ">=1.0.0", ">=1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vr, _ := ParseVersionRange(tt.input)
			got := vr.String()
			if got != tt.expect {
				t.Errorf("VersionRange.String() = %q, want %q", got, tt.expect)
			}
		})
	}
}
