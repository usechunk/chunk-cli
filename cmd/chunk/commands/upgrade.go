package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	upgradeDir string
)

var UpgradeCmd = &cobra.Command{
	Use:   "upgrade <modpack>",
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
  chunk upgrade atm9
  chunk upgrade alexinslc/my-cool-mod
  chunk upgrade atm9 --dir /opt/minecraft/server`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		modpack := args[0]
		fmt.Printf("Upgrading modpack: %s\n", modpack)

		if upgradeDir != "" {
			fmt.Printf("Server directory: %s\n", upgradeDir)
		} else {
			fmt.Printf("Server directory: ./server\n")
		}

		fmt.Println()
		fmt.Println("⚠️  Upgrade functionality not yet implemented")
		fmt.Println()
		fmt.Println("Upgrade process will:")
		fmt.Println("  ✓ Back up world data")
		fmt.Println("  ✓ Preserve player data")
		fmt.Println("  ✓ Preserve custom configs")
		fmt.Println("  ✓ Update mods")
		fmt.Println("  ✓ Update mod loader if needed")
	},
}

func init() {
	UpgradeCmd.Flags().StringVarP(&upgradeDir, "dir", "d", "", "Server directory to upgrade (default: ./server)")
}
