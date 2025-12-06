package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexinslc/chunk/internal/bench"
	"github.com/alexinslc/chunk/internal/config"
	"github.com/spf13/cobra"
)

func TestBenchCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "list benches with no args",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "remove without bench name",
			args:    []string{"remove"},
			wantErr: true,
		},
		{
			name:    "info without bench name",
			args:    []string{"info"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{Use: "chunk"}
			rootCmd.AddCommand(BenchCmd)

			_, err := executeCommand(rootCmd, append([]string{"bench"}, tt.args...)...)
			if (err != nil) != tt.wantErr {
				t.Errorf("BenchCmd error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bench.NormalizeGitHubURL(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeGitHubURL(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetBenchesDir(t *testing.T) {
	dir, err := bench.GetBenchesDir()
	if err != nil {
		t.Fatalf("GetBenchesDir() error = %v", err)
	}
	
	// Should contain .chunk/Benches
	if !filepath.IsAbs(dir) {
		t.Errorf("GetBenchesDir() returned non-absolute path: %s", dir)
	}
	
	if !contains(dir, ".chunk") || !contains(dir, "Benches") {
		t.Errorf("GetBenchesDir() = %s, should contain .chunk/Benches", dir)
	}
}

func TestConfigWithBenches(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	
	// Set HOME to temp directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	// Create config directory structure
	configDir := filepath.Join(tmpDir, ".config", "chunk")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	
	// Save the test config
	testConfigPath := filepath.Join(configDir, "config.json")
	data := []byte(`{"config_version":"1.0","benches":[{"name":"test/bench","url":"https://github.com/test/bench","path":"/tmp/test/bench","added":"2025-01-15T10:30:00Z"}]}`)
	if err := os.WriteFile(testConfigPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	
	// Load and verify
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	
	if len(loaded.Benches) != 1 {
		t.Errorf("Expected 1 bench, got %d", len(loaded.Benches))
	}
	
	if loaded.Benches[0].Name != "test/bench" {
		t.Errorf("Expected bench name 'test/bench', got '%s'", loaded.Benches[0].Name)
	}
}

// Helper function
func contains(s, substr string) bool {
	return filepath.Base(filepath.Dir(s)) == substr || filepath.Base(s) == substr
}
