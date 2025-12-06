package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexinslc/chunk/internal/bench"
	"github.com/spf13/cobra"
)

var BenchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Manage recipe benches",
	Long: `Manage recipe benches (repositories containing recipes).

Benches are Git repositories that contain modpack recipes in a Recipes/ directory.
Similar to Homebrew taps, benches allow you to add custom recipe sources.

Benches are stored in ~/.chunk/Benches/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listBenches()
	},
}

var benchAddCmd = &cobra.Command{
	Use:   "add <user/repo> [url]",
	Short: "Add a new bench",
	Long: `Add a new bench from GitHub or a custom URL.

Examples:
  chunk bench add usechunk/recipes                      # Add from GitHub
  chunk bench add myrepo git@github.com:user/repo.git  # Add with custom URL`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		url := ""
		if len(args) > 1 {
			url = args[1]
		}
		return addBench(name, url)
	},
}

var benchRemoveCmd = &cobra.Command{
	Use:   "remove <user/repo>",
	Short: "Remove a bench",
	Long: `Remove an installed bench.

Example:
  chunk bench remove usechunk/recipes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return removeBench(args[0])
	},
}

var benchInfoCmd = &cobra.Command{
	Use:   "info <user/repo>",
	Short: "Show bench details",
	Long: `Show detailed information about an installed bench.

Example:
  chunk bench info usechunk/recipes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return infoBench(args[0])
	},
}

var benchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all benches",
	Long:  `List all installed benches.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listBenches()
	},
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
	fmt.Println("   Cloning repository...")

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
	recipesPath := filepath.Join(b.Path, "Recipes")
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

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}

func init() {
	// Add subcommands
	BenchCmd.AddCommand(benchAddCmd)
	BenchCmd.AddCommand(benchRemoveCmd)
	BenchCmd.AddCommand(benchInfoCmd)
	BenchCmd.AddCommand(benchListCmd)

	// Suppress usage printing on errors
	BenchCmd.SilenceUsage = true
	BenchCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		cmd.Usage()
		return err
	})
}
