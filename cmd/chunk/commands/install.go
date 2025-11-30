package commands

import (
	"fmt"

	"github.com/alexinslc/chunk/internal/dependencies"
	"github.com/alexinslc/chunk/internal/sources"
	"github.com/spf13/cobra"
)

var (
	installDir  string
	skipDeps    bool
	showDepTree bool
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
  - Resolve mod dependencies automatically
  - Install the correct mod loader (Forge/Fabric/NeoForge)
  - Download all server-side mods
  - Generate server configurations
  - Create start scripts`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		modpack := args[0]
		fmt.Printf("Installing modpack: %s\n", modpack)

		dir := installDir
		if dir == "" {
			dir = "./server"
		}
		fmt.Printf("Installation directory: %s\n", dir)

		// Fetch modpack metadata
		manager := sources.NewSourceManager()
		pack, err := manager.Fetch(modpack)
		if err != nil {
			fmt.Printf("❌ Error fetching modpack: %v\n", err)
			return
		}

		fmt.Printf("📦 Found: %s\n", pack.Name)
		fmt.Printf("   Minecraft: %s | Loader: %s\n", pack.MCVersion, pack.Loader)

		if len(pack.Mods) > 0 {
			fmt.Printf("   Mods: %d\n", len(pack.Mods))
		}

		// Resolve dependencies unless skipped
		if !skipDeps && len(pack.Mods) > 0 {
			fmt.Println("\n🔍 Resolving dependencies...")

			result, err := resolveDependencies(pack)
			if err != nil {
				fmt.Printf("❌ Dependency resolution failed: %v\n", err)
				return
			}

			// Show dependency tree if requested
			if showDepTree {
				printDependencyTree(result)
			}

			// Report any conflicts
			if len(result.Conflicts) > 0 {
				fmt.Println("\n⚠️  Version conflicts detected:")
				for _, conflict := range result.Conflicts {
					fmt.Printf("   - %s: required by %v with versions %v\n",
						conflict.ModID, conflict.RequiredBy, conflict.Versions)
				}
			}

			// Report missing dependencies
			if len(result.MissingDeps) > 0 {
				fmt.Println("\n⚠️  Missing dependencies:")
				for _, dep := range result.MissingDeps {
					reqStr := "required"
					if !dep.Required {
						reqStr = "optional"
					}
					fmt.Printf("   - %s (%s): %s\n", dep.ModID, dep.VersionExpr, reqStr)
				}
			}

			// Show install order
			fmt.Printf("\n📋 Install order (%d mods):\n", len(result.InstallOrder))
			for i, modID := range result.InstallOrder {
				fmt.Printf("   %d. %s\n", i+1, modID)
			}
		}

		fmt.Println("\n⚠️  Full install functionality not yet implemented")
	},
}

func init() {
	InstallCmd.Flags().StringVarP(&installDir, "dir", "d", "", "Installation directory (default: ./server)")
	InstallCmd.Flags().BoolVar(&skipDeps, "skip-deps", false, "Skip automatic dependency resolution")
	InstallCmd.Flags().BoolVar(&showDepTree, "show-deps", false, "Show dependency tree")
}

// resolveDependencies resolves all dependencies for a modpack.
func resolveDependencies(pack *sources.Modpack) (*dependencies.ResolutionResult, error) {
	// Convert sources.Mod to dependencies.ModInfo
	var mods []*dependencies.ModInfo
	for _, mod := range pack.Mods {
		modInfo := &dependencies.ModInfo{
			ID:      mod.ID,
			Name:    mod.Name,
			Version: mod.Version,
		}

		// Convert dependencies
		for _, dep := range mod.Dependencies {
			modInfo.Dependencies = append(modInfo.Dependencies, dependencies.Dependency{
				ModID:       dep.ModID,
				VersionExpr: dep.VersionExpr,
				Required:    dep.Required,
			})
		}

		mods = append(mods, modInfo)
	}

	// Create resolver with a registry
	registry := &modpackRegistry{pack: pack}
	resolver := dependencies.NewResolver(registry)

	return resolver.Resolve(mods)
}

// modpackRegistry implements dependencies.ModRegistry for a modpack.
type modpackRegistry struct {
	pack *sources.Modpack
}

func (r *modpackRegistry) GetMod(modID string, versionExpr string) (*dependencies.ModInfo, error) {
	for _, mod := range r.pack.Mods {
		if mod.ID == modID || mod.Name == modID {
			return &dependencies.ModInfo{
				ID:      mod.ID,
				Name:    mod.Name,
				Version: mod.Version,
			}, nil
		}
	}
	return nil, fmt.Errorf("mod not found: %s", modID)
}

func (r *modpackRegistry) GetAvailableVersions(modID string) ([]*dependencies.ModInfo, error) {
	for _, mod := range r.pack.Mods {
		if mod.ID == modID || mod.Name == modID {
			return []*dependencies.ModInfo{
				{
					ID:      mod.ID,
					Name:    mod.Name,
					Version: mod.Version,
				},
			}, nil
		}
	}
	return nil, fmt.Errorf("mod not found: %s", modID)
}

func (r *modpackRegistry) GetLatestVersion(modID string) (*dependencies.ModInfo, error) {
	return r.GetMod(modID, "*")
}

// printDependencyTree prints the dependency tree.
func printDependencyTree(result *dependencies.ResolutionResult) {
	fmt.Println("\n📊 Dependency tree:")
	for _, mod := range result.ResolvedMods {
		if mod.InstalledBy == "" {
			// Top-level mod
			fmt.Printf("   └── %s@%s\n", mod.ModInfo.ID, mod.ModInfo.Version)
		} else {
			fmt.Printf("       └── %s@%s (required by %s)\n",
				mod.ModInfo.ID, mod.ModInfo.Version, mod.InstalledBy)
		}
	}
}
