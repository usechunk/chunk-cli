package bench

import (
	"os"
	"path/filepath"
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
