package commands

import (
	"fmt"
	"strings"

	"github.com/alexinslc/chunk/internal/search"
	"github.com/spf13/cobra"
)

var (
	searchBench string
)

var SearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for modpacks in local benches",
	Long: `Search for modpacks across all installed recipe benches.

Searches local recipe files for matches in:
  - Recipe name
  - Slug
  - Description
  - Tags
  - Author

Examples:
  chunk search "all the mods"
  chunk search atm
  chunk search fabric
  chunk search atm --bench usechunk/recipes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		searcher, err := search.NewSearcher()
		if err != nil {
			return fmt.Errorf("failed to initialize search: %w", err)
		}

		results, err := searcher.Search(query, searchBench)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		if len(results) == 0 {
			fmt.Println()
			fmt.Printf("No recipes found matching '%s'\n", query)
			fmt.Println()
			fmt.Println("Try:")
			fmt.Println("  - Different search terms")
			fmt.Println("  - Run 'chunk bench list' to see installed benches")
			fmt.Println("  - Run 'chunk bench add usechunk/recipes' to add more recipes")
			fmt.Println()
			return nil
		}

		// Display results
		fmt.Println()
		fmt.Printf("==> Found %d recipe(s)\n", len(results))
		fmt.Println()

		for _, result := range results {
			r := result.Recipe

			// Recipe name and bench
			fmt.Printf("%s (%s)\n", r.Slug, r.BenchName)

			// Description (if available)
			if r.Description != "" {
				fmt.Printf("  %s\n", r.Description)
			}

			// Metadata
			ramInfo := ""
			if r.RecommendedRAMGB > 0 {
				ramInfo = fmt.Sprintf(" | %dGB RAM", r.RecommendedRAMGB)
			}
			fmt.Printf("  MC %s | %s", r.MCVersion, capitalize(r.Loader))
			if r.LoaderVersion != "" {
				fmt.Printf(" %s", r.LoaderVersion)
			}
			fmt.Printf("%s\n", ramInfo)

			fmt.Println()
		}

		return nil
	},
}

func init() {
	SearchCmd.Flags().StringVar(&searchBench, "bench", "", "Limit search to specific bench (e.g., usechunk/recipes)")
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
