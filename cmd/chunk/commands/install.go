package commands

import (
	"fmt"
	"os"

	"github.com/alexinslc/chunk/internal/install"
	"github.com/alexinslc/chunk/internal/ui"
	"github.com/spf13/cobra"
)

var (
	installDir string
)

var InstallCmd = &cobra.Command{
	Use:   "install <modpack>",
	Short: "Install a modpack server",
	Long: `Install a modpack server from various sources.

Sources:
  - ChunkHub registry: chunk install atm9
  - GitHub repository: chunk install alexinslc/my-cool-mod
  - Modrinth: chunk install modrinth:<slug>
  - Local file: chunk install ./modpack.mrpack

The command will:
  - Download the modpack
  - Install the correct mod loader (Forge/Fabric/NeoForge)
  - Download all server-side mods
  - Generate server configurations
  - Create start scripts`,
	Args: cobra.ExactArgs(1),
	RunE: runInstall,
}

func runInstall(cmd *cobra.Command, args []string) error {
	modpack := args[0]

	fmt.Println()
	fmt.Println("🚀 Chunk Modpack Installer")
	fmt.Println()

	installer := install.NewInstaller()

	// Normalize destination directory
	destDir := installDir
	if destDir == "" {
		destDir = "./server"
	}

	opts := &install.Options{
		Identifier:   modpack,
		DestDir:      destDir,
		PreserveData: false,
	}

	result, err := installer.Install(opts)
	if err != nil {
		// Attempt rollback on failure
		if rollbackErr := installer.Rollback(destDir); rollbackErr != nil {
			ui.PrintError(fmt.Sprintf("Rollback failed: %v", rollbackErr))
		}
		ui.PrintError(fmt.Sprintf("Installation failed: %v", err))
		return err
	}

	// Print success summary
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Installation Complete!")
	fmt.Println()
	fmt.Printf("   Modpack:   %s\n", result.ModpackName)
	fmt.Printf("   Minecraft: %s\n", result.MCVersion)
	fmt.Printf("   Loader:    %s", result.Loader)
	if result.LoaderVersion != "" {
		fmt.Printf(" %s", result.LoaderVersion)
	}
	fmt.Println()
	fmt.Printf("   Mods:      %d installed\n", result.ModsInstalled)
	fmt.Printf("   Location:  %s\n", result.DestDir)
	fmt.Println()
	fmt.Println("To start the server:")
	fmt.Printf("   cd %s\n", result.DestDir)
	fmt.Println("   ./start.sh (Linux/Mac) or start.bat (Windows)")
	fmt.Println()
	fmt.Println("NOTE: Review and accept eula.txt before starting!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}

func init() {
	InstallCmd.Flags().StringVarP(&installDir, "dir", "d", "", "Installation directory (default: ./server)")

	// Suppress usage printing on errors
	InstallCmd.SilenceUsage = true
	InstallCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		cmd.Usage()
		return err
	})
}
