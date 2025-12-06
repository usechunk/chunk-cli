package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexinslc/chunk/internal/config"
	"github.com/spf13/cobra"
)

func TestUpdateCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "update without args",
			args:    []string{},
			wantErr: true, // Will error because no benches are installed
		},
		{
			name:    "update with specific bench",
			args:    []string{"--bench", "usechunk/recipes"},
			wantErr: true, // Will error because bench doesn't exist
		},
		{
			name:    "update with bench flag shorthand",
			args:    []string{"-b", "usechunk/recipes"},
			wantErr: true, // Will error because bench doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			rootCmd := &cobra.Command{Use: "chunk"}
			rootCmd.AddCommand(UpdateCmd)

			_, err := executeCommand(rootCmd, append([]string{"update"}, tt.args...)...)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateCmd error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateCommandFlags(t *testing.T) {
	// Test that the update command has the expected flags
	rootCmd := &cobra.Command{Use: "chunk"}
	rootCmd.AddCommand(UpdateCmd)

	// Check that --bench flag exists
	benchFlag := UpdateCmd.Flags().Lookup("bench")
	if benchFlag == nil {
		t.Error("Expected --bench flag to exist")
	}
	if benchFlag.Shorthand != "b" {
		t.Errorf("Expected --bench shorthand to be 'b', got '%s'", benchFlag.Shorthand)
	}
}
