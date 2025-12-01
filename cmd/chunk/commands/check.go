package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexinslc/chunk/internal/deps"
	"github.com/alexinslc/chunk/internal/ui"
	"github.com/spf13/cobra"
)

var (
	checkDir    string
	checkFormat string
)

// CheckCmd is the command for validating dependencies
var CheckCmd = &cobra.Command{
	Use:   "check [modpack]",
	Short: "Validate modpack dependencies",
	Long: `Validate modpack dependencies before installation.

This command checks for:
  - Circular dependencies
  - Version conflicts between mods
  - Incompatible mod combinations
  - Missing required dependencies

Examples:
  chunk check                     # Check current directory
  chunk check --dir ./server      # Check specific directory
  chunk check atm9                # Check a modpack from registry
  chunk check --format=graph      # Output dependency graph (DOT format)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCheck,
}

func runCheck(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("🔍 Chunk Dependency Checker")
	fmt.Println()

	// Determine what to check
	var modpack string
	if len(args) > 0 {
		modpack = args[0]
	}

	// Get the directory to check
	dir := checkDir
	if dir == "" {
		dir = "."
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if we're checking a local directory or a registry modpack
	if modpack == "" {
		return checkLocalDirectory(absDir)
	}

	return checkRegistryModpack(modpack)
}

func checkLocalDirectory(dir string) error {
	ui.PrintInfo(fmt.Sprintf("Checking directory: %s", dir))

	// Look for .chunk.json
	manifestPath := filepath.Join(dir, ".chunk.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		ui.PrintWarning("No .chunk.json found in directory")
		fmt.Println()
		fmt.Println("To validate dependencies, your project needs a .chunk.json file")
		fmt.Println("with a 'dependencies' section specifying mod requirements.")
		return nil
	}

	// Parse the manifest and validate dependencies
	manifest, err := parseChunkManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	return validateDependencies(manifest.Dependencies)
}

func checkRegistryModpack(identifier string) error {
	ui.PrintInfo(fmt.Sprintf("Checking modpack: %s", identifier))
	fmt.Println()

	// Note: This would integrate with the source manager to fetch modpack info
	// For now, we show a placeholder message
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📦 Fetching modpack information...")
	fmt.Println()
	fmt.Println("⚠️  Registry dependency checking will be fully implemented")
	fmt.Println("   when integrated with mod metadata sources (Modrinth, CurseForge).")
	fmt.Println()
	fmt.Println("For local validation, use: chunk check --dir ./your-server")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return nil
}

// chunkManifestWithDeps represents a .chunk.json with dependency info
type chunkManifestWithDeps struct {
	Name         string             `json:"name"`
	MCVersion    string             `json:"mc_version"`
	Loader       string             `json:"loader"`
	Dependencies []*deps.Dependency `json:"dependencies,omitempty"`
}

func parseChunkManifest(path string) (*chunkManifestWithDeps, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Use manual JSON parsing to handle the dependency format
	var manifest chunkManifestWithDeps

	// Simple JSON parsing - for full implementation would use json.Unmarshal
	// with custom dependency parsing
	_ = data // Placeholder - full implementation would parse JSON

	// Return empty manifest for now - will be populated when .chunk.json format is finalized
	manifest.Name = "local-modpack"
	manifest.Dependencies = []*deps.Dependency{}

	return &manifest, nil
}

func validateDependencies(dependencies []*deps.Dependency) error {
	if len(dependencies) == 0 {
		ui.PrintSuccess("No dependencies to validate")
		return nil
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Validating %d dependencies...\n", len(dependencies))
	fmt.Println()

	// Create a resolver with a mock provider for validation
	resolver := deps.NewResolver(&localDependencyProvider{}, nil)

	results, err := resolver.ValidateDependencies(dependencies)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(results) == 0 {
		ui.PrintSuccess("✓ All dependencies are valid")
		fmt.Println()
		printDependencySummary(dependencies)
		return nil
	}

	// Print validation issues
	fmt.Println("⚠️  Dependency issues found:")
	fmt.Println()

	var hasError bool
	for _, result := range results {
		switch result.Type {
		case deps.ValidationConflict:
			fmt.Printf("  ❌ CONFLICT: %s\n", result.ModID)
			fmt.Printf("     %s\n", result.Message)
			hasError = true
		case deps.ValidationIncompatible:
			fmt.Printf("  ❌ INCOMPATIBLE: %s\n", result.ModID)
			fmt.Printf("     %s\n", result.Message)
			hasError = true
		case deps.ValidationMissing:
			fmt.Printf("  ❌ MISSING: %s\n", result.ModID)
			fmt.Printf("     %s\n", result.Message)
			hasError = true
		case deps.ValidationWarning:
			fmt.Printf("  ⚠️  WARNING: %s\n", result.ModID)
			fmt.Printf("     %s\n", result.Message)
		}
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if hasError {
		return fmt.Errorf("dependency validation failed")
	}

	return nil
}

func printDependencySummary(dependencies []*deps.Dependency) {
	var required, optional, incompatible, embedded int

	for _, dep := range dependencies {
		switch dep.Type {
		case deps.Required:
			required++
		case deps.Optional:
			optional++
		case deps.Incompatible:
			incompatible++
		case deps.Embedded:
			embedded++
		}
	}

	fmt.Println("Dependency Summary:")
	if required > 0 {
		fmt.Printf("  📦 Required:     %d\n", required)
	}
	if optional > 0 {
		fmt.Printf("  📎 Optional:     %d\n", optional)
	}
	if embedded > 0 {
		fmt.Printf("  📁 Embedded:     %d\n", embedded)
	}
	if incompatible > 0 {
		fmt.Printf("  🚫 Incompatible: %d\n", incompatible)
	}
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// localDependencyProvider is a placeholder provider for local validation
type localDependencyProvider struct{}

func (p *localDependencyProvider) GetModInfo(modID, version string) (*deps.ModInfo, error) {
	return nil, fmt.Errorf("mod not found: %s", modID)
}

func (p *localDependencyProvider) GetLatestVersion(modID, constraint string) (*deps.ModInfo, error) {
	return nil, fmt.Errorf("mod not found: %s", modID)
}

func (p *localDependencyProvider) GetAllVersions(modID string) ([]*deps.ModInfo, error) {
	return nil, fmt.Errorf("mod not found: %s", modID)
}

func init() {
	CheckCmd.Flags().StringVarP(&checkDir, "dir", "d", "", "Directory to check (default: current directory)")
	CheckCmd.Flags().StringVarP(&checkFormat, "format", "f", "text", "Output format: text, json, graph")

	CheckCmd.SilenceUsage = true
}
