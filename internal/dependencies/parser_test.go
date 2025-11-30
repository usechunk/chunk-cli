package dependencies

import (
	"testing"
)

func TestParseFabricModJSON(t *testing.T) {
	jsonData := []byte(`{
		"id": "test-mod",
		"version": "1.0.0",
		"name": "Test Mod",
		"description": "A test mod",
		"depends": {
			"fabricloader": ">=0.14.0",
			"minecraft": "~1.20",
			"fabric-api": ">=0.80.0"
		},
		"recommends": {
			"modmenu": "*"
		},
		"provides": ["api-compat"]
	}`)

	mod, err := ParseFabricModJSON(jsonData)
	if err != nil {
		t.Fatalf("ParseFabricModJSON failed: %v", err)
	}

	if mod.ID != "test-mod" {
		t.Errorf("Expected ID 'test-mod', got %q", mod.ID)
	}

	if mod.Version != "1.0.0" {
		t.Errorf("Expected Version '1.0.0', got %q", mod.Version)
	}

	// Should only have fabric-api as dependency (fabricloader and minecraft are skipped)
	requiredDeps := 0
	for _, dep := range mod.Dependencies {
		if dep.Required {
			requiredDeps++
		}
	}
	if requiredDeps != 1 {
		t.Errorf("Expected 1 required dependency (fabric-api), got %d", requiredDeps)
	}

	// Check provides
	if len(mod.Provides) != 1 || mod.Provides[0] != "api-compat" {
		t.Errorf("Expected provides ['api-compat'], got %v", mod.Provides)
	}
}

func TestParseModrinthDependencies(t *testing.T) {
	jsonData := []byte(`{
		"id": "version-id-123",
		"project_id": "mod-project-id",
		"name": "Test Mod v1.0.0",
		"version_number": "1.0.0",
		"game_versions": ["1.20.1"],
		"loaders": ["fabric"],
		"dependencies": [
			{
				"project_id": "fabric-api",
				"dependency_type": "required"
			},
			{
				"project_id": "optional-mod",
				"dependency_type": "optional"
			}
		]
	}`)

	mod, err := ParseModrinthDependencies(jsonData)
	if err != nil {
		t.Fatalf("ParseModrinthDependencies failed: %v", err)
	}

	if mod.ID != "mod-project-id" {
		t.Errorf("Expected ID 'mod-project-id', got %q", mod.ID)
	}

	if mod.Version != "1.0.0" {
		t.Errorf("Expected Version '1.0.0', got %q", mod.Version)
	}

	if len(mod.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(mod.Dependencies))
	}

	// Check required dependency
	foundRequired := false
	for _, dep := range mod.Dependencies {
		if dep.ModID == "fabric-api" && dep.Required {
			foundRequired = true
		}
	}
	if !foundRequired {
		t.Error("Expected fabric-api to be a required dependency")
	}
}

func TestNormalizeFabricVersion(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"*", "*"},
		{">=1.0.0", ">=1.0.0"},
		{">1.0.0", ">1.0.0"},
		{"<=2.0.0", "<=2.0.0"},
		{"<2.0.0", "<2.0.0"},
		{">=1.0.0 <2.0.0", ">=1.0.0,<2.0.0"},
		{"1.0.0", "1.0.0"},
	}

	for _, tt := range tests {
		got := normalizeFabricVersion(tt.input)
		if got != tt.expect {
			t.Errorf("normalizeFabricVersion(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestNormalizeForgeVersion(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"", "*"},
		{"1.0.0", "1.0.0"},
	}

	for _, tt := range tests {
		got := normalizeForgeVersion(tt.input)
		if got != tt.expect {
			t.Errorf("normalizeForgeVersion(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestParseMavenVersionRange(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"[1.0,2.0]", ">=1.0,<=2.0"},
		{"[1.0,2.0)", ">=1.0,<2.0"},
		{"(1.0,2.0]", ">1.0,<=2.0"},
		{"(1.0,2.0)", ">1.0,<2.0"},
		{"[1.0,)", ">=1.0"},
		{"(,2.0]", "<=2.0"},
		{"[1.0]", "1.0"},
		{"", "*"},
	}

	for _, tt := range tests {
		got := parseMavenVersionRange(tt.input)
		if got != tt.expect {
			t.Errorf("parseMavenVersionRange(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestExtractModIDFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		expect   string
	}{
		{"jei-1.20.1-15.2.0.27.jar", "jei-1.20.1"},
		{"create-0.5.1.jar", "create"},
		{"sodium-fabric-0.5.8+mc1.20.4.jar", "sodium-fabric"},
		{"mod_1.0.0.jar", "mod"},
		{"simple-mod.jar", "simple-mod"},
	}

	for _, tt := range tests {
		got := ExtractModIDFromFilename(tt.filename)
		if got != tt.expect {
			t.Errorf("ExtractModIDFromFilename(%q) = %q, want %q", tt.filename, got, tt.expect)
		}
	}
}

func TestLooksLikeVersion(t *testing.T) {
	tests := []struct {
		input  string
		expect bool
	}{
		{"1.0.0", true},
		{"0.5.1", true},
		{"v1.0.0", true},
		{"V2.0.0", true},
		{"mc1.20.1", true},
		{"MC1.19", true},
		{"beta", false},
		{"release", false},
		{"", false},
	}

	for _, tt := range tests {
		got := looksLikeVersion(tt.input)
		if got != tt.expect {
			t.Errorf("looksLikeVersion(%q) = %v, want %v", tt.input, got, tt.expect)
		}
	}
}

func TestParseForgeModDependencies(t *testing.T) {
	deps := []struct {
		ModID        string
		Mandatory    bool
		VersionRange string
		Ordering     string
		Side         string
	}{
		{ModID: "forge", Mandatory: true, VersionRange: "[47.2.0,)", Side: "BOTH"},
		{ModID: "minecraft", Mandatory: true, VersionRange: "[1.20.1,1.21)", Side: "BOTH"},
		{ModID: "jei", Mandatory: true, VersionRange: "[15.0.0,16.0.0)", Side: "BOTH"},
		{ModID: "optional-mod", Mandatory: false, VersionRange: "*", Side: "CLIENT"},
	}

	result := ParseForgeModDependencies("test-mod", deps)

	// Should only have jei and optional-mod (forge and minecraft are skipped)
	if len(result) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(result))
	}

	// Check jei dependency
	foundJei := false
	for _, dep := range result {
		if dep.ModID == "jei" {
			foundJei = true
			if !dep.Required {
				t.Error("jei should be required")
			}
		}
	}
	if !foundJei {
		t.Error("Expected jei dependency")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{invalid json}`)

	_, err := ParseFabricModJSON(invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	_, err = ParseModrinthDependencies(invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
