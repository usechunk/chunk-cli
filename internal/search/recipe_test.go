package search

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLoadRecipe_JSON(t *testing.T) {
	// Create a temporary test recipe file
	tmpDir := t.TempDir()
	recipeFile := filepath.Join(tmpDir, "test-recipe.json")

	recipeContent := `{
		"name": "Test Modpack",
		"description": "A test modpack",
		"mc_version": "1.20.1",
		"loader": "forge",
		"loader_version": "47.3.0",
		"recommended_ram_gb": 8,
		"tags": ["test", "forge"]
	}`

	if err := os.WriteFile(recipeFile, []byte(recipeContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	recipe, err := LoadRecipe(recipeFile, "test/bench")
	if err != nil {
		t.Fatalf("LoadRecipe failed: %v", err)
	}

	// Verify recipe contents
	if recipe.Name != "Test Modpack" {
		t.Errorf("Expected name 'Test Modpack', got '%s'", recipe.Name)
	}
	if recipe.MCVersion != "1.20.1" {
		t.Errorf("Expected mc_version '1.20.1', got '%s'", recipe.MCVersion)
	}
	if recipe.Loader != "forge" {
		t.Errorf("Expected loader 'forge', got '%s'", recipe.Loader)
	}
	if recipe.BenchName != "test/bench" {
		t.Errorf("Expected bench name 'test/bench', got '%s'", recipe.BenchName)
	}
	if recipe.Slug != "test-recipe" {
		t.Errorf("Expected slug 'test-recipe', got '%s'", recipe.Slug)
	}
}

func TestLoadRecipe_YAML(t *testing.T) {
	// Create a temporary test recipe file
	tmpDir := t.TempDir()
	recipeFile := filepath.Join(tmpDir, "test-recipe.yaml")

	recipeContent := `name: Test Modpack
description: A test modpack
mc_version: "1.20.1"
loader: fabric
loader_version: "0.15.0"
recommended_ram_gb: 4
tags:
  - test
  - fabric`

	if err := os.WriteFile(recipeFile, []byte(recipeContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	recipe, err := LoadRecipe(recipeFile, "test/bench")
	if err != nil {
		t.Fatalf("LoadRecipe failed: %v", err)
	}

	// Verify recipe contents
	if recipe.Name != "Test Modpack" {
		t.Errorf("Expected name 'Test Modpack', got '%s'", recipe.Name)
	}
	if recipe.MCVersion != "1.20.1" {
		t.Errorf("Expected mc_version '1.20.1', got '%s'", recipe.MCVersion)
	}
	if recipe.Loader != "fabric" {
		t.Errorf("Expected loader 'fabric', got '%s'", recipe.Loader)
	}
	if len(recipe.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(recipe.Tags))
	}
}

func TestLoadRecipesFromBench(t *testing.T) {
	// Create a temporary bench with Recipes directory
	tmpDir := t.TempDir()
	recipesDir := filepath.Join(tmpDir, "Recipes")
	if err := os.MkdirAll(recipesDir, 0755); err != nil {
		t.Fatalf("Failed to create Recipes directory: %v", err)
	}

	// Create test recipe files
	recipe1 := filepath.Join(recipesDir, "recipe1.json")
	recipe1Content := `{
		"name": "Recipe 1",
		"mc_version": "1.20.1",
		"loader": "forge"
	}`
	if err := os.WriteFile(recipe1, []byte(recipe1Content), 0644); err != nil {
		t.Fatalf("Failed to create recipe1: %v", err)
	}

	recipe2 := filepath.Join(recipesDir, "recipe2.yaml")
	recipe2Content := `name: Recipe 2
mc_version: "1.19.2"
loader: fabric`
	if err := os.WriteFile(recipe2, []byte(recipe2Content), 0644); err != nil {
		t.Fatalf("Failed to create recipe2: %v", err)
	}

	// Create a non-recipe file that should be ignored
	readme := filepath.Join(recipesDir, "README.md")
	if err := os.WriteFile(readme, []byte("# Recipes"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}

	// Load recipes
	recipes, err := LoadRecipesFromBench(tmpDir, "test/bench")
	if err != nil {
		t.Fatalf("LoadRecipesFromBench failed: %v", err)
	}

	// Verify we loaded 2 recipes (README should be ignored)
	if len(recipes) != 2 {
		t.Errorf("Expected 2 recipes, got %d", len(recipes))
	}

	// Verify recipe names
	foundRecipe1 := false
	foundRecipe2 := false
	for _, r := range recipes {
		if r.Name == "Recipe 1" {
			foundRecipe1 = true
		}
		if r.Name == "Recipe 2" {
			foundRecipe2 = true
		}
	}

	if !foundRecipe1 {
		t.Error("Recipe 1 not found in loaded recipes")
	}
	if !foundRecipe2 {
		t.Error("Recipe 2 not found in loaded recipes")
	}
}

func TestLoadRecipesFromBench_NoRecipesDir(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := LoadRecipesFromBench(tmpDir, "test/bench")
	if err == nil {
		t.Error("Expected error when Recipes directory doesn't exist")
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		query    string
		minScore int // Minimum expected score (0 means no match expected)
	}{
		{
			name:     "exact match",
			text:     "all the mods",
			query:    "all the mods",
			minScore: 100,
		},
		{
			name:     "exact word match",
			text:     "all the mods 9",
			query:    "mods",
			minScore: 80,
		},
		{
			name:     "prefix match",
			text:     "minecraft",
			query:    "mine",
			minScore: 50,
		},
		{
			name:     "contains match",
			text:     "all the mods",
			query:    "the",
			minScore: 30,
		},
		{
			name:     "no match",
			text:     "fabric skyblock",
			query:    "forge",
			minScore: 0,
		},
		{
			name:     "case insensitive",
			text:     "all the mods",
			query:    "all",
			minScore: 50, // Should match as prefix (after lowercase)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := fuzzyMatch(tt.text, tt.query)
			if tt.minScore > 0 && score < tt.minScore {
				t.Errorf("Expected score >= %d, got %d", tt.minScore, score)
			}
			if tt.minScore == 0 && score > 0 {
				t.Errorf("Expected no match (score 0), got %d", score)
			}
		})
	}
}

func TestMatchRecipe(t *testing.T) {
	recipe := &Recipe{
		Name:        "All The Mods 9",
		Slug:        "all-the-mods-9",
		Description: "Kitchen sink pack with 400+ mods",
		Tags:        []string{"kitchen-sink", "tech", "magic"},
		Author:      "ATM Team",
	}

	tests := []struct {
		name          string
		query         string
		shouldMatch   bool
		expectedField string
	}{
		{
			name:          "match by name",
			query:         "all the mods",
			shouldMatch:   true,
			expectedField: "name",
		},
		{
			name:          "match by slug",
			query:         "all-the-mods-9",
			shouldMatch:   true,
			expectedField: "slug",
		},
		{
			name:          "match by description",
			query:         "kitchen sink",
			shouldMatch:   true,
			expectedField: "name", // "kitchen-sink" tag might match first
		},
		{
			name:          "match by tag",
			query:         "tech",
			shouldMatch:   true,
			expectedField: "tags",
		},
		{
			name:        "match by author",
			query:       "atm",
			shouldMatch: true,
		},
		{
			name:        "no match",
			query:       "fabric",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchRecipe(recipe, tt.query)
			if tt.shouldMatch && result == nil {
				t.Error("Expected match but got nil")
			}
			if !tt.shouldMatch && result != nil {
				t.Errorf("Expected no match but got result with score %d", result.Score)
			}
		})
	}
}

func TestSearch_Integration(t *testing.T) {
	// Setup temporary config and bench
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "chunk")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Set HOME to temp directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create a test git repository
	testRepoDir := filepath.Join(tmpDir, "test-repo")
	recipesDir := filepath.Join(testRepoDir, "Recipes")
	if err := os.MkdirAll(recipesDir, 0755); err != nil {
		t.Fatalf("Failed to create test repo: %v", err)
	}

	// Initialize as git repo
	cmd := exec.Command("git", "init", testRepoDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping test: git not available: %v", err)
	}

	// Create test recipe
	recipeFile := filepath.Join(recipesDir, "test-modpack.json")
	recipeContent := `{
		"name": "Test Modpack",
		"description": "A test modpack for integration testing",
		"mc_version": "1.20.1",
		"loader": "forge",
		"loader_version": "47.3.0",
		"tags": ["test", "integration"]
	}`
	if err := os.WriteFile(recipeFile, []byte(recipeContent), 0644); err != nil {
		t.Fatalf("Failed to create recipe: %v", err)
	}

	// Commit the recipe
	cmd = exec.Command("git", "-C", testRepoDir, "add", ".")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", testRepoDir, "config", "user.email", "test@test.com")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", testRepoDir, "config", "user.name", "Test User")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", testRepoDir, "commit", "-m", "Initial")
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping test: git commit failed: %v", err)
	}

	// Note: We can't easily test the full Search integration without
	// creating a full bench manager setup. The unit tests above cover
	// the core functionality.
}
