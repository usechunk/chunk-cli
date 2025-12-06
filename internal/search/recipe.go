package search

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Recipe represents a modpack recipe from a bench
type Recipe struct {
	// File metadata
	FilePath string `json:"-" yaml:"-"` // Not exported in JSON/YAML
	BenchName string `json:"-" yaml:"-"` // Not exported in JSON/YAML
	Slug     string `json:"slug,omitempty" yaml:"slug,omitempty"` // Recipe filename without extension
	
	// Recipe metadata
	Name             string   `json:"name" yaml:"name"`
	Description      string   `json:"description,omitempty" yaml:"description,omitempty"`
	MCVersion        string   `json:"mc_version" yaml:"mc_version"`
	Loader           string   `json:"loader" yaml:"loader"`
	LoaderVersion    string   `json:"loader_version,omitempty" yaml:"loader_version,omitempty"`
	RecommendedRAMGB int      `json:"recommended_ram_gb,omitempty" yaml:"recommended_ram_gb,omitempty"`
	JavaVersion      int      `json:"java_version,omitempty" yaml:"java_version,omitempty"`
	Tags             []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Author           string   `json:"author,omitempty" yaml:"author,omitempty"`
	Homepage         string   `json:"homepage,omitempty" yaml:"homepage,omitempty"`
}

// SearchResult represents a recipe match with relevance score
type SearchResult struct {
	Recipe   *Recipe
	Score    int    // Higher score = better match
	MatchedField string // Field that matched the query
}

// LoadRecipe loads a recipe from a JSON or YAML file
func LoadRecipe(path string, benchName string) (*Recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read recipe file: %w", err)
	}

	var recipe Recipe
	ext := strings.ToLower(filepath.Ext(path))
	
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &recipe); err != nil {
			return nil, fmt.Errorf("failed to parse JSON recipe: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &recipe); err != nil {
			return nil, fmt.Errorf("failed to parse YAML recipe: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}

	// Set metadata
	recipe.FilePath = path
	recipe.BenchName = benchName
	
	// Generate slug from filename if not provided
	if recipe.Slug == "" {
		fileName := filepath.Base(path)
		recipe.Slug = strings.TrimSuffix(fileName, filepath.Ext(fileName))
	}

	return &recipe, nil
}

// LoadRecipesFromBench loads all recipes from a bench's Recipes directory
func LoadRecipesFromBench(benchPath string, benchName string) ([]*Recipe, error) {
	recipesDir := filepath.Join(benchPath, "Recipes")
	
	// Check if Recipes directory exists
	if _, err := os.Stat(recipesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("Recipes directory not found: %s", recipesDir)
	}

	entries, err := os.ReadDir(recipesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read Recipes directory: %w", err)
	}

	var recipes []*Recipe
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process JSON, YAML, and YML files
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			continue
		}

		filePath := filepath.Join(recipesDir, entry.Name())
		recipe, err := LoadRecipe(filePath, benchName)
		if err != nil {
			// Skip files that can't be parsed, but don't fail completely
			continue
		}

		recipes = append(recipes, recipe)
	}

	return recipes, nil
}
