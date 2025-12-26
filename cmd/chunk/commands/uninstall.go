package commands

import (
	"fmt"
	"os"

	"github.com/alexinslc/chunk/internal/ui"
	"github.com/alexinslc/chunk/internal/uninstall"
	"github.com/spf13/cobra"
)

var (
	uninstallDir        string
	uninstallKeepWorlds bool
	uninstallForce      bool
)

var UninstallCmd = &cobra.Command{
	Use:   "uninstall <modpack>",
	Short: "Uninstall a modpack server",
	Long: `Uninstall a modpack server installation and optionally preserve world data.

This command will:
  - Remove modpack files (mods, configs, libraries)
  - Optionally preserve world data and player files
  - Remove the installation from tracking
  - Prompt for confirmation before deletion

Examples:
  chunk uninstall atm9                      # Prompt for world preservation
  chunk uninstall atm9 --keep-worlds        # Keep world and player data
  chunk uninstall atm9 --force              # No confirmation prompts
  chunk uninstall atm9 --dir ./myserver     # Uninstall from custom directory`,
	Args: cobra.ExactArgs(1),
	RunE: runUninstall,
}

func runUninstall(cmd *cobra.Command, args []string) error {
	modpack := args[0]

	fmt.Println()
	fmt.Println("🗑️  Chunk Modpack Uninstaller")
	fmt.Println()
	fmt.Printf("Uninstalling: %s\n", modpack)

	// Normalize destination directory
	destDir := uninstallDir
	if destDir == "" {
		destDir = "./server"
	}

	// Create uninstaller
	uninstaller, err := uninstall.NewUninstaller()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create uninstaller: %v", err))
		return err
	}

	// Prepare options
	opts := &uninstall.Options{
		ServerDir:   destDir,
		ModpackSlug: modpack,
		KeepWorlds:  uninstallKeepWorlds,
		Force:       uninstallForce,
	}

	// Perform uninstall
	result, err := uninstaller.Uninstall(opts)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Uninstall failed: %v", err))
		return err
	}

	// Print success summary
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Uninstall Complete!")
	fmt.Println()

	if len(result.RemovedPaths) > 0 {
		fmt.Printf("   Removed %d items\n", len(result.RemovedPaths))
	}

	if len(result.PreservedPaths) > 0 {
		fmt.Printf("   Preserved %d items (world data)\n", len(result.PreservedPaths))
	}

	fmt.Printf("   Location: %s\n", destDir)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}

func init() {
	UninstallCmd.Flags().StringVarP(&uninstallDir, "dir", "d", "", "Server directory to uninstall from (default: ./server)")
	UninstallCmd.Flags().BoolVar(&uninstallKeepWorlds, "keep-worlds", false, "Keep world and player data")
	UninstallCmd.Flags().BoolVar(&uninstallForce, "force", false, "Skip confirmation prompts (respects --keep-worlds)")

	// Suppress usage printing on errors
	UninstallCmd.SilenceUsage = true
	UninstallCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		cmd.Usage()
		return err
	})
}
