package tracking

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewTracker(t *testing.T) {
	tracker, err := NewTracker()
	if err != nil {
		t.Fatalf("NewTracker failed: %v", err)
	}

	if tracker == nil {
		t.Fatal("Expected tracker to be non-nil")
	}

	if tracker.registryPath == "" {
		t.Error("Expected registryPath to be set")
	}

	// Verify path contains .chunk directory
	if !filepath.IsAbs(tracker.registryPath) {
		t.Error("Expected registryPath to be absolute")
	}
}

func TestTrackerLoadEmpty(t *testing.T) {
	// Create temporary tracker with custom path
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	registry, err := tracker.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if registry == nil {
		t.Fatal("Expected registry to be non-nil")
	}

	if registry.Installations == nil {
		t.Error("Expected Installations to be initialized")
	}

	if len(registry.Installations) != 0 {
		t.Errorf("Expected empty installations, got %d", len(registry.Installations))
	}
}

func TestTrackerSaveAndLoad(t *testing.T) {
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	// Create test registry
	registry := &InstallationRegistry{
		Installations: []*Installation{
			{
				Slug:        "test-modpack",
				Version:     "1.0.0",
				Bench:       "usechunk/recipes",
				Path:        "/opt/minecraft/test",
				InstalledAt: time.Now().UTC(),
				RecipeSnapshot: map[string]interface{}{
					"name": "Test Modpack",
					"mc_version": "1.20.1",
				},
			},
		},
	}

	// Save registry
	if err := tracker.Save(registry); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load registry
	loaded, err := tracker.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Installations) != 1 {
		t.Fatalf("Expected 1 installation, got %d", len(loaded.Installations))
	}

	inst := loaded.Installations[0]
	if inst.Slug != "test-modpack" {
		t.Errorf("Expected slug 'test-modpack', got '%s'", inst.Slug)
	}
	if inst.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", inst.Version)
	}
	if inst.Bench != "usechunk/recipes" {
		t.Errorf("Expected bench 'usechunk/recipes', got '%s'", inst.Bench)
	}
	if inst.Path != "/opt/minecraft/test" {
		t.Errorf("Expected path '/opt/minecraft/test', got '%s'", inst.Path)
	}
}

func TestTrackerAddInstallation(t *testing.T) {
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	installation := &Installation{
		Slug:        "atm9",
		Version:     "0.3.2",
		Bench:       "usechunk/recipes",
		Path:        "/opt/minecraft/atm9",
		InstalledAt: time.Now().UTC(),
		RecipeSnapshot: map[string]interface{}{
			"name": "All the Mods 9",
		},
	}

	// Add installation
	if err := tracker.AddInstallation(installation); err != nil {
		t.Fatalf("AddInstallation failed: %v", err)
	}

	// Verify it was added
	loaded, err := tracker.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Installations) != 1 {
		t.Fatalf("Expected 1 installation, got %d", len(loaded.Installations))
	}

	if loaded.Installations[0].Slug != "atm9" {
		t.Errorf("Expected slug 'atm9', got '%s'", loaded.Installations[0].Slug)
	}
}

func TestTrackerAddInstallationDuplicatePath(t *testing.T) {
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	path := "/opt/minecraft/test"

	// Add first installation
	installation1 := &Installation{
		Slug:        "modpack-v1",
		Version:     "1.0.0",
		Bench:       "usechunk/recipes",
		Path:        path,
		InstalledAt: time.Now().UTC(),
		RecipeSnapshot: map[string]interface{}{
			"name": "Modpack v1",
		},
	}

	if err := tracker.AddInstallation(installation1); err != nil {
		t.Fatalf("AddInstallation failed: %v", err)
	}

	// Add second installation at same path (should update)
	installation2 := &Installation{
		Slug:        "modpack-v2",
		Version:     "2.0.0",
		Bench:       "usechunk/recipes",
		Path:        path,
		InstalledAt: time.Now().UTC(),
		RecipeSnapshot: map[string]interface{}{
			"name": "Modpack v2",
		},
	}

	if err := tracker.AddInstallation(installation2); err != nil {
		t.Fatalf("AddInstallation failed: %v", err)
	}

	// Verify only one installation exists and it's the updated one
	loaded, err := tracker.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Installations) != 1 {
		t.Fatalf("Expected 1 installation, got %d", len(loaded.Installations))
	}

	if loaded.Installations[0].Slug != "modpack-v2" {
		t.Errorf("Expected slug 'modpack-v2', got '%s'", loaded.Installations[0].Slug)
	}
	if loaded.Installations[0].Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", loaded.Installations[0].Version)
	}
}

func TestTrackerRemoveInstallation(t *testing.T) {
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	// Add installation
	installation := &Installation{
		Slug:        "test-modpack",
		Version:     "1.0.0",
		Bench:       "usechunk/recipes",
		Path:        "/opt/minecraft/test",
		InstalledAt: time.Now().UTC(),
		RecipeSnapshot: map[string]interface{}{},
	}

	if err := tracker.AddInstallation(installation); err != nil {
		t.Fatalf("AddInstallation failed: %v", err)
	}

	// Remove installation
	if err := tracker.RemoveInstallation("/opt/minecraft/test"); err != nil {
		t.Fatalf("RemoveInstallation failed: %v", err)
	}

	// Verify it was removed
	loaded, err := tracker.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Installations) != 0 {
		t.Errorf("Expected 0 installations, got %d", len(loaded.Installations))
	}
}

