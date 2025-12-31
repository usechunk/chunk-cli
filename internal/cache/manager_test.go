package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexinslc/chunk/internal/tracking"
)

func TestNewManager(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Manager is nil")
	}

	if manager.cacheDir == "" {
		t.Error("Cache directory is empty")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test-file.mrpack", "test-file.mrpack"},
		{"test file.mrpack", "test_file.mrpack"},
		{"test/file.mrpack", "test_file.mrpack"},
		{"test:file.mrpack", "test_file.mrpack"},
		{"test123-ABC_xyz.mrpack", "test123-ABC_xyz.mrpack"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetCachePath(t *testing.T) {
	// Create a temporary cache directory for testing
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	cachePath := manager.GetCachePath("atm9", "0.3.1", "modpack.mrpack")
	expectedPath := filepath.Join(tmpDir, ".chunk", DownloadsDir, "atm9-0.3.1-modpack.mrpack")

	if cachePath != expectedPath {
		t.Errorf("GetCachePath() = %q, want %q", cachePath, expectedPath)
	}
}

func TestMetadataOperations(t *testing.T) {
	// Create a temporary cache directory for testing
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create a test file
	cachePath := filepath.Join(manager.cacheDir, "test-file.mrpack")
	if err := os.WriteFile(cachePath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Save metadata
	metadata := &DownloadMetadata{
		Slug:         "test",
		Version:      "1.0.0",
		Filename:     "test-file.mrpack",
		Size:         12,
		DownloadURL:  "https://example.com/test.mrpack",
		SHA256:       "abc123",
		DownloadedAt: time.Now(),
		LastUsedAt:   time.Now(),
	}

	if err := manager.SaveMetadata(cachePath, metadata); err != nil {
		t.Fatalf("Failed to save metadata: %v", err)
	}

	// Load metadata
	loadedMetadata, err := manager.LoadMetadata(cachePath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	if loadedMetadata == nil {
		t.Fatal("Loaded metadata is nil")
	}

	if loadedMetadata.Slug != metadata.Slug {
		t.Errorf("Loaded slug = %q, want %q", loadedMetadata.Slug, metadata.Slug)
	}

	if loadedMetadata.Version != metadata.Version {
		t.Errorf("Loaded version = %q, want %q", loadedMetadata.Version, metadata.Version)
	}
}

func TestAnalyzeCache(t *testing.T) {
	// Create a temporary cache directory for testing
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create test files
	// 1. Outdated version
	outdatedPath := filepath.Join(manager.cacheDir, "atm9-0.3.1-modpack.mrpack")
	if err := os.WriteFile(outdatedPath, []byte("outdated content"), 0644); err != nil {
		t.Fatalf("Failed to write outdated file: %v", err)
	}
	if err := manager.SaveMetadata(outdatedPath, &DownloadMetadata{
		Slug:         "atm9",
		Version:      "0.3.1",
		Filename:     "modpack.mrpack",
		Size:         16,
		DownloadedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Failed to save metadata for outdated file: %v", err)
	}

	// 2. Current version
	currentPath := filepath.Join(manager.cacheDir, "atm9-0.3.2-modpack.mrpack")
	if err := os.WriteFile(currentPath, []byte("current content"), 0644); err != nil {
		t.Fatalf("Failed to write current file: %v", err)
	}
	if err := manager.SaveMetadata(currentPath, &DownloadMetadata{
		Slug:         "atm9",
		Version:      "0.3.2",
		Filename:     "modpack.mrpack",
		Size:         15,
		DownloadedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Failed to save metadata for current file: %v", err)
	}

	// 3. Uninstalled modpack
	uninstalledPath := filepath.Join(manager.cacheDir, "vh-1.18.1-modpack.mrpack")
	if err := os.WriteFile(uninstalledPath, []byte("uninstalled"), 0644); err != nil {
		t.Fatalf("Failed to write uninstalled file: %v", err)
	}
	if err := manager.SaveMetadata(uninstalledPath, &DownloadMetadata{
		Slug:         "vault-hunters",
		Version:      "1.18.1",
		Filename:     "modpack.mrpack",
		Size:         11,
		DownloadedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Failed to save metadata for uninstalled file: %v", err)
	}

	// 4. Partial download (no metadata)
	partialPath := filepath.Join(manager.cacheDir, "partial.tmp")
	if err := os.WriteFile(partialPath, []byte("partial"), 0644); err != nil {
		t.Fatalf("Failed to write partial file: %v", err)
	}

	// Create tracked installation
	tracker, _ := tracking.NewTracker()
	installation := &tracking.Installation{
		Slug:        "atm9",
		Version:     "0.3.2",
		Bench:       "test",
		Path:        filepath.Join(tmpDir, "server"),
		InstalledAt: time.Now(),
	}
	tracker.AddInstallation(installation)

	// Analyze cache
	stats, err := manager.AnalyzeCache()
	if err != nil {
		t.Fatalf("Failed to analyze cache: %v", err)
	}

	if stats.TotalFiles != 4 {
		t.Errorf("TotalFiles = %d, want 4", stats.TotalFiles)
	}

	if stats.OutdatedFiles != 1 {
		t.Errorf("OutdatedFiles = %d, want 1", stats.OutdatedFiles)
	}

	if stats.UninstalledFiles != 1 {
		t.Errorf("UninstalledFiles = %d, want 1", stats.UninstalledFiles)
	}

	if stats.PartialFiles != 1 {
		t.Errorf("PartialFiles = %d, want 1", stats.PartialFiles)
	}

	if len(stats.FilesToRemove) != 3 {
		t.Errorf("FilesToRemove count = %d, want 3", len(stats.FilesToRemove))
	}
}

func TestGetCacheSize(t *testing.T) {
	// Create a temporary cache directory for testing
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create test files
	file1 := filepath.Join(manager.cacheDir, "file1.mrpack")
	if err := os.WriteFile(file1, []byte("test content 1"), 0644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}

	file2 := filepath.Join(manager.cacheDir, "file2.mrpack")
	if err := os.WriteFile(file2, []byte("test content 2"), 0644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	totalSize, fileCount, err := manager.GetCacheSize()
	if err != nil {
		t.Fatalf("Failed to get cache size: %v", err)
	}

	if fileCount != 2 {
		t.Errorf("FileCount = %d, want 2", fileCount)
	}

	expectedSize := int64(len("test content 1") + len("test content 2"))
	if totalSize != expectedSize {
		t.Errorf("TotalSize = %d, want %d", totalSize, expectedSize)
	}
}

func TestRemoveFile(t *testing.T) {
	// Create a temporary cache directory for testing
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create test file and metadata
	cachePath := filepath.Join(manager.cacheDir, "test-file.mrpack")
	if err := os.WriteFile(cachePath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	metadata := &DownloadMetadata{
		Slug:         "test",
		Version:      "1.0.0",
		Filename:     "test-file.mrpack",
		DownloadedAt: time.Now(),
	}
	if err := manager.SaveMetadata(cachePath, metadata); err != nil {
		t.Fatalf("Failed to save metadata: %v", err)
	}

	// Verify files exist
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("Cache file does not exist")
	}

	metadataPath := manager.GetMetadataPath(cachePath)
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Fatal("Metadata file does not exist")
	}

	// Remove file
	if err := manager.RemoveFile(cachePath); err != nil {
		t.Fatalf("Failed to remove file: %v", err)
	}

	// Verify files are removed
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("Cache file still exists after removal")
	}

	if _, err := os.Stat(metadataPath); !os.IsNotExist(err) {
		t.Error("Metadata file still exists after removal")
	}
}
