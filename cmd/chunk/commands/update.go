package commands

import (
	"fmt"
	"os"

	"github.com/alexinslc/chunk/internal/bench"
	"github.com/spf13/cobra"
)

var (
	updateBenchFlag string
)

var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update recipe benches",
	Long: `Update recipe benches by pulling the latest changes from Git.

Similar to 'brew update', this command updates all installed benches
or a specific bench if the --bench flag is provided.

Examples:
  chunk update                          # Update all benches
  chunk update --bench usechunk/recipes # Update specific bench`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate()
	},
}

func init() {
	UpdateCmd.Flags().StringVarP(&updateBenchFlag, "bench", "b", "", "Update only a specific bench (user/repo)")
	UpdateCmd.SilenceUsage = true
	UpdateCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		cmd.Usage()
		return err
	})
}

func runUpdate() error {
	manager, err := bench.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize bench manager: %w", err)
	}

	fmt.Println()
	fmt.Println("==> Updating benches")
	fmt.Println()

	var results []*bench.UpdateResult

	if updateBenchFlag != "" {
		// Update specific bench
		result, err := manager.Update(updateBenchFlag)
		if err != nil {
			return err
		}
		results = []*bench.UpdateResult{result}
	} else {
		// Update all benches
		allResults, err := manager.UpdateAll()
		if err != nil {
			return err
		}
		results = allResults
	}

	// Display results
	updatedCount := 0
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("==> %s\n", result.BenchName)
			fmt.Printf("  ⚠️  Error: %v\n", result.Error)
			fmt.Println()
			continue
		}

		if result.AlreadyUpToDate {
			fmt.Printf("==> %s\n", result.BenchName)
			fmt.Println("  Already up to date.")
			fmt.Println()
			continue
		}

		// Bench was updated
		updatedCount++
		fmt.Printf("==> %s\n", result.BenchName)

		// Show updated recipes
		for _, update := range result.UpdatedRecipes {
			if update.OldVersion != "" && update.NewVersion != "" {
				fmt.Printf("  Updated %s (%s → %s)\n", update.Name, update.OldVersion, update.NewVersion)
			} else {
				fmt.Printf("  Updated %s\n", update.Name)
			}
		}

		// Show new recipes
		for _, newRecipe := range result.NewRecipes {
			fmt.Printf("  New recipe: %s\n", newRecipe)
		}

		// Show removed recipes
		for _, removedRecipe := range result.RemovedRecipes {
			fmt.Printf("  Removed: %s\n", removedRecipe)
		}

		// If no changes were detected but update was successful
		if len(result.UpdatedRecipes) == 0 && len(result.NewRecipes) == 0 && len(result.RemovedRecipes) == 0 {
			fmt.Println("  Updated (no recipe changes)")
		}

		fmt.Println()
	}

	// Summary
	if updatedCount > 0 {
		fmt.Printf("==> Updated %d bench(es)\n", updatedCount)
	} else {
		fmt.Println("==> All benches are up to date")
	}
	fmt.Println()

	return nil
}
