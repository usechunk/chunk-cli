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

	// Get bench manager for checking outdated status
	var benchManager *bench.Manager
	if checkOutdated {
		var err error
		benchManager, err = bench.NewManager()
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
		if checkOutdated && benchManager != nil {
			if isOutdated, latestVersion := checkIfOutdated(inst, benchManager); isOutdated {
				fmt.Printf(" [outdated: %s available]", latestVersion)
			}
		}

		fmt.Println()

		// Display installation path
		fmt.Printf("  Installed: %s\n", inst.Path)

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

// checkIfOutdated checks if an installation has a newer version available
func checkIfOutdated(inst *tracking.Installation, benchManager *bench.Manager) (bool, string) {
	if benchManager == nil {
		return false, ""
	}

	benches := benchManager.List()

	// Find the recipe in benches
	for _, b := range benches {
		// Check if this is the bench the installation came from
		if inst.Bench != "" && b.Name != inst.Bench {
			continue
		}

		recipes, err := search.LoadRecipesFromBench(b.Path, b.Name)
		if err != nil {
			continue
		}

		for _, recipe := range recipes {
			// Match by slug
			if strings.EqualFold(recipe.Slug, inst.Slug) {
				// Compare versions
				if recipe.Version != "" && recipe.Version != inst.Version {
					return true, recipe.Version
				}
			}
		}
	}

	return false, ""
}
