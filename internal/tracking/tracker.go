package tracking

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Installation represents a single modpack installation record
type Installation struct {
	Slug           string                 `json:"slug"`
	Version        string                 `json:"version"`
	Bench          string                 `json:"bench"`
	Path           string                 `json:"path"`
	InstalledAt    time.Time              `json:"installed_at"`
	RecipeSnapshot map[string]interface{} `json:"recipe_snapshot"`
}

// InstallationRegistry contains all tracked installations
type InstallationRegistry struct {
	Installations []*Installation `json:"installations"`
}

// Tracker manages installation tracking in ~/.chunk/installed.json
type Tracker struct {
	registryPath string
}

// NewTracker creates a new installation tracker
func NewTracker() (*Tracker, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	chunkDir := filepath.Join(home, ".chunk")
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .chunk directory: %w", err)
	}

	registryPath := filepath.Join(chunkDir, "installed.json")

	return &Tracker{
		registryPath: registryPath,
	}, nil
}

// Load reads the installation registry from disk
func (t *Tracker) Load() (*InstallationRegistry, error) {
	// If file doesn't exist, return empty registry
	if _, err := os.Stat(t.registryPath); os.IsNotExist(err) {
		return &InstallationRegistry{
			Installations: []*Installation{},
		}, nil
	}

	data, err := os.ReadFile(t.registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read installed.json: %w", err)
	}

	var registry InstallationRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse installed.json: %w", err)
	}

	// Ensure installations array is not nil
	if registry.Installations == nil {
		registry.Installations = []*Installation{}
	}

	return &registry, nil
}

// Save writes the installation registry to disk
func (t *Tracker) Save(registry *InstallationRegistry) error {
	if registry == nil {
		return fmt.Errorf("registry cannot be nil")
	}

	// Ensure installations array is not nil for JSON serialization
	if registry.Installations == nil {
		registry.Installations = []*Installation{}
	}

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(t.registryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write installed.json: %w", err)
	}

	return nil
}

// AddInstallation adds a new installation record to the registry
func (t *Tracker) AddInstallation(installation *Installation) error {
	if installation == nil {
		return fmt.Errorf("installation cannot be nil")
	}

	if err := validateInstallation(installation); err != nil {
		return fmt.Errorf("invalid installation: %w", err)
	}

	registry, err := t.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Check if installation at this path already exists
	for i, existing := range registry.Installations {
		if existing.Path == installation.Path {
			// Update existing installation
			registry.Installations[i] = installation
			return t.Save(registry)
		}
	}

	// Add new installation
	registry.Installations = append(registry.Installations, installation)
	return t.Save(registry)
}

// RemoveInstallation removes an installation record by path
func (t *Tracker) RemoveInstallation(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	registry, err := t.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Find and remove the installation
	for i, installation := range registry.Installations {
		if installation.Path == path {
			registry.Installations = append(registry.Installations[:i], registry.Installations[i+1:]...)
			return t.Save(registry)
		}
	}

	// Not found is not an error
	return nil
}

// GetInstallation retrieves an installation record by path
func (t *Tracker) GetInstallation(path string) (*Installation, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	registry, err := t.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	for _, installation := range registry.Installations {
		if installation.Path == path {
			return installation, nil
		}
	}

	return nil, nil // Not found
}

// ListInstallations returns all installation records
func (t *Tracker) ListInstallations() ([]*Installation, error) {
	registry, err := t.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	return registry.Installations, nil
}

// UpdateInstallation updates an existing installation record
func (t *Tracker) UpdateInstallation(installation *Installation) error {
	if installation == nil {
		return fmt.Errorf("installation cannot be nil")
	}

	if err := validateInstallation(installation); err != nil {
		return fmt.Errorf("invalid installation: %w", err)
	}

	registry, err := t.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Find and update the installation
	found := false
	for i, existing := range registry.Installations {
		if existing.Path == installation.Path {
			registry.Installations[i] = installation
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("installation not found at path: %s", installation.Path)
	}

	return t.Save(registry)
}

// validateInstallation validates an installation record
func validateInstallation(installation *Installation) error {
	if installation.Slug == "" {
		return fmt.Errorf("slug is required")
	}

	if installation.Version == "" {
		return fmt.Errorf("version is required")
	}

	if installation.Path == "" {
		return fmt.Errorf("path is required")
	}

	if installation.InstalledAt.IsZero() {
		return fmt.Errorf("installed_at is required")
	}

	return nil
}
