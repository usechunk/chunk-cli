package sources

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRecipeClientFetch(t *testing.T) {
	// Setup temporary test environment
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "chunk")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Set HOME to temp directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create a test bench with recipes
	testBenchDir := filepath.Join(tmpDir, "test-bench")
	recipesDir := filepath.Join(testBenchDir, "Recipes")
	if err := os.MkdirAll(recipesDir, 0755); err != nil {
		t.Fatalf("Failed to create recipes dir: %v", err)
	}

	// Initialize as git repo (required for benches)
	cmd := exec.Command("git", "init", testBenchDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping test: git not available: %v", err)
	}

	// Create test recipe
	recipeFile := filepath.Join(recipesDir, "test-modpack.json")
	recipeContent := `{
		"name": "Test Modpack",
		"slug": "test-modpack",
		"description": "A test modpack",
		"mc_version": "1.20.1",
		"loader": "forge",
		"loader_version": "47.3.0",
		"recommended_ram_gb": 4,
		"tags": ["test", "forge"],
		"download_url": "http://example.com/test.mrpack",
		"sha256": "abc123"
	}`
	if err := os.WriteFile(recipeFile, []byte(recipeContent), 0644); err != nil {
		t.Fatalf("Failed to create recipe: %v", err)
	}

	// Commit the recipe
	cmd = exec.Command("git", "-C", testBenchDir, "add", ".")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", testBenchDir, "config", "user.email", "test@test.com")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", testBenchDir, "config", "user.name", "Test User")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", testBenchDir, "commit", "-m", "Initial")
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping test: git commit failed: %v", err)
	}

	// Add bench to config
	benchesDir := filepath.Join(tmpDir, ".chunk", "Benches", "test-bench")
	if err := os.MkdirAll(benchesDir, 0755); err != nil {
		t.Fatalf("Failed to create benches dir: %v", err)
	}

	// Clone the test bench
	cmd = exec.Command("git", "clone", testBenchDir, benchesDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to clone bench: %v", err)
	}

	// Create config file with bench
	configFile := filepath.Join(configDir, "config.json")
	configContent := `{
		"config_version": "1.0",
		"telemetry_asked": false,
		"benches": [
			{
				"name": "test-bench",
				"url": "` + testBenchDir + `",
				"path": "` + benchesDir + `",
				"added": "2025-01-01T00:00:00Z"
			}
		]
	}`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Test fetching recipe
	client := NewRecipeClient()
	modpack, err := client.Fetch("test-modpack")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// Verify modpack contents
	if modpack.Name != "Test Modpack" {
		t.Errorf("Expected name 'Test Modpack', got '%s'", modpack.Name)
	}
	if modpack.Identifier != "test-modpack" {
		t.Errorf("Expected identifier 'test-modpack', got '%s'", modpack.Identifier)
	}
	if modpack.MCVersion != "1.20.1" {
		t.Errorf("Expected mc_version '1.20.1', got '%s'", modpack.MCVersion)
	}
	if modpack.Loader != LoaderForge {
		t.Errorf("Expected loader 'forge', got '%s'", modpack.Loader)
	}
	if modpack.ManifestURL != "http://example.com/test.mrpack" {
		t.Errorf("Expected manifest_url 'http://example.com/test.mrpack', got '%s'", modpack.ManifestURL)
	}
}

