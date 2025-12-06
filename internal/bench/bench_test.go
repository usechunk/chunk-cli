package bench

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexinslc/chunk/internal/config"
)

func TestNormalizeGitHubURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "github shorthand",
			input:    "usechunk/recipes",
			expected: "https://github.com/usechunk/recipes",
		},
		{
			name:     "https url unchanged",
			input:    "https://github.com/usechunk/recipes",
			expected: "https://github.com/usechunk/recipes",
		},
		{
			name:     "http url unchanged",
			input:    "http://example.com/repo.git",
			expected: "http://example.com/repo.git",
		},
		{
			name:     "ssh url unchanged",
			input:    "git@github.com:usechunk/recipes.git",
			expected: "git@github.com:usechunk/recipes.git",
		},
		{
			name:     "single word unchanged",
			input:    "recipes",
			expected: "recipes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeGitHubURL(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeGitHubURL(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetBenchesDir(t *testing.T) {
	dir, err := GetBenchesDir()
	if err != nil {
		t.Fatalf("GetBenchesDir() error = %v", err)
	}

	if !filepath.IsAbs(dir) {
		t.Errorf("GetBenchesDir() returned non-absolute path: %s", dir)
	}

	// Should end with .chunk/Benches
	expected := filepath.Join(".chunk", "Benches")
	if !endsWithPath(dir, expected) {
		t.Errorf("GetBenchesDir() = %s, should end with %s", dir, expected)
	}
}

func TestManagerList(t *testing.T) {
	// Setup temporary config
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "chunk")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Set HOME to temp directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create empty config
	cfg := &config.Config{
		ConfigVersion: "1.0",
		Benches:       []config.Bench{},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	benches := manager.List()
	if len(benches) != 0 {
		t.Errorf("Expected 0 benches, got %d", len(benches))
	}
}

func TestManagerGet(t *testing.T) {
	// Setup temporary config
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "chunk")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Set HOME to temp directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config with a bench
	testBench := config.Bench{
		Name: "test/bench",
		URL:  "https://github.com/test/bench",
		Path: "/tmp/test/bench",
	}
	cfg := &config.Config{
		ConfigVersion: "1.0",
		Benches:       []config.Bench{testBench},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Test getting existing bench
	bench, err := manager.Get("test/bench")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if bench.Name != "test/bench" {
		t.Errorf("Expected bench name 'test/bench', got '%s'", bench.Name)
	}

	// Test getting non-existent bench
	_, err = manager.Get("nonexistent/bench")
	if err == nil {
		t.Error("Expected error for non-existent bench, got nil")
	}
}

// Helper function to check if path ends with a specific pattern
func endsWithPath(path, suffix string) bool {
	return filepath.Base(filepath.Dir(path)) == filepath.Base(filepath.Dir(suffix)) &&
		filepath.Base(path) == filepath.Base(suffix)
}

func TestValidateGitURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid https url",
			url:     "https://github.com/user/repo",
			wantErr: false,
		},
		{
			name:    "valid http url",
			url:     "http://example.com/repo.git",
			wantErr: false,
		},
		{
			name:    "valid ssh url",
			url:     "git@github.com:user/repo.git",
			wantErr: false,
		},
		{
			name:    "valid ssh protocol",
			url:     "ssh://git@github.com/user/repo.git",
			wantErr: false,
		},
		{
			name:    "valid file path",
			url:     "file:///tmp/repo",
			wantErr: false,
		},
		{
			name:    "valid absolute path",
			url:     "/tmp/test-repo",
			wantErr: false,
		},
		{
			name:    "valid relative path",
			url:     "./test-repo",
			wantErr: false,
		},
		{
			name:    "empty url",
			url:     "",
			wantErr: true,
		},
		{
			name:    "url with semicolon",
			url:     "https://example.com/repo;rm -rf",
			wantErr: true,
		},
		{
			name:    "url with ampersand",
			url:     "https://example.com/repo&malicious",
			wantErr: true,
		},
		{
			name:    "url with pipe",
			url:     "https://example.com/repo|cat",
			wantErr: true,
		},
		{
			name:    "url with backtick",
			url:     "https://example.com/repo`whoami`",
			wantErr: true,
		},
		{
			name:    "url with command substitution",
			url:     "https://example.com/$(whoami)/repo",
			wantErr: true,
		},
		{
			name:    "url with variable expansion",
			url:     "https://example.com/${VAR}/repo",
			wantErr: true,
		},
		{
			name:    "invalid protocol",
			url:     "ftp://example.com/repo",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGitURL(%s) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestManagerAdd(t *testing.T) {
// Setup temporary config
tmpDir := t.TempDir()
configDir := filepath.Join(tmpDir, ".config", "chunk")
if err := os.MkdirAll(configDir, 0755); err != nil {
t.Fatalf("Failed to create config dir: %v", err)
}

// Set HOME to temp directory
originalHome := os.Getenv("HOME")
os.Setenv("HOME", tmpDir)
defer os.Setenv("HOME", originalHome)

// Create empty config
cfg := &config.Config{
ConfigVersion: "1.0",
Benches:       []config.Bench{},
}
if err := cfg.Save(); err != nil {
t.Fatalf("Failed to save config: %v", err)
}

// Create a test git repository
testRepoDir := filepath.Join(tmpDir, "test-repo")
if err := os.MkdirAll(filepath.Join(testRepoDir, "Recipes"), 0755); err != nil {
t.Fatalf("Failed to create test repo: %v", err)
}
// Initialize as git repo
cmd := exec.Command("git", "init", testRepoDir)
if err := cmd.Run(); err != nil {
t.Skipf("Skipping test: git not available: %v", err)
}
// Create a test file and commit
testFile := filepath.Join(testRepoDir, "Recipes", "test.yaml")
if err := os.WriteFile(testFile, []byte("test: recipe"), 0644); err != nil {
t.Fatalf("Failed to create test file: %v", err)
}
cmd = exec.Command("git", "-C", testRepoDir, "add", ".")
if err := cmd.Run(); err != nil {
t.Skipf("Skipping test: git add failed: %v", err)
}
cmd = exec.Command("git", "-C", testRepoDir, "config", "user.email", "test@test.com")
cmd.Run()
cmd = exec.Command("git", "-C", testRepoDir, "config", "user.name", "Test User")
cmd.Run()
cmd = exec.Command("git", "-C", testRepoDir, "commit", "-m", "Initial commit")
if err := cmd.Run(); err != nil {
t.Skipf("Skipping test: git commit failed: %v", err)
}

manager, err := NewManager()
if err != nil {
t.Fatalf("NewManager() error = %v", err)
}

// Test adding a bench
err = manager.Add("test/repo", testRepoDir)
if err != nil {
t.Errorf("Add() error = %v", err)
}

// Verify bench was added
benches := manager.List()
if len(benches) != 1 {
t.Errorf("Expected 1 bench after add, got %d", len(benches))
}

// Test adding duplicate bench
err = manager.Add("test/repo", testRepoDir)
if err == nil {
t.Error("Expected error when adding duplicate bench, got nil")
}

// Test adding bench without Recipes directory
noRecipesDir := filepath.Join(tmpDir, "no-recipes")
if err := os.MkdirAll(noRecipesDir, 0755); err != nil {
t.Fatalf("Failed to create no-recipes dir: %v", err)
}
cmd = exec.Command("git", "init", noRecipesDir)
cmd.Run()
cmd = exec.Command("git", "-C", noRecipesDir, "config", "user.email", "test@test.com")
cmd.Run()
cmd = exec.Command("git", "-C", noRecipesDir, "config", "user.name", "Test User")
cmd.Run()
emptyFile := filepath.Join(noRecipesDir, "README.md")
os.WriteFile(emptyFile, []byte("test"), 0644)
cmd = exec.Command("git", "-C", noRecipesDir, "add", ".")
cmd.Run()
cmd = exec.Command("git", "-C", noRecipesDir, "commit", "-m", "Initial")
cmd.Run()

err = manager.Add("invalid/repo", noRecipesDir)
if err == nil {
t.Error("Expected error for repo without Recipes directory, got nil")
}
if !strings.Contains(err.Error(), "no Recipes/") {
t.Errorf("Expected error about missing Recipes directory, got: %v", err)
}

// Test path traversal protection
err = manager.Add("../malicious", testRepoDir)
if err == nil {
t.Error("Expected error for path traversal attempt, got nil")
}
}

func TestManagerRemove(t *testing.T) {
// Setup temporary config
tmpDir := t.TempDir()
configDir := filepath.Join(tmpDir, ".config", "chunk")
if err := os.MkdirAll(configDir, 0755); err != nil {
t.Fatalf("Failed to create config dir: %v", err)
}

// Set HOME to temp directory
originalHome := os.Getenv("HOME")
os.Setenv("HOME", tmpDir)
defer os.Setenv("HOME", originalHome)

// Create a bench directory
benchesDir := filepath.Join(tmpDir, ".chunk", "Benches", "test", "repo")
if err := os.MkdirAll(benchesDir, 0755); err != nil {
t.Fatalf("Failed to create bench dir: %v", err)
}

// Create config with a bench
testBench := config.Bench{
Name: "test/repo",
URL:  "https://github.com/test/repo",
Path: benchesDir,
}
cfg := &config.Config{
ConfigVersion: "1.0",
Benches:       []config.Bench{testBench},
}
if err := cfg.Save(); err != nil {
t.Fatalf("Failed to save config: %v", err)
}

manager, err := NewManager()
if err != nil {
t.Fatalf("NewManager() error = %v", err)
}

// Test removing existing bench
err = manager.Remove("test/repo")
if err != nil {
t.Errorf("Remove() error = %v", err)
}

// Verify bench was removed
benches := manager.List()
if len(benches) != 0 {
t.Errorf("Expected 0 benches after remove, got %d", len(benches))
}

// Verify directory was deleted
if _, err := os.Stat(benchesDir); !os.IsNotExist(err) {
t.Error("Expected bench directory to be deleted")
}

// Test removing non-existent bench
err = manager.Remove("nonexistent/repo")
if err == nil {
t.Error("Expected error when removing non-existent bench, got nil")
}
}
