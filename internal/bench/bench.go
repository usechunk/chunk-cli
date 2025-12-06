package bench

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	// Determine the bench path
	benchPath := filepath.Join(benchesDir, name)

	// Clone the repository
	cmd := exec.Command("git", "clone", url, benchPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Validate that Recipes directory exists
	recipesDir := filepath.Join(benchPath, "Recipes")
	if _, err := os.Stat(recipesDir); os.IsNotExist(err) {
		// Clean up the cloned directory
		os.RemoveAll(benchPath)
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
		os.RemoveAll(benchPath)
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
	for _, b := range m.config.Benches {
		if b.Name == name {
			return &b, nil
		}
	}
	return nil, fmt.Errorf("bench '%s' not found", name)
}
