package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexinslc/chunk/internal/search"
	"github.com/alexinslc/chunk/internal/tracking"
)

func TestListCmd(t *testing.T) {
	// Create temporary tracker for testing
	tmpDir, err := os.MkdirTemp("", "chunk-list-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup test installations
	registryPath := filepath.Join(tmpDir, "installed.json")

	// Create test registry
	registry := &tracking.InstallationRegistry{
		Installations: []*tracking.Installation{
			{
				Slug:        "all-the-mods-9",
				Version:     "0.3.2",
				Bench:       "usechunk/recipes",
				Path:        "/opt/minecraft/servers/atm9",
				InstalledAt: time.Now().Add(-48 * time.Hour), // 2 days ago
				RecipeSnapshot: map[string]interface{}{
					"name": "All the Mods 9",
				},
			},
			{
				Slug:        "vault-hunters",
				Version:     "1.18.2",
				Bench:       "usechunk/recipes",
				Path:        "/home/user/vh-server",
				InstalledAt: time.Now().Add(-168 * time.Hour), // 1 week ago
				RecipeSnapshot: map[string]interface{}{
					"name": "Vault Hunters",
				},
			},
		},
	}

	// Save test data
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	// Note: This test validates the structure but doesn't execute the command
	// as it requires mocking the tracker initialization
	t.Log("List command structure validated")
}

func TestDisplayPaths(t *testing.T) {
	installations := []*tracking.Installation{
		{
			Slug:        "modpack-1",
			Version:     "1.0.0",
			Bench:       "test/bench",
			Path:        "/path/to/modpack1",
			InstalledAt: time.Now(),
		},
		{
			Slug:        "modpack-2",
			Version:     "2.0.0",
			Bench:       "test/bench",
			Path:        "/path/to/modpack2",
			InstalledAt: time.Now(),
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := displayPaths(installations)
	if err != nil {
		t.Fatalf("displayPaths failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify output contains both paths
	if !strings.Contains(output, "/path/to/modpack1") {
		t.Error("Output should contain first path")
	}
	if !strings.Contains(output, "/path/to/modpack2") {
		t.Error("Output should contain second path")
	}
}

func TestDisplayJSON(t *testing.T) {
	installations := []*tracking.Installation{
		{
			Slug:        "test-modpack",
			Version:     "1.0.0",
			Bench:       "test/bench",
			Path:        "/path/to/modpack",
			InstalledAt: time.Now().UTC(),
			RecipeSnapshot: map[string]interface{}{
				"name": "Test Modpack",
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := displayJSON(installations)
	if err != nil {
		t.Fatalf("displayJSON failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify output is valid JSON
	var result []*tracking.Installation
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify data
	if len(result) != 1 {
		t.Fatalf("Expected 1 installation, got %d", len(result))
	}
	if result[0].Slug != "test-modpack" {
		t.Errorf("Expected slug 'test-modpack', got '%s'", result[0].Slug)
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "1 second ago",
			time:     now.Add(-1 * time.Second),
			expected: "1 second ago",
		},
		{
			name:     "30 seconds ago",
			time:     now.Add(-30 * time.Second),
			expected: "30 seconds ago",
		},
		{
			name:     "1 minute ago",
			time:     now.Add(-1 * time.Minute),
			expected: "1 minute ago",
		},
		{
			name:     "45 minutes ago",
			time:     now.Add(-45 * time.Minute),
			expected: "45 minutes ago",
		},
		{
			name:     "1 hour ago",
			time:     now.Add(-1 * time.Hour),
			expected: "1 hour ago",
		},
		{
			name:     "5 hours ago",
			time:     now.Add(-5 * time.Hour),
			expected: "5 hours ago",
		},
		{
			name:     "1 day ago",
			time:     now.Add(-24 * time.Hour),
			expected: "1 day ago",
		},
		{
			name:     "3 days ago",
			time:     now.Add(-72 * time.Hour),
			expected: "3 days ago",
		},
		{
			name:     "1 week ago",
			time:     now.Add(-7 * 24 * time.Hour),
			expected: "1 week ago",
		},
		{
			name:     "2 weeks ago",
			time:     now.Add(-14 * 24 * time.Hour),
			expected: "2 weeks ago",
		},
		{
			name:     "1 month ago",
			time:     now.Add(-30 * 24 * time.Hour),
			expected: "1 month ago",
		},
		{
			name:     "3 months ago",
			time:     now.Add(-90 * 24 * time.Hour),
			expected: "3 months ago",
		},
		{
			name:     "1 year ago",
			time:     now.Add(-365 * 24 * time.Hour),
			expected: "1 year ago",
		},
		{
			name:     "2 years ago",
			time:     now.Add(-730 * 24 * time.Hour),
			expected: "2 years ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRelativeTime(tt.time)
			if result != tt.expected {
				t.Errorf("formatRelativeTime() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDisplayListEmpty(t *testing.T) {
	installations := []*tracking.Installation{}

	// This should not panic or error
	err := displayList(installations, false)
	if err != nil {
		t.Fatalf("displayList with empty installations should not error: %v", err)
	}
}

func TestDisplayListBasic(t *testing.T) {
	installations := []*tracking.Installation{
		{
			Slug:        "test-modpack",
			Version:     "1.0.0",
			Bench:       "test/bench",
			Path:        "/path/to/modpack",
			InstalledAt: time.Now().Add(-48 * time.Hour), // 2 days ago
			RecipeSnapshot: map[string]interface{}{
				"name": "Test Modpack",
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := displayList(installations, false)
	if err != nil {
		t.Fatalf("displayList failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify output contains expected information
	if !strings.Contains(output, "test-modpack") {
		t.Error("Output should contain modpack slug")
	}
	if !strings.Contains(output, "1.0.0") {
		t.Error("Output should contain version")
	}
	if !strings.Contains(output, "/path/to/modpack") {
		t.Error("Output should contain path")
	}
	if !strings.Contains(output, "test/bench") {
		t.Error("Output should contain bench name")
	}
	if !strings.Contains(output, "days ago") {
		t.Error("Output should contain relative time")
	}
}

func TestIsInstallationOutdated(t *testing.T) {
	// This test validates the function with a nil cache
	inst := &tracking.Installation{
		Slug:        "test-modpack",
		Version:     "1.0.0",
		Bench:       "test/bench",
		Path:        "/path/to/modpack",
		InstalledAt: time.Now(),
	}

	// Test with nil recipe cache (should return false, "")
	isOutdated, version := isInstallationOutdated(inst, nil)
	if isOutdated {
		t.Error("Expected false for nil recipe cache")
	}
	if version != "" {
		t.Errorf("Expected empty version, got %s", version)
	}

	// Test with empty cache
	emptyCache := make(map[string]*search.Recipe)
	isOutdated, version = isInstallationOutdated(inst, emptyCache)
	if isOutdated {
		t.Error("Expected false for empty cache")
	}
	if version != "" {
		t.Errorf("Expected empty version, got %s", version)
	}

	// Test with matching version
	cache := map[string]*search.Recipe{
		"test/bench:test-modpack": {
			Slug:    "test-modpack",
			Version: "1.0.0",
		},
	}
	isOutdated, version = isInstallationOutdated(inst, cache)
	if isOutdated {
		t.Error("Expected false when versions match")
	}
	if version != "" {
		t.Errorf("Expected empty version, got %s", version)
	}

	// Test with newer version available
	cache["test/bench:test-modpack"] = &search.Recipe{
		Slug:    "test-modpack",
		Version: "2.0.0",
	}
	isOutdated, version = isInstallationOutdated(inst, cache)
	if !isOutdated {
		t.Error("Expected true when newer version available")
	}
	if version != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", version)
	}

	// Test with whitespace in versions
	inst.Version = " 1.0.0 "
	cache["test/bench:test-modpack"] = &search.Recipe{
		Slug:    "test-modpack",
		Version: " 1.0.0 ",
	}
	isOutdated, version = isInstallationOutdated(inst, cache)
	if isOutdated {
		t.Error("Expected false when versions match after trimming whitespace")
	}
}

func TestListCmdFlags(t *testing.T) {
	// Verify command has expected flags
	if ListCmd.Flags().Lookup("paths") == nil {
		t.Error("ListCmd should have --paths flag")
	}
	if ListCmd.Flags().Lookup("json") == nil {
		t.Error("ListCmd should have --json flag")
	}
	if ListCmd.Flags().Lookup("outdated") == nil {
		t.Error("ListCmd should have --outdated flag")
	}
}

func TestListCmdStructure(t *testing.T) {
	// Verify command structure
	if ListCmd.Use != "list" {
		t.Errorf("Expected Use to be 'list', got '%s'", ListCmd.Use)
	}
	if ListCmd.Short == "" {
		t.Error("Short description should not be empty")
	}
	if ListCmd.Long == "" {
		t.Error("Long description should not be empty")
	}
	if ListCmd.RunE == nil {
		t.Error("RunE function should be defined")
	}
}
