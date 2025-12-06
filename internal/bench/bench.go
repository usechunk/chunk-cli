package bench

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/alexinslc/chunk/internal/config"
)

// Manager handles bench operations
type Manager struct {
	config *config.Config
}

// NewManager creates a new bench manager
func NewManager() (*Manager, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &Manager{config: cfg}, nil
}

// GetBenchesDir returns the directory where benches are stored
func GetBenchesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".chunk", "Benches"), nil
}

// NormalizeGitHubURL converts user/repo shorthand to full GitHub URL
func NormalizeGitHubURL(input string) string {
	// If it looks like a URL already, return as-is
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") || strings.HasPrefix(input, "git@") {
		return input
	}

	// If it's in user/repo format, expand to GitHub URL
	if strings.Contains(input, "/") && !strings.Contains(input, "://") {
		return fmt.Sprintf("https://github.com/%s", input)
	}

	return input
}

// Add adds a new bench
func (m *Manager) Add(name string, url string) error {
	// Normalize the URL if needed
	if url == "" {
		url = NormalizeGitHubURL(name)
	}

	// Validate URL format to prevent command injection
	if err := validateGitURL(url); err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check if bench already exists
	for _, b := range m.config.Benches {
		if b.Name == name {
			return fmt.Errorf("bench '%s' already exists", name)
		}
	}

	// Get benches directory
	benchesDir, err := GetBenchesDir()
	if err != nil {
		return err
	}

	// Create benches directory if it doesn't exist
	if err := os.MkdirAll(benchesDir, 0755); err != nil {
		return fmt.Errorf("failed to create benches directory: %w", err)
	}

	// Validate and sanitize the bench name to prevent path traversal
	if strings.Contains(name, "..") || filepath.IsAbs(name) {
		return fmt.Errorf("invalid bench name: cannot contain '..' or be an absolute path")
	}

	// Determine the bench path
	benchPath := filepath.Clean(filepath.Join(benchesDir, name))

	// Ensure the benchPath is still under benchesDir
	rel, err := filepath.Rel(benchesDir, benchPath)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return fmt.Errorf("invalid bench name: path traversal detected")
	}
	// Clone the repository
	cmd := exec.Command("git", "clone", url, benchPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Validate that Recipes directory exists
	recipesDir := filepath.Join(benchPath, "Recipes")
	if _, err := os.Stat(recipesDir); os.IsNotExist(err) {
		// Clean up the cloned directory
		if cleanupErr := os.RemoveAll(benchPath); cleanupErr != nil {
			return fmt.Errorf("invalid bench: no Recipes/ directory found (cleanup also failed: %v)", cleanupErr)
		}
		return fmt.Errorf("invalid bench: no Recipes/ directory found")
	}

	// Add to config
	bench := config.Bench{
		Name:  name,
		URL:   url,
		Path:  benchPath,
		Added: time.Now(),
	}
	m.config.Benches = append(m.config.Benches, bench)

	// Save config
	if err := m.config.Save(); err != nil {
		// Clean up on config save failure
		if cleanupErr := os.RemoveAll(benchPath); cleanupErr != nil {
			return fmt.Errorf("failed to save config: %w (also failed to cleanup: %v)", err, cleanupErr)
		}
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// Remove removes a bench
func (m *Manager) Remove(name string) error {
	// Find the bench
	benchIndex := -1
	var benchToRemove config.Bench
	for i, b := range m.config.Benches {
		if b.Name == name {
			benchIndex = i
			benchToRemove = b
			break
		}
	}

	if benchIndex == -1 {
		return fmt.Errorf("bench '%s' not found", name)
	}

	// Remove the directory
	if err := os.RemoveAll(benchToRemove.Path); err != nil {
		return fmt.Errorf("failed to remove bench directory: %w", err)
	}

	// Remove from config
	m.config.Benches = append(m.config.Benches[:benchIndex], m.config.Benches[benchIndex+1:]...)

	// Save config
	if err := m.config.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// List returns all benches
func (m *Manager) List() []config.Bench {
	return m.config.Benches
}

// Get returns a specific bench by name
func (m *Manager) Get(name string) (*config.Bench, error) {
	for i, b := range m.config.Benches {
		if b.Name == name {
			return &m.config.Benches[i], nil
		}
	}
	return nil, fmt.Errorf("bench '%s' not found", name)
}

// validateGitURL validates that a URL is safe to use with git clone
func validateGitURL(url string) error {
	// Check for empty URL
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Check for shell metacharacters that could be dangerous
	dangerousChars := []string{";", "&", "|", "`", "(", ")", "<", ">", "\n", "\r"}
	for _, char := range dangerousChars {
		if strings.Contains(url, char) {
			return fmt.Errorf("URL contains invalid character: %s", char)
		}
	}
	// Block command substitution and variable expansion patterns
	if strings.Contains(url, "$(") || strings.Contains(url, "${") {
		return fmt.Errorf("URL contains potentially dangerous shell pattern: $() or ${}")
	}

	// Validate that it's a reasonable URL format
	// Allow: http://, https://, git@, ssh://, file://, or local paths
	validPrefixes := []string{"http://", "https://", "git@", "ssh://", "file://", "/", "./", "../"}
	hasValidPrefix := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(url, prefix) {
			hasValidPrefix = true
			break
		}
	}

	if !hasValidPrefix {
		return fmt.Errorf("URL must start with a valid protocol or path")
	}

	return nil
}

// UpdateResult contains the result of updating a bench
type UpdateResult struct {
	BenchName       string
	Success         bool
	AlreadyUpToDate bool
	Error           error
	NewRecipes      []string
	UpdatedRecipes  []RecipeUpdate
	RemovedRecipes  []string
}

// RecipeUpdate represents an updated recipe with version info
type RecipeUpdate struct {
	Name       string
	OldVersion string
	NewVersion string
}

// Update updates a specific bench by running git pull
func (m *Manager) Update(name string) (*UpdateResult, error) {
	// Find the bench
	benchIndex := -1
	var benchToUpdate *config.Bench
	for i, b := range m.config.Benches {
		if b.Name == name {
			benchIndex = i
			benchToUpdate = &m.config.Benches[i]
			break
		}
	}

	if benchIndex == -1 {
		return nil, fmt.Errorf("bench '%s' not found", name)
	}

	result := &UpdateResult{
		BenchName: name,
		Success:   false,
	}

	// Check if the directory exists
	if _, err := os.Stat(benchToUpdate.Path); os.IsNotExist(err) {
		result.Error = fmt.Errorf("bench directory not found: %s", benchToUpdate.Path)
		return result, result.Error
	}

	// Get the current HEAD before pulling
	oldHeadCmd := exec.Command("git", "-C", benchToUpdate.Path, "rev-parse", "HEAD")
	oldHeadOutput, err := oldHeadCmd.Output()
	if err != nil {
		result.Error = fmt.Errorf("failed to get current HEAD: %w", err)
		return result, result.Error
	}
	oldHead := strings.TrimSpace(string(oldHeadOutput))

	// Run git pull
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", "-C", benchToUpdate.Path, "pull")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		result.Error = fmt.Errorf("git pull failed: %w\nOutput: %s\nError: %s", err, stdout.String(), stderr.String())
		return result, result.Error
	}

	// Get the new HEAD after pulling
	newHeadCmd := exec.Command("git", "-C", benchToUpdate.Path, "rev-parse", "HEAD")
	newHeadOutput, err := newHeadCmd.Output()
	if err != nil {
		result.Error = fmt.Errorf("failed to get new HEAD: %w", err)
		return result, result.Error
	}
	newHead := strings.TrimSpace(string(newHeadOutput))

	// Check if already up to date
	if oldHead == newHead {
		result.AlreadyUpToDate = true
		result.Success = true
		return result, nil
	}

	// Parse git diff to find changed recipes
	diffCmd := exec.Command("git", "-C", benchToUpdate.Path, "diff", "--name-status", oldHead, newHead)
	diffOutput, err := diffCmd.Output()
	if err != nil {
		// If diff fails, still consider the update successful and save timestamp
		return m.updateBenchTimestamp(benchToUpdate, result)
	}

	// Parse the diff output
	lines := strings.Split(strings.TrimSpace(string(diffOutput)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		filePath := parts[1]

		// Only consider files in Recipes/ directory with .json, .yaml, or .yml extension
		if !strings.HasPrefix(filePath, "Recipes/") {
			continue
		}

		ext := strings.ToLower(filepath.Ext(filePath))
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			continue
		}

		fileName := filepath.Base(filePath)

		switch status {
		case "A": // Added
			result.NewRecipes = append(result.NewRecipes, fileName)
		case "M": // Modified
			// Try to extract version information
			oldVersion, newVersion := extractVersionChange(benchToUpdate.Path, oldHead, newHead, filePath)
			result.UpdatedRecipes = append(result.UpdatedRecipes, RecipeUpdate{
				Name:       fileName,
				OldVersion: oldVersion,
				NewVersion: newVersion,
			})
		case "D": // Deleted
			result.RemovedRecipes = append(result.RemovedRecipes, fileName)
		}
	}

	// Update last_updated timestamp
	return m.updateBenchTimestamp(benchToUpdate, result)
}

// updateBenchTimestamp updates the last_updated timestamp for a bench and saves the config
func (m *Manager) updateBenchTimestamp(bench *config.Bench, result *UpdateResult) (*UpdateResult, error) {
	now := time.Now()
	bench.LastUpdated = &now
	if err := m.config.Save(); err != nil {
		result.Error = fmt.Errorf("update succeeded but failed to save config: %w", err)
		result.Success = true
		return result, result.Error
	}
	result.Success = true
	return result, nil
}

// UpdateAll updates all benches
func (m *Manager) UpdateAll() ([]*UpdateResult, error) {
	if len(m.config.Benches) == 0 {
		return nil, fmt.Errorf("no benches installed")
	}

	results := make([]*UpdateResult, 0, len(m.config.Benches))

	for _, bench := range m.config.Benches {
		result, err := m.Update(bench.Name)
		// Always include result, even on error
		if result != nil {
			results = append(results, result)
		} else if err != nil {
			// Create error result if Update returned nil result
			results = append(results, &UpdateResult{
				BenchName: bench.Name,
				Success:   false,
				Error:     err,
			})
		}
	}

	return results, nil
}

// extractVersionChange attempts to extract version information from recipe file changes
func extractVersionChange(repoPath, oldCommit, newCommit, filePath string) (string, string) {
	// Get old version
	oldCmd := exec.Command("git", "-C", repoPath, "show", fmt.Sprintf("%s:%s", oldCommit, filePath))
	oldContent, err := oldCmd.Output()
	if err != nil {
		return "", ""
	}

	// Get new version
	newCmd := exec.Command("git", "-C", repoPath, "show", fmt.Sprintf("%s:%s", newCommit, filePath))
	newContent, err := newCmd.Output()
	if err != nil {
		return "", ""
	}

	// Try to extract version from content (looking for version patterns)
	// Matches standard semver: X.Y.Z with optional pre-release and build metadata
	versionRegex := regexp.MustCompile(`(?i)version["\s:]+([0-9]+\.[0-9]+\.[0-9]+(?:-[a-zA-Z0-9]+(?:\.[a-zA-Z0-9]+)*)?(?:\+[a-zA-Z0-9]+(?:\.[a-zA-Z0-9]+)*)?)`)

	oldMatches := versionRegex.FindStringSubmatch(string(oldContent))
	newMatches := versionRegex.FindStringSubmatch(string(newContent))

	oldVersion := ""
	newVersion := ""

	if len(oldMatches) > 1 {
		oldVersion = oldMatches[1]
	}
	if len(newMatches) > 1 {
		newVersion = newMatches[1]
	}

	return oldVersion, newVersion
}

// EnsureCoreBench automatically adds the core usechunk/recipes bench if no benches are installed.
// This mimics Homebrew's behavior of adding homebrew-core on first run.
// Returns nil if benches already exist, if CHUNK_NO_AUTO_BENCH=1 is set, or if successful.
func EnsureCoreBench() error {
	// Check if auto-bench is disabled via environment variable
	if os.Getenv("CHUNK_NO_AUTO_BENCH") == "1" {
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// If benches already exist, nothing to do
	if len(cfg.Benches) > 0 {
		return nil
	}

	// Display message to user
	fmt.Println()
	fmt.Println("No recipe benches installed. Adding core bench...")
	fmt.Println("==> Cloning usechunk/recipes")
	fmt.Println()

	// Create a manager and add the core bench
	manager := &Manager{config: cfg}
	if err := manager.Add("usechunk/recipes", ""); err != nil {
		// If it fails, return error but don't block the command
		return fmt.Errorf("failed to add core bench: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ Core bench added successfully!")
	fmt.Println()

	return nil
}
