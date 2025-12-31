package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexinslc/chunk/internal/search"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "My Modpack",
			expected: "my-modpack",
		},
		{
			name:     "name with special characters",
			input:    "All The Mods 9!",
			expected: "all-the-mods-9",
		},
		{
			name:     "name with multiple spaces",
			input:    "Cool   Modpack   Name",
			expected: "cool-modpack-name",
		},
		{
			name:     "name with underscores",
			input:    "my_cool_modpack",
			expected: "my-cool-modpack",
		},
		{
			name:     "name with leading/trailing spaces",
			input:    "  Test Modpack  ",
			expected: "test-modpack",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.input)
			if result != tt.expected {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSaveRecipe(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	recipe := &search.Recipe{
		Name:             "Test Modpack",
		Slug:             "test-modpack",
		Description:      "A test modpack",
		MCVersion:        "1.20.1",
		Loader:           "forge",
		LoaderVersion:    "47.3.0",
		DownloadURL:      "https://example.com/modpack.zip",
		SHA256:           "abc123def456",
		RecommendedRAMGB: 6,
		DiskSpaceGB:      8,
		License:          "MIT",
	}

	outputPath := filepath.Join(tmpDir, "test-modpack.json")
	err := saveRecipe(recipe, outputPath)
	if err != nil {
		t.Fatalf("saveRecipe() failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("recipe file was not created: %s", outputPath)
	}

	// Verify content can be read back
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read recipe file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("recipe file is empty")
	}
}

func TestSaveRecipeCreatesDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	recipe := &search.Recipe{
		Name:        "Test Modpack",
		Slug:        "test-modpack",
		Description: "A test modpack",
		MCVersion:   "1.20.1",
		Loader:      "forge",
	}

	// Use a subdirectory that doesn't exist
	outputPath := filepath.Join(tmpDir, "subdir", "test-modpack.json")
	err := saveRecipe(recipe, outputPath)
	if err != nil {
		t.Fatalf("saveRecipe() failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("recipe file was not created: %s", outputPath)
	}
}

func TestRecipeCommandExists(t *testing.T) {
	if RecipeCmd == nil {
		t.Fatal("RecipeCmd is nil")
	}

	if RecipeCmd.Use != "recipe" {
		t.Errorf("RecipeCmd.Use = %q, want %q", RecipeCmd.Use, "recipe")
	}

	// Check that create subcommand exists
	createCmd := RecipeCmd.Commands()
	if len(createCmd) == 0 {
		t.Fatal("RecipeCmd has no subcommands")
	}

	found := false
	for _, cmd := range createCmd {
		if cmd.Use == "create" {
			found = true
			break
		}
	}

	if !found {
		t.Error("create subcommand not found")
	}
}
