package search

import (
	"sort"
	"strings"

	"github.com/alexinslc/chunk/internal/bench"
	"github.com/alexinslc/chunk/internal/config"
)

// Searcher handles recipe searching across benches
type Searcher struct {
	manager *bench.Manager
}

// NewSearcher creates a new recipe searcher
func NewSearcher() (*Searcher, error) {
	manager, err := bench.NewManager()
	if err != nil {
		return nil, err
	}
	return &Searcher{manager: manager}, nil
}

// Search searches for recipes matching the query across all benches
// If benchFilter is not empty, only search in that specific bench
func (s *Searcher) Search(query string, benchFilter string) ([]*SearchResult, error) {
	benches := s.manager.List()

	// Filter to specific bench if requested
	if benchFilter != "" {
		filtered := []config.Bench{}
		for _, b := range benches {
			if b.Name == benchFilter {
				filtered = append(filtered, b)
				break
			}
		}
		benches = filtered
	}

	var allResults []*SearchResult

	// Load and search recipes from each bench
	for _, bench := range benches {
		recipes, err := LoadRecipesFromBench(bench.Path, bench.Name)
		if err != nil {
			// Skip benches that fail to load
			continue
		}

		for _, recipe := range recipes {
			if result := matchRecipe(recipe, query); result != nil {
				allResults = append(allResults, result)
			}
		}
	}

	// Sort results by score (highest first)
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	return allResults, nil
}

// matchRecipe checks if a recipe matches the query and returns a SearchResult
func matchRecipe(recipe *Recipe, query string) *SearchResult {
	query = strings.ToLower(query)
	score := 0
	matchedField := ""

	// Check name (highest priority)
	if matchScore := fuzzyMatch(strings.ToLower(recipe.Name), query); matchScore > 0 {
		score += matchScore * 10
		if matchedField == "" {
			matchedField = "name"
		}
	}

	// Check slug (high priority)
	if matchScore := fuzzyMatch(strings.ToLower(recipe.Slug), query); matchScore > 0 {
		score += matchScore * 8
		if matchedField == "" {
			matchedField = "slug"
		}
	}

	// Check description (medium priority)
	if matchScore := fuzzyMatch(strings.ToLower(recipe.Description), query); matchScore > 0 {
		score += matchScore * 3
		if matchedField == "" {
			matchedField = "description"
		}
	}

	// Check tags (medium priority)
	for _, tag := range recipe.Tags {
		if matchScore := fuzzyMatch(strings.ToLower(tag), query); matchScore > 0 {
			score += matchScore * 5
			if matchedField == "" {
				matchedField = "tags"
			}
		}
	}

	// Check author (low priority)
	if matchScore := fuzzyMatch(strings.ToLower(recipe.Author), query); matchScore > 0 {
		score += matchScore * 2
		if matchedField == "" {
			matchedField = "author"
		}
	}

	// Only return results with positive score
	if score > 0 {
		return &SearchResult{
			Recipe:       recipe,
			Score:        score,
			MatchedField: matchedField,
		}
	}

	return nil
}

// fuzzyMatch performs case-insensitive fuzzy matching
// Returns a score based on match quality:
// - 100 for exact match
// - 80 for exact word match
// - 50 for prefix match
// - 30 for contains match
// - 0 for no match
func fuzzyMatch(text, query string) int {
	if text == "" || query == "" {
		return 0
	}

	// Exact match
	if text == query {
		return 100
	}

	// Exact word match (query is a complete word in text)
	words := strings.Fields(text)
	for _, word := range words {
		if word == query {
			return 80
		}
	}

	// Prefix match
	if strings.HasPrefix(text, query) {
		return 50
	}

	// Contains match
	if strings.Contains(text, query) {
		return 30
	}

	// No match
	return 0
}