func TestTrackerRemoveInstallationNotFound(t *testing.T) {
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	// Try to remove non-existent installation (should not error)
	if err := tracker.RemoveInstallation("/non/existent/path"); err != nil {
		t.Errorf("RemoveInstallation should not error for non-existent path: %v", err)
	}
}

func TestTrackerGetInstallation(t *testing.T) {
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	// Add installation
	installation := &Installation{
		Slug:        "test-modpack",
		Version:     "1.0.0",
		Bench:       "usechunk/recipes",
		Path:        "/opt/minecraft/test",
		InstalledAt: time.Now().UTC(),
		RecipeSnapshot: map[string]interface{}{},
	}

	if err := tracker.AddInstallation(installation); err != nil {
		t.Fatalf("AddInstallation failed: %v", err)
	}

	// Get installation
	found, err := tracker.GetInstallation("/opt/minecraft/test")
	if err != nil {
		t.Fatalf("GetInstallation failed: %v", err)
	}

	if found == nil {
		t.Fatal("Expected installation to be found")
	}

	if found.Slug != "test-modpack" {
		t.Errorf("Expected slug 'test-modpack', got '%s'", found.Slug)
	}
}

func TestTrackerGetInstallationNotFound(t *testing.T) {
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	// Get non-existent installation
	found, err := tracker.GetInstallation("/non/existent/path")
	if err != nil {
		t.Fatalf("GetInstallation failed: %v", err)
	}

	if found != nil {
		t.Error("Expected nil for non-existent installation")
	}
}

func TestTrackerListInstallations(t *testing.T) {
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	// Add multiple installations
	installations := []*Installation{
		{
			Slug:        "modpack-1",
			Version:     "1.0.0",
			Bench:       "usechunk/recipes",
			Path:        "/opt/minecraft/pack1",
			InstalledAt: time.Now().UTC(),
			RecipeSnapshot: map[string]interface{}{},
		},
		{
			Slug:        "modpack-2",
			Version:     "2.0.0",
			Bench:       "usechunk/recipes",
			Path:        "/opt/minecraft/pack2",
			InstalledAt: time.Now().UTC(),
			RecipeSnapshot: map[string]interface{}{},
		},
	}

	for _, inst := range installations {
		if err := tracker.AddInstallation(inst); err != nil {
			t.Fatalf("AddInstallation failed: %v", err)
		}
	}

	// List installations
	list, err := tracker.ListInstallations()
	if err != nil {
		t.Fatalf("ListInstallations failed: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("Expected 2 installations, got %d", len(list))
	}
}

func TestTrackerUpdateInstallation(t *testing.T) {
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	// Add installation
	installation := &Installation{
		Slug:        "test-modpack",
		Version:     "1.0.0",
		Bench:       "usechunk/recipes",
		Path:        "/opt/minecraft/test",
		InstalledAt: time.Now().UTC(),
		RecipeSnapshot: map[string]interface{}{},
	}

	if err := tracker.AddInstallation(installation); err != nil {
		t.Fatalf("AddInstallation failed: %v", err)
	}

	// Update installation
	installation.Version = "2.0.0"
	installation.RecipeSnapshot = map[string]interface{}{
		"updated": true,
	}

	if err := tracker.UpdateInstallation(installation); err != nil {
		t.Fatalf("UpdateInstallation failed: %v", err)
	}

	// Verify update
	found, err := tracker.GetInstallation("/opt/minecraft/test")
	if err != nil {
		t.Fatalf("GetInstallation failed: %v", err)
	}

	if found.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", found.Version)
	}
}

func TestTrackerUpdateInstallationNotFound(t *testing.T) {
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	// Try to update non-existent installation
	installation := &Installation{
		Slug:        "test-modpack",
		Version:     "1.0.0",
		Bench:       "usechunk/recipes",
		Path:        "/non/existent/path",
		InstalledAt: time.Now().UTC(),
		RecipeSnapshot: map[string]interface{}{},
	}

	err := tracker.UpdateInstallation(installation)
	if err == nil {
		t.Error("Expected error when updating non-existent installation")
	}
}

