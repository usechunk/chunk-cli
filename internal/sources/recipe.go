package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexinslc/chunk/internal/bench"
	"github.com/alexinslc/chunk/internal/checksum"
	"github.com/alexinslc/chunk/internal/config"
	"github.com/alexinslc/chunk/internal/search"
)

// RecipeClient handles fetching modpacks from local recipe benches
type RecipeClient struct {
	httpClient *http.Client
	manager    *bench.Manager
}

// ParseRecipeIdentifier splits an identifier into bench name and recipe name
// Supports formats:
//   - "recipe" -> ("", "recipe")
//   - "bench::recipe" -> ("bench", "recipe")
func ParseRecipeIdentifier(identifier string) (benchName, recipeName string) {
	if strings.Contains(identifier, "::") {
		parts := strings.SplitN(identifier, "::", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}
	return "", identifier
}

// NewRecipeClient creates a new recipe client
func NewRecipeClient() *RecipeClient {
	manager, err := bench.NewManager()
	if err != nil {
		// If bench manager fails to initialize, we'll still create the client
		// but findRecipe will return an appropriate error when called
		manager = nil
	}
	return &RecipeClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
		manager: manager,
	}
}

// Fetch fetches a modpack from a recipe
// Supports formats:
//   - "atm9" - searches all benches (core bench first)
//   - "usechunk/recipes::atm9" - forces specific bench
func (c *RecipeClient) Fetch(identifier string) (*Modpack, error) {
	// Parse identifier to extract bench and recipe name
	benchName, recipeName := ParseRecipeIdentifier(identifier)

	// Find the recipe
	recipe, err := c.FindRecipe(recipeName, benchName)
	if err != nil {
		return nil, err
	}

	// Validate recipe has download URL
	if recipe.DownloadURL == "" {
		return nil, fmt.Errorf("recipe \"%s\" does not have a download_url", recipe.Slug)
	}

	// Convert recipe to Modpack
	modpack, err := c.recipeToModpack(recipe)
	if err != nil {
		return nil, fmt.Errorf("failed to convert recipe: %w", err)
	}

	return modpack, nil
}

// Search searches for recipes in local benches
func (c *RecipeClient) Search(query string) ([]*ModpackSearchResult, error) {
	searcher, err := search.NewSearcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create searcher: %w", err)
	}

	results, err := searcher.Search(query, "")
	if err != nil {
		return nil, err
	}

	// Convert search results to ModpackSearchResults
	var modpackResults []*ModpackSearchResult
	for _, result := range results {
		r := result.Recipe
		modpackResults = append(modpackResults, &ModpackSearchResult{
			Name:        r.Name,
			Identifier:  r.Slug,
			Description: r.Description,
			MCVersion:   r.MCVersion,
			Loader:      LoaderType(r.Loader),
			Source:      "recipe",
		})
	}

	return modpackResults, nil
}

// GetVersions returns available versions for a recipe
func (c *RecipeClient) GetVersions(identifier string) ([]*Version, error) {
	// Parse identifier
	benchName, recipeName := ParseRecipeIdentifier(identifier)

	// Find the recipe
	recipe, err := c.FindRecipe(recipeName, benchName)
	if err != nil {
		return nil, err
	}

	// Recipes currently have a single version
	version := &Version{
		Version:     recipe.Version,
		MCVersion:   recipe.MCVersion,
		Loader:      LoaderType(recipe.Loader),
		ReleaseDate: "",
		IsStable:    true,
		DownloadURL: recipe.DownloadURL,
		SHA256:      recipe.SHA256,
	}

	if version.Version == "" {
		version.Version = "latest"
	}

	return []*Version{version}, nil
}

// FindRecipe searches for a recipe in local benches
func (c *RecipeClient) FindRecipe(recipeName string, benchFilter string) (*search.Recipe, error) {
	if c.manager == nil {
		return nil, fmt.Errorf("bench manager not initialized")
	}

	benches := c.manager.List()
	if len(benches) == 0 {
		return nil, fmt.Errorf("no benches installed. Add a bench with: chunk bench add usechunk/recipes")
	}

	// Filter to specific bench if requested
	var searchBenches []config.Bench
	if benchFilter != "" {
		for _, b := range benches {
			if b.Name == benchFilter {
				searchBenches = []config.Bench{b}
				break
			}
		}
		if len(searchBenches) == 0 {
			return nil, fmt.Errorf("bench \"%s\" not found", benchFilter)
		}
	} else {
		searchBenches = benches
	}

	// Search for recipe in benches (load recipes once per bench)
	for _, bench := range searchBenches {
		recipes, err := search.LoadRecipesFromBench(bench.Path, bench.Name)
		if err != nil {
			// Skip benches that fail to load
			continue
		}

		// Look for exact slug match first, then name match
		for _, recipe := range recipes {
			if recipe.Slug == recipeName {
				return recipe, nil
			}
		}
		
		// If no slug match, try name match (case-insensitive)
		for _, recipe := range recipes {
			if strings.EqualFold(recipe.Name, recipeName) {
				return recipe, nil
			}
		}
	}

	return nil, fmt.Errorf("recipe \"%s\" not found in installed benches", recipeName)
}

