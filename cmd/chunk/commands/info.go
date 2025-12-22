package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alexinslc/chunk/internal/bench"
	"github.com/alexinslc/chunk/internal/search"
	"github.com/spf13/cobra"
)

var InfoCmd = &cobra.Command{
	Use:   "info <recipe>",
	Short: "Show detailed information about a recipe",
	Long: `Show detailed information about a recipe from local benches.

Works offline by reading recipe files from installed benches.

Examples:
  chunk info all-the-mods-9
  chunk info atm9 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		recipeName := args[0]

		// Find the recipe
		recipe, err := findRecipe(recipeName)
		if err != nil {
			return err
		}

		// Display the info
		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			return displayRecipeInfoJSON(recipe)
		}
		return displayRecipeInfo(recipe)
	},
}

func init() {
	InfoCmd.Flags().Bool("json", false, "Output in JSON format")
}

// findRecipe searches for a recipe across all benches
func findRecipe(recipeName string) (*search.Recipe, error) {
	manager, err := bench.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize bench manager: %w", err)
	}

	benches := manager.List()
	if len(benches) == 0 {
		return nil, fmt.Errorf("no benches installed. Run 'chunk bench add usechunk/recipes' to add recipes")
	}

	// Try to find the recipe in any bench
	for _, b := range benches {
		recipes, err := search.LoadRecipesFromBench(b.Path, b.Name)
		if err != nil {
			continue
		}

		for _, recipe := range recipes {
			// Match by slug or name (case-insensitive)
			if strings.EqualFold(recipe.Slug, recipeName) ||
				strings.EqualFold(recipe.Name, recipeName) {
				return recipe, nil
			}
		}
	}

	return nil, fmt.Errorf("recipe '%s' not found in any bench. Run 'chunk search %s' to find similar recipes", recipeName, recipeName)
}

// displayRecipeInfo displays recipe information in human-readable format
func displayRecipeInfo(recipe *search.Recipe) error {
	fmt.Println()

	// Header with name and description
	headerText := recipe.Slug
	if recipe.Description != "" {
		headerText = fmt.Sprintf("%s: %s", recipe.Slug, recipe.Description)
	}
	fmt.Printf("==> %s\n", headerText)

	// Homepage if available
	if recipe.Homepage != "" {
		fmt.Println(recipe.Homepage)
	}
	fmt.Println()

	// Version info
	if recipe.Version != "" {
		fmt.Printf("Version: %s\n", recipe.Version)
	}

	// Minecraft and loader info
	fmt.Printf("Minecraft: %s\n", recipe.MCVersion)
	loaderText := capitalize(recipe.Loader)
	if recipe.LoaderVersion != "" {
		loaderText = fmt.Sprintf("%s %s", loaderText, recipe.LoaderVersion)
	}
	fmt.Printf("Loader: %s\n", loaderText)

	// License
	if recipe.License != "" {
		fmt.Printf("License: %s\n", recipe.License)
	}
	fmt.Println()

	// Requirements section
	hasRequirements := recipe.RecommendedRAMGB > 0 || recipe.DiskSpaceGB > 0 || recipe.JavaVersion > 0
	if hasRequirements {
		fmt.Println("Requirements:")
		if recipe.RecommendedRAMGB > 0 {
			fmt.Printf("  RAM: %d GB\n", recipe.RecommendedRAMGB)
		}
		if recipe.DiskSpaceGB > 0 {
			fmt.Printf("  Disk: %d GB\n", recipe.DiskSpaceGB)
		}
		if recipe.JavaVersion > 0 {
			fmt.Printf("  Java: %d\n", recipe.JavaVersion)
		}
		fmt.Println()
	}

	// Download info
	if recipe.DownloadURL != "" {
		fmt.Printf("Download URL: %s\n", recipe.DownloadURL)
		if recipe.DownloadSizeMB > 0 {
			fmt.Printf("Download Size: %d MB\n", recipe.DownloadSizeMB)
		}
		fmt.Println()
	}

	// Source bench
	fmt.Printf("From: %s\n", recipe.BenchName)
	fmt.Println()

	return nil
}

// displayRecipeInfoJSON displays recipe information in JSON format
func displayRecipeInfoJSON(recipe *search.Recipe) error {
	// Create a custom output structure for JSON
	output := map[string]interface{}{
		"slug":        recipe.Slug,
		"name":        recipe.Name,
		"description": recipe.Description,
		"version":     recipe.Version,
		"mc_version":  recipe.MCVersion,
		"loader":      recipe.Loader,
		"bench_name":  recipe.BenchName,
	}

	// Add optional fields only if they're set
	if recipe.LoaderVersion != "" {
		output["loader_version"] = recipe.LoaderVersion
	}
	if recipe.Homepage != "" {
		output["homepage"] = recipe.Homepage
	}
	if recipe.License != "" {
		output["license"] = recipe.License
	}
	if recipe.RecommendedRAMGB > 0 {
		output["recommended_ram_gb"] = recipe.RecommendedRAMGB
	}
	if recipe.DiskSpaceGB > 0 {
		output["disk_space_gb"] = recipe.DiskSpaceGB
	}
	if recipe.JavaVersion > 0 {
		output["java_version"] = recipe.JavaVersion
	}
	if recipe.DownloadURL != "" {
		output["download_url"] = recipe.DownloadURL
	}
	if recipe.DownloadSizeMB > 0 {
		output["download_size_mb"] = recipe.DownloadSizeMB
	}
	if recipe.Author != "" {
		output["author"] = recipe.Author
	}
	if len(recipe.Tags) > 0 {
		output["tags"] = recipe.Tags
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}
