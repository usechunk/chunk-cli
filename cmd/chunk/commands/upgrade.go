package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexinslc/chunk/internal/install"
	"github.com/alexinslc/chunk/internal/preserve"
	"github.com/alexinslc/chunk/internal/sources"
	"github.com/alexinslc/chunk/internal/tracking"
	"github.com/alexinslc/chunk/internal/ui"
	"github.com/spf13/cobra"
)

var (
	upgradeDir    string
	dryRun        bool
	skipBackup    bool
	upgradeVerify bool
)

var UpgradeCmd = &cobra.Command{
	Use:   "upgrade [modpack]",
	Short: "Upgrade an existing modpack server",
	Long: `Upgrade an existing modpack server installation to the latest version.

This command will:
  - Preserve world data
  - Preserve player data
  - Preserve custom configuration files
  - Download the latest modpack version
  - Update mods and mod loader if needed
  - Provide warnings before any destructive operations

Examples:
  chunk upgrade                              # Upgrade from installed.json
  chunk upgrade atm9                         # Upgrade specific modpack
  chunk upgrade atm9 --dir /opt/minecraft/server
  chunk upgrade --dry-run                    # Preview changes without upgrading`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUpgrade,
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("🔄 Chunk Modpack Upgrader")
	fmt.Println()

	// Determine server directory
	serverDir := upgradeDir
	if serverDir == "" {
		serverDir = "./server"
	}

	// Make path absolute
	absServerDir, err := filepath.Abs(serverDir)
	if err != nil {
		return fmt.Errorf("failed to resolve server path: %w", err)
	}

	// Check if directory exists
	if _, err := os.Stat(absServerDir); os.IsNotExist(err) {
		return fmt.Errorf("server directory does not exist: %s", absServerDir)
	}

	// Try to get modpack identifier from args or from tracking
	var identifier string
	if len(args) > 0 {
		identifier = args[0]
	} else {
		// Try to get from tracking system
		tracker, err := tracking.NewTracker()
		if err != nil {
			return fmt.Errorf("failed to initialize tracker: %w", err)
		}

		installation, err := tracker.GetInstallation(absServerDir)
		if err != nil {
			return fmt.Errorf("failed to get installation info: %w", err)
		}

		if installation == nil {
			return fmt.Errorf("no installation found at %s. Please specify the modpack identifier.", absServerDir)
		}

		identifier = installation.Slug
		ui.PrintInfo(fmt.Sprintf("Detected modpack: %s", identifier))
	}

	// Get current version info
	currentVersion, currentRecipe, err := getCurrentVersion(absServerDir)
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Could not detect current version: %v", err))
		currentVersion = "unknown"
	}

	ui.PrintInfo(fmt.Sprintf("Current version: %s", currentVersion))

	// Fetch latest version from sources
	ui.PrintInfo("Checking for updates...")
	sourceManager := sources.NewSourceManager()
	newModpack, err := sourceManager.Fetch(identifier)
	if err != nil {
		return fmt.Errorf("failed to fetch latest version: %w", err)
	}

	newVersion := getModpackVersion(newModpack)
	ui.PrintInfo(fmt.Sprintf("Available version: %s", newVersion))

	// Compare versions
	if currentVersion == newVersion && currentVersion != "unknown" {
		fmt.Println()
		ui.PrintSuccess("Already up to date!")
		return nil
	}

	// Show what will change
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 Upgrade Summary")
	fmt.Println()
	fmt.Printf("   Modpack:       %s\n", newModpack.Name)
	fmt.Printf("   Current:       %s\n", currentVersion)
	fmt.Printf("   New:           %s\n", newVersion)
	fmt.Printf("   Minecraft:     %s\n", newModpack.MCVersion)
	fmt.Printf("   Loader:        %s %s\n", newModpack.Loader, newModpack.LoaderVersion)
	fmt.Printf("   Server Mods:   %d\n", len(newModpack.Mods))

	// Show version comparison if we have recipe data
	if currentRecipe != nil {
		fmt.Println()
		currentMCVersion, _ := currentRecipe["mc_version"].(string)
		currentLoader, _ := currentRecipe["loader"].(string)
		currentLoaderVersion, _ := currentRecipe["loader_version"].(string)

		if currentMCVersion != "" && currentMCVersion != newModpack.MCVersion {
			fmt.Printf("   ⚠️  Minecraft version changing: %s → %s\n", currentMCVersion, newModpack.MCVersion)
		}
		if currentLoader != "" && currentLoader != string(newModpack.Loader) {
			fmt.Printf("   ⚠️  Loader type changing: %s → %s\n", currentLoader, newModpack.Loader)
		}
		if currentLoaderVersion != "" && currentLoaderVersion != newModpack.LoaderVersion {
			fmt.Printf("   📦 Loader version: %s → %s\n", currentLoaderVersion, newModpack.LoaderVersion)
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Dry run - stop here
	if dryRun {
		fmt.Println()
		ui.PrintInfo("Dry run mode - no changes made")
		fmt.Println()
		fmt.Println("To perform the upgrade, run without --dry-run")
		return nil
	}

	// Check for critical files to preserve
	preserver := preserve.NewDataPreserver()
	criticalFiles := preserver.GetCriticalFiles(absServerDir)

	if len(criticalFiles) > 0 {
		fmt.Println()
		fmt.Println("💾 Data to preserve:")
		for _, file := range criticalFiles {
			fmt.Printf("   • %s\n", file)
		}
	}

	// Create backup unless skipped
	var backupDir string
	if !skipBackup && len(criticalFiles) > 0 {
		fmt.Println()
		spinner := ui.NewSpinner("Creating backup...")
		spinner.Start()
		backupDir, err = preserver.BackupBeforeUpgrade(absServerDir)
		if err != nil {
			spinner.Error(fmt.Sprintf("Backup failed: %v", err))
			return fmt.Errorf("backup failed: %w", err)
		}
		spinner.Success(fmt.Sprintf("Backup created: %s", filepath.Base(backupDir)))
	}

	// Perform upgrade using install engine
	fmt.Println()
	ui.PrintInfo("Downloading and installing new version...")

	installer := install.NewInstaller()
	opts := &install.Options{
		Identifier:   identifier,
		DestDir:      absServerDir,
		PreserveData: true,
		SkipVerify:   !upgradeVerify,
	}

	result, err := installer.Install(opts)
	if err != nil {
		// Attempt rollback if we have a backup
		if backupDir != "" {
			ui.PrintWarning("Installation failed, attempting rollback...")
			if restoreErr := preserver.RestoreFromBackup(absServerDir, backupDir); restoreErr != nil {
				ui.PrintError(fmt.Sprintf("Rollback failed: %v", restoreErr))
				return fmt.Errorf("upgrade failed and rollback failed: %w, rollback error: %v", err, restoreErr)
			}
			ui.PrintSuccess("Successfully rolled back to previous version")
		}
		return fmt.Errorf("upgrade failed: %w", err)
	}

	// Restore preserved data
	if backupDir != "" {
		fmt.Println()
		spinner := ui.NewSpinner("Restoring preserved data...")
		spinner.Start()

		// Restore world data
		worldPaths := []string{"world", "world_nether", "world_the_end"}
		for _, worldPath := range worldPaths {
			srcPath := filepath.Join(backupDir, worldPath)
			dstPath := filepath.Join(absServerDir, worldPath)

			if _, err := os.Stat(srcPath); err == nil {
				// Remove new world if it exists
				os.RemoveAll(dstPath)
				if err := preserver.CopyDir(srcPath, dstPath); err != nil {
					ui.PrintWarning(fmt.Sprintf("Failed to restore %s: %v", worldPath, err))
				}
			}
		}

		// Restore server configuration files
		configFiles := []string{
			"server.properties",
			"whitelist.json",
			"ops.json",
			"banned-players.json",
			"banned-ips.json",
		}

		for _, configFile := range configFiles {
			srcPath := filepath.Join(backupDir, configFile)
			dstPath := filepath.Join(absServerDir, configFile)

			if _, err := os.Stat(srcPath); err == nil {
				if err := preserver.CopyFile(srcPath, dstPath); err != nil {
					ui.PrintWarning(fmt.Sprintf("Failed to restore %s: %v", configFile, err))
				}
			}
		}

		spinner.Success("Data restored")
	}

	// Update tracking
	if trackErr := install.TrackInstallation(result, identifier); trackErr != nil {
		ui.PrintWarning(fmt.Sprintf("Failed to update tracking: %v", trackErr))
	}

	// Clean up backup after successful upgrade
	if backupDir != "" && !skipBackup {
		ui.PrintInfo(fmt.Sprintf("Backup retained at: %s", backupDir))
		ui.PrintInfo("Remove manually if no longer needed")
	}

	// Print success summary
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Upgrade Complete!")
	fmt.Println()
	fmt.Printf("   Modpack:   %s\n", result.ModpackName)
	fmt.Printf("   Version:   %s\n", newVersion)
	fmt.Printf("   Location:  %s\n", result.DestDir)
	if backupDir != "" {
		fmt.Printf("   Backup:    %s\n", backupDir)
	}
	fmt.Println()
	fmt.Println("To start the server:")
	fmt.Printf("   cd %s\n", result.DestDir)
	fmt.Println("   ./start.sh (Linux/Mac) or start.bat (Windows)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}

// getCurrentVersion reads the current version from .chunk-recipe.json
func getCurrentVersion(serverDir string) (string, map[string]interface{}, error) {
	recipeFile := filepath.Join(serverDir, ".chunk-recipe.json")

	data, err := os.ReadFile(recipeFile)
	if err != nil {
		return "unknown", nil, err
	}

	var recipe map[string]interface{}
	if err := json.Unmarshal(data, &recipe); err != nil {
		return "unknown", nil, err
	}

	// Try to get version from recipe
	if version, ok := recipe["version"].(string); ok && version != "" {
		return version, recipe, nil
	}

	// Fallback to mc_version + loader as version
	mcVersion, _ := recipe["mc_version"].(string)
	loader, _ := recipe["loader"].(string)
	if mcVersion != "" && loader != "" {
		return fmt.Sprintf("%s-%s", mcVersion, loader), recipe, nil
	}

	return "unknown", recipe, nil
}

// getModpackVersion extracts a version string from a modpack
func getModpackVersion(modpack *sources.Modpack) string {
	// For now, use MC version + loader as version
	// In the future, recipes should have explicit version fields
	return fmt.Sprintf("%s-%s", modpack.MCVersion, modpack.Loader)
}

func init() {
	UpgradeCmd.Flags().StringVarP(&upgradeDir, "dir", "d", "", "Server directory to upgrade (default: ./server)")
	UpgradeCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without upgrading")
	UpgradeCmd.Flags().BoolVar(&skipBackup, "skip-backup", false, "Skip backup creation (not recommended)")
	UpgradeCmd.Flags().BoolVar(&upgradeVerify, "verify", true, "Verify checksums of downloaded files")

	// Suppress usage printing on errors
	UpgradeCmd.SilenceUsage = true
}
