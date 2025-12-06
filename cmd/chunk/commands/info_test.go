package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexinslc/chunk/internal/config"
	"github.com/spf13/cobra"
)

func TestInfoCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:        "info without args",
			args:        []string{},
			wantErr:     true,
			errContains: "accepts 1 arg(s), received 0",
		},
		{
			name:        "info with too many args",
			args:        []string{"recipe1", "recipe2"},
			wantErr:     true,
			errContains: "accepts 1 arg(s), received 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := InfoCmd
			cmd.SetArgs(tt.args)

			// Capture output
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// Reset command for next test
			cmd.SetArgs([]string{})
		})
	}
}

func TestInfoCommandWithBench(t *testing.T) {
	// Create a temporary directory for test benches
	tmpDir := t.TempDir()
	oldConfigPath := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldConfigPath)

	// Create test bench structure
	benchPath := filepath.Join(tmpDir, ".chunk", "Benches", "test-bench")
	recipesPath := filepath.Join(benchPath, "Recipes")
	if err := os.MkdirAll(recipesPath, 0755); err != nil {
		t.Fatalf("failed to create test bench: %v", err)
	}

	// Create a test recipe file
	testRecipe := `{
  "name": "Test Modpack",
  "slug": "test-modpack",
  "version": "1.0.0",
  "description": "A test modpack",
  "mc_version": "1.20.1",
  "loader": "forge",
  "loader_version": "47.2.0",
  "java_version": 17,
  "recommended_ram_gb": 8,
  "disk_space_gb": 10,
  "license": "MIT",
  "homepage": "https://example.com/test-modpack",
  "download_url": "https://example.com/download",
  "download_size_mb": 500
}`
	recipeFile := filepath.Join(recipesPath, "test-modpack.json")
	if err := os.WriteFile(recipeFile, []byte(testRecipe), 0644); err != nil {
		t.Fatalf("failed to create test recipe: %v", err)
	}

	// Create config with the test bench
	cfg := &config.Config{
		Benches: []config.Bench{
			{
				Name: "test-bench",
				Path: benchPath,
				URL:  "https://example.com/test-bench",
			},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save test config: %v", err)
	}

	tests := []struct {
		name         string
		args         []string
		wantErr      bool
		outputChecks []string
	}{
		{
			name:    "info with valid recipe",
			args:    []string{"test-modpack"},
			wantErr: false,
			outputChecks: []string{
				"test-modpack",
				"A test modpack",
				"1.0.0",
				"1.20.1",
				"Forge",
				"47.2.0",
				"MIT",
				"8 GB",
				"10 GB",
				"test-bench",
			},
		},
		{
			name:         "info with non-existent recipe",
			args:         []string{"non-existent"},
			wantErr:      true,
			outputChecks: []string{"not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create a fresh command
			cmd := &cobra.Command{
				Use: "info",
				RunE: func(cmd *cobra.Command, args []string) error {
					return InfoCmd.RunE(cmd, args)
				},
			}
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// Check output contains expected strings
			for _, check := range tt.outputChecks {
				if !strings.Contains(output, check) && (err == nil || !strings.Contains(err.Error(), check)) {
					t.Errorf("expected output to contain '%s', got:\n%s\nerror: %v", check, output, err)
				}
			}
		})
	}
}

func TestInfoCommandJSON(t *testing.T) {
	// Create a temporary directory for test benches
	tmpDir := t.TempDir()
	oldConfigPath := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldConfigPath)

	// Create test bench structure
	benchPath := filepath.Join(tmpDir, ".chunk", "Benches", "test-bench")
	recipesPath := filepath.Join(benchPath, "Recipes")
	if err := os.MkdirAll(recipesPath, 0755); err != nil {
		t.Fatalf("failed to create test bench: %v", err)
	}

	// Create a test recipe file
	testRecipe := `{
  "name": "Test Modpack",
  "slug": "test-modpack",
  "version": "1.0.0",
  "mc_version": "1.20.1",
  "loader": "forge",
  "java_version": 17
}`
	recipeFile := filepath.Join(recipesPath, "test-modpack.json")
	if err := os.WriteFile(recipeFile, []byte(testRecipe), 0644); err != nil {
		t.Fatalf("failed to create test recipe: %v", err)
	}

	// Create config with the test bench
	cfg := &config.Config{
		Benches: []config.Bench{
			{
				Name: "test-bench",
				Path: benchPath,
				URL:  "https://example.com/test-bench",
			},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save test config: %v", err)
	}

	// Test JSON output
	infoJSON = true
	defer func() { infoJSON = false }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := &cobra.Command{
		Use: "info",
		RunE: func(cmd *cobra.Command, args []string) error {
			return InfoCmd.RunE(cmd, args)
		},
	}
	cmd.SetArgs([]string{"test-modpack"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify JSON fields
	expectedFields := map[string]interface{}{
		"name":         "Test Modpack",
		"slug":         "test-modpack",
		"version":      "1.0.0",
		"mc_version":   "1.20.1",
		"loader":       "forge",
		"java_version": float64(17),
		"bench_name":   "test-bench",
	}

	for key, expectedValue := range expectedFields {
		actualValue, ok := result[key]
		if !ok {
			t.Errorf("expected field '%s' not found in JSON output", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("field '%s': expected '%v', got '%v'", key, expectedValue, actualValue)
		}
	}
}
