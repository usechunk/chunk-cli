package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexinslc/chunk/internal/tracking"
)

const (
	// DownloadsDir is the subdirectory name for cached downloads
	DownloadsDir = "downloads"
	// MetadataFile is the name of the metadata file for each download
	MetadataFile = ".metadata.json"
)

// DownloadMetadata stores information about a cached download
type DownloadMetadata struct {
	Slug         string    `json:"slug"`
	Version      string    `json:"version"`
	Filename     string    `json:"filename"`
	Size         int64     `json:"size"`
	DownloadURL  string    `json:"download_url"`
	SHA256       string    `json:"sha256"`
	DownloadedAt time.Time `json:"downloaded_at"`
	LastUsedAt   time.Time `json:"last_used_at"`
}

// CachedFile represents a file in the download cache with its metadata
type CachedFile struct {
	Path     string
	Metadata *DownloadMetadata
	Size     int64
	Reason   string // Reason for removal: "partial", "outdated", "uninstalled"
}

// CleanupStats contains statistics about a cleanup operation
type CleanupStats struct {
	TotalFiles       int
	OutdatedFiles    int
	UninstalledFiles int
	PartialFiles     int
	TotalSize        int64
	FilesToRemove    []*CachedFile
}

// Manager handles download cache operations
type Manager struct {
	cacheDir string
	tracker  *tracking.Tracker
}

// NewManager creates a new cache manager
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDir := filepath.Join(home, ".chunk", DownloadsDir)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	tracker, err := tracking.NewTracker()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracker: %w", err)
	}

	return &Manager{
		cacheDir: cacheDir,
		tracker:  tracker,
	}, nil
}

// GetCachePath returns the path to a cached file for the given slug, version, and filename.
func (m *Manager) GetCachePath(slug, version, filename string) string {
	// Create a safe filename using slug and version
	safeName := sanitizeFilename(fmt.Sprintf("%s-%s-%s", slug, version, filename))
	return filepath.Join(m.cacheDir, safeName)
}

// GetMetadataPath returns the path to the metadata file associated with a cached download.
func (m *Manager) GetMetadataPath(cachePath string) string {
	dir := filepath.Dir(cachePath)
	base := filepath.Base(cachePath)
	return filepath.Join(dir, "."+base+".metadata.json")
}

// SaveMetadata saves metadata for a cached download to disk.
func (m *Manager) SaveMetadata(cachePath string, metadata *DownloadMetadata) error {
	metadataPath := m.GetMetadataPath(cachePath)
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// LoadMetadata loads metadata for a cached download from disk, returning nil if no metadata exists.
func (m *Manager) LoadMetadata(cachePath string) (*DownloadMetadata, error) {
	metadataPath := m.GetMetadataPath(cachePath)
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No metadata available
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata DownloadMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// ListCachedFiles returns all cached files with their associated metadata.
func (m *Manager) ListCachedFiles() ([]*CachedFile, error) {
	entries, err := os.ReadDir(m.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*CachedFile{}, nil
		}
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	var cachedFiles []*CachedFile
	for _, entry := range entries {
		// Skip metadata files and directories
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		filePath := filepath.Join(m.cacheDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		metadata, _ := m.LoadMetadata(filePath)

		cachedFiles = append(cachedFiles, &CachedFile{
			Path:     filePath,
			Metadata: metadata,
			Size:     info.Size(),
		})
	}

	return cachedFiles, nil
}

// AnalyzeCache analyzes the cache and identifies files that can be removed based on
// outdated versions, uninstalled modpacks, and partial downloads.
func (m *Manager) AnalyzeCache() (*CleanupStats, error) {
	cachedFiles, err := m.ListCachedFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list cached files: %w", err)
	}

	installations, err := m.tracker.ListInstallations()
	if err != nil {
		return nil, fmt.Errorf("failed to list installations: %w", err)
	}

	// Build a map of installed modpacks and their versions
	installedMap := make(map[string]map[string]bool)
	for _, inst := range installations {
		if installedMap[inst.Slug] == nil {
			installedMap[inst.Slug] = make(map[string]bool)
		}
		installedMap[inst.Slug][inst.Version] = true
	}

	stats := &CleanupStats{
		TotalFiles:    len(cachedFiles),
		FilesToRemove: []*CachedFile{},
	}

	for _, file := range cachedFiles {
		shouldRemove := false
		var reason string

		// Check if it's a partial download (no metadata)
		if file.Metadata == nil {
			stats.PartialFiles++
			shouldRemove = true
			reason = "partial"
		} else {
			// Check if modpack is installed
			versions, isInstalled := installedMap[file.Metadata.Slug]
			if !isInstalled {
				stats.UninstalledFiles++
				shouldRemove = true
				reason = "uninstalled"
			} else if !versions[file.Metadata.Version] {
				// Modpack is installed but this is an outdated version
				stats.OutdatedFiles++
				shouldRemove = true
				reason = "outdated"
			}
		}

		if shouldRemove {
			file.Reason = reason
			stats.TotalSize += file.Size
			stats.FilesToRemove = append(stats.FilesToRemove, file)
		}
	}

	return stats, nil
}

// GetCacheSize returns the total size of all cached files in bytes and the count of files.
func (m *Manager) GetCacheSize() (int64, int, error) {
	cachedFiles, err := m.ListCachedFiles()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list cached files: %w", err)
	}

	var totalSize int64
	for _, file := range cachedFiles {
		totalSize += file.Size
	}

	return totalSize, len(cachedFiles), nil
}

// RemoveFile removes a cached file and its associated metadata from disk.
func (m *Manager) RemoveFile(cachePath string) error {
	// Remove the file
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	// Remove the metadata file
	metadataPath := m.GetMetadataPath(cachePath)
	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove metadata file: %w", err)
	}

	return nil
}

// CleanupFiles removes the specified list of files from the cache.
func (m *Manager) CleanupFiles(files []*CachedFile) error {
	for _, file := range files {
		if err := m.RemoveFile(file.Path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", file.Path, err)
		}
	}
	return nil
}

// CleanupAll removes all cached files from the download cache.
func (m *Manager) CleanupAll() error {
	cachedFiles, err := m.ListCachedFiles()
	if err != nil {
		return fmt.Errorf("failed to list cached files: %w", err)
	}

	return m.CleanupFiles(cachedFiles)
}

// UpdateLastUsed updates the last used timestamp for a cached file's metadata.
func (m *Manager) UpdateLastUsed(cachePath string) error {
	metadata, err := m.LoadMetadata(cachePath)
	if err != nil || metadata == nil {
		// If no metadata exists, we can't update it
		return nil
	}

	metadata.LastUsedAt = time.Now()
	return m.SaveMetadata(cachePath, metadata)
}

// sanitizeFilename creates a safe filename from a string
func sanitizeFilename(name string) string {
	// Replace unsafe characters with underscores
	var result []byte
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' {
			result = append(result, ch)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}