// recipeToModpack converts a Recipe to a Modpack
func (c *RecipeClient) recipeToModpack(recipe *search.Recipe) (*Modpack, error) {
	// Parse loader type
	var loaderType LoaderType
	switch strings.ToLower(recipe.Loader) {
	case "forge":
		loaderType = LoaderForge
	case "fabric":
		loaderType = LoaderFabric
	case "neoforge":
		loaderType = LoaderNeoForge
	default:
		return nil, fmt.Errorf("unsupported loader type: %s", recipe.Loader)
	}

	modpack := &Modpack{
		Name:           recipe.Name,
		Identifier:     recipe.Slug,
		Description:    recipe.Description,
		MCVersion:      recipe.MCVersion,
		Loader:         loaderType,
		LoaderVersion:  recipe.LoaderVersion,
		Author:         recipe.Author,
		Source:         fmt.Sprintf("recipe:%s", recipe.BenchName),
		Mods:           []*Mod{}, // Recipes don't include individual mod lists
		Dependencies:   []string{},
		RecommendedRAM: recipe.RecommendedRAMGB,
		ManifestURL:    recipe.DownloadURL,
	}

	return modpack, nil
}

// DownloadFile downloads a file from the recipe's download URL with progress
func (c *RecipeClient) DownloadFile(downloadURL string, dest io.Writer, progressCallback func(downloaded, total int64)) error {
	resp, err := c.httpClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	// Get total size
	totalSize := resp.ContentLength

	// Create a progress reader
	var downloaded int64
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			_, writeErr := dest.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write: %w", writeErr)
			}
			downloaded += int64(n)
			if progressCallback != nil {
				progressCallback(downloaded, totalSize)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read: %w", err)
		}
	}

	return nil
}

// VerifyChecksum verifies the SHA256 checksum of a file
func VerifyChecksum(filePath string, expectedSHA256 string) error {
	if expectedSHA256 == "" {
		return nil // No checksum to verify
	}

	checksums := &checksum.Checksums{
		SHA256: expectedSHA256,
	}

	return checksum.VerifyFile(filePath, checksums)
}

// ExtractArchive extracts a downloaded archive to the destination directory
func ExtractArchive(archivePath, destDir string) error {
	// Determine archive type by extension
	ext := strings.ToLower(filepath.Ext(archivePath))

	switch ext {
	case ".zip", ".mrpack":
		// Use existing local client extraction
		localClient := NewLocalClient()
		return localClient.Extract(archivePath, destDir)
	default:
		return fmt.Errorf("unsupported archive format: %s", ext)
	}
}

// SaveRecipeSnapshot saves a .chunk-recipe.json file in the server directory
func SaveRecipeSnapshot(recipe *search.Recipe, destDir string) error {
	snapshot := map[string]interface{}{
		"slug":               recipe.Slug,
		"bench":              recipe.BenchName,
		"name":               recipe.Name,
		"version":            recipe.Version,
		"description":        recipe.Description,
		"mc_version":         recipe.MCVersion,
		"loader":             recipe.Loader,
		"loader_version":     recipe.LoaderVersion,
		"author":             recipe.Author,
		"recommended_ram_gb": recipe.RecommendedRAMGB,
		"download_url":       recipe.DownloadURL,
		"sha256":             recipe.SHA256,
		"installed_at":       time.Now().UTC().Format(time.RFC3339),
	}

	// Write to .chunk-recipe.json
	snapshotPath := filepath.Join(destDir, ".chunk-recipe.json")
	file, err := os.Create(snapshotPath)
	if err != nil {
		return fmt.Errorf("failed to create recipe snapshot: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		return fmt.Errorf("failed to write recipe snapshot: %w", err)
	}

	return nil
}