func TestValidateInstallation(t *testing.T) {
	tests := []struct {
		name          string
		installation  *Installation
		expectError   bool
		errorContains string
	}{
		{
			name: "valid installation",
			installation: &Installation{
				Slug:        "test",
				Version:     "1.0.0",
				Path:        "/path",
				InstalledAt: time.Now(),
				RecipeSnapshot: map[string]interface{}{},
			},
			expectError: false,
		},
		{
			name: "missing slug",
			installation: &Installation{
				Version:     "1.0.0",
				Path:        "/path",
				InstalledAt: time.Now(),
				RecipeSnapshot: map[string]interface{}{},
			},
			expectError:   true,
			errorContains: "slug",
		},
		{
			name: "missing version",
			installation: &Installation{
				Slug:        "test",
				Path:        "/path",
				InstalledAt: time.Now(),
				RecipeSnapshot: map[string]interface{}{},
			},
			expectError:   true,
			errorContains: "version",
		},
		{
			name: "missing path",
			installation: &Installation{
				Slug:        "test",
				Version:     "1.0.0",
				InstalledAt: time.Now(),
				RecipeSnapshot: map[string]interface{}{},
			},
			expectError:   true,
			errorContains: "path",
		},
		{
			name: "missing installed_at",
			installation: &Installation{
				Slug:        "test",
				Version:     "1.0.0",
				Path:        "/path",
				RecipeSnapshot: map[string]interface{}{},
			},
			expectError:   true,
			errorContains: "installed_at",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInstallation(tt.installation)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestTrackerSaveNilRegistry(t *testing.T) {
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	err := tracker.Save(nil)
	if err == nil {
		t.Error("Expected error when saving nil registry")
	}
}

func TestTrackerAddNilInstallation(t *testing.T) {
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	err := tracker.AddInstallation(nil)
	if err == nil {
		t.Error("Expected error when adding nil installation")
	}
}

func TestTrackerMultipleSameRecipeDifferentPaths(t *testing.T) {
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	// Add same modpack at two different paths
	installation1 := &Installation{
		Slug:        "atm9",
		Version:     "0.3.2",
		Bench:       "usechunk/recipes",
		Path:        "/opt/minecraft/atm9-server1",
		InstalledAt: time.Now().UTC(),
		RecipeSnapshot: map[string]interface{}{},
	}

	installation2 := &Installation{
		Slug:        "atm9",
		Version:     "0.3.2",
		Bench:       "usechunk/recipes",
		Path:        "/opt/minecraft/atm9-server2",
		InstalledAt: time.Now().UTC(),
		RecipeSnapshot: map[string]interface{}{},
	}

	if err := tracker.AddInstallation(installation1); err != nil {
		t.Fatalf("AddInstallation failed: %v", err)
	}

	if err := tracker.AddInstallation(installation2); err != nil {
		t.Fatalf("AddInstallation failed: %v", err)
	}

	// Verify both installations exist
	list, err := tracker.ListInstallations()
	if err != nil {
		t.Fatalf("ListInstallations failed: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("Expected 2 installations, got %d", len(list))
	}

	// Verify paths are different
	paths := make(map[string]bool)
	for _, inst := range list {
		paths[inst.Path] = true
	}

	if len(paths) != 2 {
		t.Error("Expected 2 different paths")
	}
}

func TestTrackerJSONFormat(t *testing.T) {
	tracker := createTestTracker(t)
	defer cleanupTestTracker(t, tracker)

	// Add installation
	installation := &Installation{
		Slug:        "all-the-mods-9",
		Version:     "0.3.2",
		Bench:       "usechunk/recipes",
		Path:        "/opt/minecraft/servers/atm9",
		InstalledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		RecipeSnapshot: map[string]interface{}{
			"name":       "All the Mods 9",
			"mc_version": "1.20.1",
		},
	}

	if err := tracker.AddInstallation(installation); err != nil {
		t.Fatalf("AddInstallation failed: %v", err)
	}

	// Read raw JSON
	data, err := os.ReadFile(tracker.registryPath)
	if err != nil {
		t.Fatalf("Failed to read registry file: %v", err)
	}

	// Parse JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify structure
	installations, ok := raw["installations"].([]interface{})
	if !ok {
		t.Fatal("Expected 'installations' array in JSON")
	}

	if len(installations) != 1 {
		t.Fatalf("Expected 1 installation in JSON, got %d", len(installations))
	}

	inst := installations[0].(map[string]interface{})
	if inst["slug"] != "all-the-mods-9" {
		t.Errorf("Expected slug 'all-the-mods-9', got '%v'", inst["slug"])
	}
	if inst["version"] != "0.3.2" {
		t.Errorf("Expected version '0.3.2', got '%v'", inst["version"])
	}
	if inst["bench"] != "usechunk/recipes" {
		t.Errorf("Expected bench 'usechunk/recipes', got '%v'", inst["bench"])
	}
	if inst["path"] != "/opt/minecraft/servers/atm9" {
		t.Errorf("Expected path '/opt/minecraft/servers/atm9', got '%v'", inst["path"])
	}
}

// Helper functions

func createTestTracker(t *testing.T) *Tracker {
	tmpDir, err := os.MkdirTemp("", "chunk-tracking-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	return &Tracker{
		registryPath: filepath.Join(tmpDir, "installed.json"),
	}
}

func cleanupTestTracker(t *testing.T, tracker *Tracker) {
	dir := filepath.Dir(tracker.registryPath)
	if err := os.RemoveAll(dir); err != nil {
		t.Logf("Failed to cleanup test directory: %v", err)
	}
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

func containsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}