func TestRecipeClientFetchWithBench(t *testing.T) {
	// Similar setup to TestRecipeClientFetch
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "chunk")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	testBenchDir := filepath.Join(tmpDir, "test-bench")
	recipesDir := filepath.Join(testBenchDir, "Recipes")
	if err := os.MkdirAll(recipesDir, 0755); err != nil {
		t.Fatalf("Failed to create recipes dir: %v", err)
	}

	cmd := exec.Command("git", "init", testBenchDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping test: git not available: %v", err)
	}

	recipeFile := filepath.Join(recipesDir, "test-modpack.json")
	recipeContent := `{
		"name": "Test Modpack",
		"slug": "test-modpack",
		"mc_version": "1.20.1",
		"loader": "forge",
		"download_url": "http://example.com/test.mrpack"
	}`
	if err := os.WriteFile(recipeFile, []byte(recipeContent), 0644); err != nil {
		t.Fatalf("Failed to create recipe: %v", err)
	}

	cmd = exec.Command("git", "-C", testBenchDir, "add", ".")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", testBenchDir, "config", "user.email", "test@test.com")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", testBenchDir, "config", "user.name", "Test User")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", testBenchDir, "commit", "-m", "Initial")
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping test: git commit failed: %v", err)
	}

	benchesDir := filepath.Join(tmpDir, ".chunk", "Benches", "my-bench")
	if err := os.MkdirAll(benchesDir, 0755); err != nil {
		t.Fatalf("Failed to create benches dir: %v", err)
	}

	cmd = exec.Command("git", "clone", testBenchDir, benchesDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to clone bench: %v", err)
	}

	configFile := filepath.Join(configDir, "config.json")
	configContent := `{
		"config_version": "1.0",
		"telemetry_asked": false,
		"benches": [
			{
				"name": "my-bench",
				"url": "` + testBenchDir + `",
				"path": "` + benchesDir + `",
				"added": "2025-01-01T00:00:00Z"
			}
		]
	}`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Test fetching with explicit bench specification
	client := NewRecipeClient()
	modpack, err := client.Fetch("my-bench::test-modpack")
	if err != nil {
		t.Fatalf("Fetch with bench failed: %v", err)
	}

	if modpack.Name != "Test Modpack" {
		t.Errorf("Expected name 'Test Modpack', got '%s'", modpack.Name)
	}
}

func TestRecipeClientFetchNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "chunk")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config with no benches
	configFile := filepath.Join(configDir, "config.json")
	configContent := `{
		"config_version": "1.0",
		"telemetry_asked": false,
		"benches": []
	}`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	client := NewRecipeClient()
	_, err := client.Fetch("nonexistent-recipe")
	if err == nil {
		t.Error("Expected error when fetching nonexistent recipe")
	}
}

func TestRecipeClientSearch(t *testing.T) {
	// Similar setup but test search functionality
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "chunk")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	testBenchDir := filepath.Join(tmpDir, "test-bench")
	recipesDir := filepath.Join(testBenchDir, "Recipes")
	if err := os.MkdirAll(recipesDir, 0755); err != nil {
		t.Fatalf("Failed to create recipes dir: %v", err)
	}

	cmd := exec.Command("git", "init", testBenchDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping test: git not available: %v", err)
	}

	// Create multiple test recipes
	recipes := []struct {
		filename string
		content  string
	}{
		{
			filename: "atm9.json",
			content: `{
				"name": "All The Mods 9",
				"slug": "atm9",
				"mc_version": "1.20.1",
				"loader": "forge",
				"tags": ["kitchen-sink"],
				"download_url": "http://example.com/atm9.mrpack"
			}`,
		},
		{
			filename: "create.json",
			content: `{
				"name": "Create Above and Beyond",
				"slug": "create-above-beyond",
				"mc_version": "1.18.2",
				"loader": "forge",
				"tags": ["tech"],
				"download_url": "http://example.com/create.mrpack"
			}`,
		},
	}

	for _, recipe := range recipes {
		recipeFile := filepath.Join(recipesDir, recipe.filename)
		if err := os.WriteFile(recipeFile, []byte(recipe.content), 0644); err != nil {
			t.Fatalf("Failed to create recipe %s: %v", recipe.filename, err)
		}
	}

	cmd = exec.Command("git", "-C", testBenchDir, "add", ".")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", testBenchDir, "config", "user.email", "test@test.com")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", testBenchDir, "config", "user.name", "Test User")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", testBenchDir, "commit", "-m", "Initial")
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping test: git commit failed: %v", err)
	}

	benchesDir := filepath.Join(tmpDir, ".chunk", "Benches", "test-bench")
	if err := os.MkdirAll(benchesDir, 0755); err != nil {
		t.Fatalf("Failed to create benches dir: %v", err)
	}

	cmd = exec.Command("git", "clone", testBenchDir, benchesDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to clone bench: %v", err)
	}

	configFile := filepath.Join(configDir, "config.json")
	configContent := `{
		"config_version": "1.0",
		"telemetry_asked": false,
		"benches": [
			{
				"name": "test-bench",
				"url": "` + testBenchDir + `",
				"path": "` + benchesDir + `",
				"added": "2025-01-01T00:00:00Z"
			}
		]
	}`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Test search
	client := NewRecipeClient()
	results, err := client.Search("all the mods")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one search result")
	}

	// Verify first result is relevant
	if len(results) > 0 {
		if results[0].Source != "recipe" {
			t.Errorf("Expected source 'recipe', got '%s'", results[0].Source)
		}
	}
}
