package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexinslc/chunk/internal/bench"
	"github.com/spf13/cobra"
)

var BenchCmd = &cobra.Command{
	Use:   "bench [user/repo] [url]",
	Short: "Manage recipe benches",
	Long: `Manage recipe benches (repositories containing recipes).

Benches are Git repositories that contain modpack recipes in a Recipes/ directory.
Similar to Homebrew taps, benches allow you to add custom recipe sources.

Examples:
  chunk bench                        # List all benches
  chunk bench usechunk/recipes       # Add bench from GitHub
  chunk bench myuser/repo git@github.com:myuser/repo.git  # Add with custom URL
  chunk bench remove usechunk/recipes  # Remove a bench
  chunk bench info usechunk/recipes    # Show bench details

Benches are stored in ~/.chunk/Benches/`,
	RunE: runBench,
}

func runBench(cmd *cobra.Command, args []string) error {
	// No arguments - list all benches
	if len(args) == 0 {
		return listBenches()
	}

	// Check for subcommands
	if args[0] == "remove" {
		if len(args) < 2 {
			return fmt.Errorf("bench name required for remove")
		}
		return removeBench(args[1])
	}

	if args[0] == "info" {
		if len(args) < 2 {
			return fmt.Errorf("bench name required for info")
		}
		return infoBench(args[1])
	}

	// Otherwise, treat as add operation
	name := args[0]
	url := ""
	if len(args) > 1 {
		url = args[1]
	}

	return addBench(name, url)
}

func listBenches() error {
	manager, err := bench.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize bench manager: %w", err)
	}

	benches := manager.List()

	if len(benches) == 0 {
		fmt.Println()
		fmt.Println("No benches installed.")
		fmt.Println()
		fmt.Println("Add a bench with:")
		fmt.Println("  chunk bench usechunk/recipes")
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Printf("📚 Installed Benches (%d)\n", len(benches))
	fmt.Println()

	for _, b := range benches {
		fmt.Printf("  %s\n", b.Name)
		fmt.Printf("    URL:   %s\n", b.URL)
		fmt.Printf("    Path:  %s\n", b.Path)
		fmt.Printf("    Added: %s\n", b.Added.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	return nil
}

func addBench(name string, url string) error {
	manager, err := bench.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize bench manager: %w", err)
	}

	fmt.Println()
	fmt.Printf("🔄 Adding bench: %s\n", name)
	if url != "" {
		fmt.Printf("   URL: %s\n", url)
	} else {
		normalizedURL := bench.NormalizeGitHubURL(name)
		fmt.Printf("   URL: %s\n", normalizedURL)
	}
	fmt.Println()

	if err := manager.Add(name, url); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("✅ Bench '%s' added successfully!\n", name)
	fmt.Println()

	return nil
}

func removeBench(name string) error {
	manager, err := bench.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize bench manager: %w", err)
	}

	fmt.Println()
	fmt.Printf("🗑️  Removing bench: %s\n", name)
	fmt.Println()

	if err := manager.Remove(name); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("✅ Bench '%s' removed successfully!\n", name)
	fmt.Println()

	return nil
}

func infoBench(name string) error {
	manager, err := bench.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize bench manager: %w", err)
	}

	b, err := manager.Get(name)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📚 Bench: %s\n", b.Name)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Printf("  URL:   %s\n", b.URL)
	fmt.Printf("  Path:  %s\n", b.Path)
	fmt.Printf("  Added: %s\n", b.Added.Format("2006-01-02 15:04:05"))
	fmt.Println()

	// Count recipes
	recipesDir := strings.TrimPrefix(b.Path, "~")
	if strings.HasPrefix(recipesDir, "/") {
		recipesPath := fmt.Sprintf("%s/Recipes", b.Path)
		if entries, err := os.ReadDir(recipesPath); err == nil {
			recipeCount := 0
			for _, entry := range entries {
				if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml")) {
					recipeCount++
				}
			}
			if recipeCount > 0 {
				fmt.Printf("  Recipes: %d\n", recipeCount)
				fmt.Println()
			}
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}

func init() {
	// Suppress usage printing on errors
	BenchCmd.SilenceUsage = true
	BenchCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		cmd.Usage()
		return err
	})
}
