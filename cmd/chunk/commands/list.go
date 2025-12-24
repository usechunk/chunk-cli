package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexinslc/chunk/internal/bench"
	"github.com/alexinslc/chunk/internal/search"
	"github.com/alexinslc/chunk/internal/tracking"
	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed modpacks",
	Long: `List all installed modpacks tracked in ~/.chunk/installed.json.

Shows installation details including version, path, and source bench.

Examples:
  chunk list                # Show all installed modpacks
  chunk list --paths        # Show only installation paths
  chunk list --json         # Output in JSON format
  chunk list --outdated     # Show which have updates available`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pathsOnly, _ := cmd.Flags().GetBool("paths")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		outdated, _ := cmd.Flags().GetBool("outdated")

		tracker, err := tracking.NewTracker()
		if err != nil {
			return fmt.Errorf("failed to initialize tracker: %w", err)
		}

		installations, err := tracker.ListInstallations()
		if err != nil {
			return fmt.Errorf("failed to list installations: %w", err)
		}

		// Handle empty state
		if len(installations) == 0 {
			if jsonOutput {
				fmt.Println("[]")
				return nil
			}
			fmt.Println()
			fmt.Println("No modpacks installed yet.")
			fmt.Println()
			fmt.Println("Install a modpack with:")
			fmt.Println("  chunk install <recipe>")
			fmt.Println()
			fmt.Println("Search for recipes with:")
			fmt.Println("  chunk search <query>")
			fmt.Println()
			return nil
		}

		// Handle different output formats
		if pathsOnly {
			return displayPaths(installations)
		}

		if jsonOutput {
			return displayJSON(installations)
		}

		return displayList(installations, outdated)
	},
}

func init() {
	ListCmd.Flags().Bool("paths", false, "Show only installation paths")
	ListCmd.Flags().Bool("json", false, "Output in JSON format")
	ListCmd.Flags().Bool("outdated", false, "Show which installations have updates available")
}

// displayPaths shows only installation paths
func displayPaths(installations []*tracking.Installation) error {
	for _, inst := range installations {
		fmt.Println(inst.Path)
	}
	return nil
}

// displayJSON outputs installations in JSON format
func displayJSON(installations []*tracking.Installation) error {
	jsonData, err := json.MarshalIndent(installations, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

// displayList shows formatted list of installations
func displayList(installations []*tracking.Installation, checkOutdated bool) error {
	fmt.Println()
	fmt.Printf("==> Installed modpacks (%d)\n", len(installations))
	fmt.Println()

	// Pre-load all recipes once if checking for outdated
	var recipeCache map[string]*search.Recipe
	if checkOutdated {
		var err error
		recipeCache, err = loadAllRecipes()
		if err != nil {
			// Don't fail completely, just skip outdated check
			fmt.Printf("Warning: Could not check for updates: %v\n", err)
			checkOutdated = false
		}
	}

	for _, inst := range installations {
		// Display slug and version
		fmt.Printf("%s (%s)", inst.Slug, inst.Version)

		// Check if outdated
		if checkOutdated && recipeCache != nil {
			if isOutdated, latestVersion := isInstallationOutdated(inst, recipeCache); isOutdated {
				fmt.Printf(" [outdated: %s available]", latestVersion)
			}
		}

		fmt.Println()

		// Display installation path
		fmt.Printf("  Path: %s\n", inst.Path)

		// Display bench source
		if inst.Bench != "" {
			fmt.Printf("  From: %s\n", inst.Bench)
		}

		// Display installed time
		if !inst.InstalledAt.IsZero() {
			fmt.Printf("  Installed: %s\n", formatRelativeTime(inst.InstalledAt))
		}

		fmt.Println()
	}

	return nil
}

// formatRelativeTime formats a timestamp as relative time (e.g., "2 days ago")
func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)

	seconds := int(duration.Seconds())
	minutes := int(duration.Minutes())
	hours := int(duration.Hours())
	days := hours / 24
	weeks := days / 7
	months := days / 30
	years := days / 365

	switch {
	case seconds < 60:
		if seconds == 1 {
			return "1 second ago"
		}
		return fmt.Sprintf("%d seconds ago", seconds)
	case minutes < 60:
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case hours < 24:
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case days < 7:
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case days < 30:
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case days < 365:
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// loadAllRecipes loads all recipes from all benches and caches them by slug
// Returns a map where the key is "bench:slug" to handle same slug in different benches
func loadAllRecipes() (map[string]*search.Recipe, error) {
	manager, err := bench.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize bench manager: %w", err)
	}

	benches := manager.List()
	recipeCache := make(map[string]*search.Recipe)

	for _, b := range benches {
		recipes, err := search.LoadRecipesFromBench(b.Path, b.Name)
		if err != nil {
			continue
		}

		for _, recipe := range recipes {
			// Store with both bench-specific and generic keys
			// Bench-specific key for when bench is known
			benchKey := fmt.Sprintf("%s:%s", b.Name, strings.ToLower(recipe.Slug))
			recipeCache[benchKey] = recipe

			// Generic key only if not already set (first bench wins for generic lookup)
			genericKey := strings.ToLower(recipe.Slug)
			if _, exists := recipeCache[genericKey]; !exists {
				recipeCache[genericKey] = recipe
			}
		}
	}

	return recipeCache, nil
}

// isInstallationOutdated checks if an installation has a newer version available
// Uses pre-loaded recipe cache for O(1) lookups
func isInstallationOutdated(inst *tracking.Installation, recipeCache map[string]*search.Recipe) (bool, string) {
	if recipeCache == nil {
		return false, ""
	}

	var recipe *search.Recipe

	// First try to find by bench-specific key if bench is known
	if inst.Bench != "" {
		benchKey := fmt.Sprintf("%s:%s", inst.Bench, strings.ToLower(inst.Slug))
		recipe = recipeCache[benchKey]
	}

	// Fall back to generic key if not found or bench not specified
	if recipe == nil {
		genericKey := strings.ToLower(inst.Slug)
		recipe = recipeCache[genericKey]
	}

	// No recipe found
	if recipe == nil {
		return false, ""
	}

	// Compare versions with proper trimming
	installedVersion := strings.TrimSpace(inst.Version)
	recipeVersion := strings.TrimSpace(recipe.Version)

	// If recipe has no version info, can't determine if outdated
	if recipeVersion == "" {
		return false, ""
	}

	// Simple string comparison after trimming
	// Note: This doesn't use semantic versioning, but handles basic cases
	// For proper semver, would need a library like github.com/Masterminds/semver
	if recipeVersion != installedVersion {
		return true, recipeVersion
	}

	return false, ""
}
