package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexinslc/chunk/internal/tracking"
	"github.com/spf13/cobra"
)

func TestUpgradeCommandDryRun(t *testing.T) {
	// Create a temporary test directory
	tmpDir, err := os.MkdirTemp("", "chunk-upgrade-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	serverDir := filepath.Join(tmpDir, "server")
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatalf("Failed to create server dir: %v", err)
	}

	// Create world data
	worldDir := filepath.Join(serverDir, "world")
	if err := os.MkdirAll(worldDir, 0755); err != nil {
		t.Fatalf("Failed to create world dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worldDir, "level.dat"), []byte("test world"), 0644); err != nil {
		t.Fatalf("Failed to create level.dat: %v", err)
	}

	// Create .chunk-recipe.json
	recipeData := map[string]interface{}{
		"slug":               "test-pack",
		"bench":              "test-bench",
		"name":               "Test Pack",
		"version":            "1.0.0",
		"mc_version":         "1.20.1",
		"loader":             "forge",
		"loader_version":     "47.2.0",
		"recommended_ram_gb": 4,
		"installed_at":       "2024-01-01T00:00:00Z",
	}
	recipeJSON, err := json.MarshalIndent(recipeData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal recipe data: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, ".chunk-recipe.json"), recipeJSON, 0644); err != nil {
		t.Fatalf("Failed to create recipe file: %v", err)
	}

	// Test that upgrade with non-existent modpack shows appropriate error
	rootCmd := &cobra.Command{Use: "chunk"}
	
	// Create a fresh command instance to avoid flag state issues
	testUpgradeCmd := &cobra.Command{
		Use:   UpgradeCmd.Use,
		Short: UpgradeCmd.Short,
		Long:  UpgradeCmd.Long,
		Args:  UpgradeCmd.Args,
		RunE:  UpgradeCmd.RunE,
	}
	testUpgradeCmd.Flags().StringVarP(&upgradeDir, "dir", "d", "", "Server directory to upgrade (default: ./server)")
	testUpgradeCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without upgrading")
	testUpgradeCmd.Flags().BoolVar(&skipBackup, "skip-backup", false, "Skip backup creation (not recommended)")
	testUpgradeCmd.Flags().BoolVar(&upgradeVerify, "verify", true, "Verify checksums of downloaded files")
	testUpgradeCmd.SilenceUsage = true
	
	rootCmd.AddCommand(testUpgradeCmd)

	_, err = executeCommand(rootCmd, "upgrade", "test-pack", "--dir", serverDir, "--dry-run")
	if err == nil {
		t.Error("Expected error for non-existent modpack, got nil")
	}
}

func TestUpgradeWithTracking(t *testing.T) {
	// Create a temporary test directory
	tmpDir, err := os.MkdirTemp("", "chunk-upgrade-tracking-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	serverDir := filepath.Join(tmpDir, "server")
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatalf("Failed to create server dir: %v", err)
	}

	// Create .chunk-recipe.json
	recipeData := map[string]interface{}{
		"slug":           "atm9",
		"bench":          "usechunk/recipes",
		"name":           "All the Mods 9",
		"version":        "0.3.1",
		"mc_version":     "1.20.1",
		"loader":         "forge",
		"loader_version": "47.2.0",
		"installed_at":   "2024-01-01T00:00:00Z",
	}
	recipeJSON, err := json.MarshalIndent(recipeData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal recipe data: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, ".chunk-recipe.json"), recipeJSON, 0644); err != nil {
		t.Fatalf("Failed to create recipe file: %v", err)
	}

	// Add to tracking
	tracker, err := tracking.NewTracker()
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}

	installation := &tracking.Installation{
		Slug:    "atm9",
		Version: "0.3.1",
		Bench:   "usechunk/recipes",
		Path:    serverDir,
	}
	// Set InstalledAt to a non-zero value
	installation.InstalledAt = installation.InstalledAt.Add(1)

	if err := tracker.AddInstallation(installation); err != nil {
		t.Fatalf("Failed to add installation: %v", err)
	}

	// Test upgrade without modpack arg (should detect from tracking)
	rootCmd := &cobra.Command{Use: "chunk"}
	
	// Create a fresh command instance to avoid flag state issues
	testUpgradeCmd := &cobra.Command{
		Use:   UpgradeCmd.Use,
		Short: UpgradeCmd.Short,
		Long:  UpgradeCmd.Long,
		Args:  UpgradeCmd.Args,
		RunE:  UpgradeCmd.RunE,
	}
	testUpgradeCmd.Flags().StringVarP(&upgradeDir, "dir", "d", "", "Server directory to upgrade (default: ./server)")
	testUpgradeCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without upgrading")
	testUpgradeCmd.Flags().BoolVar(&skipBackup, "skip-backup", false, "Skip backup creation (not recommended)")
	testUpgradeCmd.Flags().BoolVar(&upgradeVerify, "verify", true, "Verify checksums of downloaded files")
	testUpgradeCmd.SilenceUsage = true
	
	rootCmd.AddCommand(testUpgradeCmd)

	_, err = executeCommand(rootCmd, "upgrade", "--dir", serverDir, "--dry-run")
	// Should fail because we don't have benches, but should detect the modpack
	if err == nil {
		t.Error("Expected error for missing benches, got nil")
	}

	// Clean up tracking
	if err := tracker.RemoveInstallation(serverDir); err != nil {
		t.Logf("Warning: failed to clean up tracking: %v", err)
	}
}

func TestGetCurrentVersion(t *testing.T) {
	tests := []struct {
		name           string
		recipeData     map[string]interface{}
		expectedVer    string
		expectError    bool
		createRecipe   bool
	}{
		{
			name: "with explicit version",
			recipeData: map[string]interface{}{
				"version":        "1.0.0",
				"mc_version":     "1.20.1",
				"loader":         "forge",
				"loader_version": "47.2.0",
			},
			expectedVer:  "1.0.0",
			expectError:  false,
			createRecipe: true,
		},
		{
			name: "without version fallback to mc+loader",
			recipeData: map[string]interface{}{
				"mc_version":     "1.20.1",
				"loader":         "forge",
				"loader_version": "47.2.0",
			},
			expectedVer:  "1.20.1-forge",
			expectError:  false,
			createRecipe: true,
		},
		{
			name:         "missing recipe file",
			recipeData:   nil,
			expectedVer:  "unknown",
			expectError:  true,
			createRecipe: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "chunk-version-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if tt.createRecipe {
				recipeJSON, err := json.MarshalIndent(tt.recipeData, "", "  ")
				if err != nil {
					t.Fatalf("Failed to marshal recipe data: %v", err)
				}
				if err := os.WriteFile(filepath.Join(tmpDir, ".chunk-recipe.json"), recipeJSON, 0644); err != nil {
					t.Fatalf("Failed to create recipe file: %v", err)
				}
			}

			version, recipe, err := getCurrentVersion(tmpDir)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if recipe == nil {
					t.Error("Expected recipe data, got nil")
				}
			}

			if version != tt.expectedVer {
				t.Errorf("Expected version %q, got %q", tt.expectedVer, version)
			}
		})
	}
}
