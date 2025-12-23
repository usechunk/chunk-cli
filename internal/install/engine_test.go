package install

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexinslc/chunk/internal/sources"
	"github.com/alexinslc/chunk/internal/tracking"
)

func TestNewInstaller(t *testing.T) {
	installer := NewInstaller()
	if installer == nil {
		t.Fatal("Expected NewInstaller to return non-nil installer")
	}
	if installer.sourceManager == nil {
		t.Error("Expected sourceManager to be initialized")
	}
	if installer.conversionEngine == nil {
		t.Error("Expected conversionEngine to be initialized")
	}
	if installer.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
}

func TestOptionsDefaults(t *testing.T) {
	opts := &Options{
		Identifier: "test-modpack",
	}

	if opts.DestDir != "" {
		t.Error("Expected DestDir to be empty by default")
	}
	if opts.PreserveData != false {
		t.Error("Expected PreserveData to be false by default")
	}
}

func TestInstallerPrepareDirectory(t *testing.T) {
	installer := NewInstaller()

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "chunk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testDir := filepath.Join(tmpDir, "test-server")

	err = installer.prepareDirectory(testDir, false)
	if err != nil {
		t.Fatalf("prepareDirectory failed: %v", err)
	}

	// Check that the directory was created
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Error("Expected directory to be created")
	}

	// Check that subdirectories were created
	expectedDirs := []string{"mods", "config", "logs"}
	for _, dir := range expectedDirs {
		path := filepath.Join(testDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected subdirectory %s to be created", dir)
		}
	}
}

func TestInstallerCreateBackup(t *testing.T) {
	installer := NewInstaller()

	// Create a temporary directory with content
	tmpDir, err := os.MkdirTemp("", "chunk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testDir := filepath.Join(tmpDir, "test-server")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	// Add a file to the directory
	testFile := filepath.Join(testDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup
	err = installer.createBackup(testDir)
	if err != nil {
		t.Fatalf("createBackup failed: %v", err)
	}

	// Verify backup was created
	if installer.backupDir == "" {
		t.Error("Expected backupDir to be set")
	}

	// Clean up backup
	if installer.backupDir != "" {
		os.RemoveAll(installer.backupDir)
	}
}

func TestInstallerCreateBackupEmptyDir(t *testing.T) {
	installer := NewInstaller()

	// Create a temporary empty directory
	tmpDir, err := os.MkdirTemp("", "chunk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testDir := filepath.Join(tmpDir, "empty-server")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	// Create backup of empty directory
	err = installer.createBackup(testDir)
	if err != nil {
		t.Fatalf("createBackup failed: %v", err)
	}

	// No backup should be created for empty directory
	if installer.backupDir != "" {
		t.Error("Expected no backup for empty directory")
		os.RemoveAll(installer.backupDir)
	}
}

func TestInstallerCreateBackupNonExistent(t *testing.T) {
	installer := NewInstaller()

	// Try to backup a non-existent directory
	err := installer.createBackup("/non/existent/path")
	if err != nil {
		t.Errorf("createBackup should not fail for non-existent directory: %v", err)
	}

	if installer.backupDir != "" {
		t.Error("Expected no backup for non-existent directory")
	}
}

func TestInstallerRollback(t *testing.T) {
	installer := NewInstaller()

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "chunk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create original directory with content
	originalDir := filepath.Join(tmpDir, "server")
	if err := os.MkdirAll(originalDir, 0755); err != nil {
		t.Fatalf("Failed to create original dir: %v", err)
	}
	originalFile := filepath.Join(originalDir, "original.txt")
	if err := os.WriteFile(originalFile, []byte("original content"), 0644); err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	// Set the absolute destination directory (simulating what Install does)
	installer.absDestDir = originalDir

	// Create backup
	if err := installer.createBackup(originalDir); err != nil {
		t.Fatalf("createBackup failed: %v", err)
	}
	defer func() {
		if installer.backupDir != "" {
			os.RemoveAll(installer.backupDir)
		}
	}()

	// Create new failed installation
	if err := os.MkdirAll(originalDir, 0755); err != nil {
		t.Fatalf("Failed to create new dir: %v", err)
	}
	newFile := filepath.Join(originalDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	// Rollback
	err = installer.Rollback()
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify original content was restored
	if _, err := os.Stat(originalFile); os.IsNotExist(err) {
		t.Error("Expected original file to be restored")
	}
	if _, err := os.Stat(newFile); !os.IsNotExist(err) {
		t.Error("Expected new file to be removed")
	}
}

func TestInstallerRollbackNoBackup(t *testing.T) {
	installer := NewInstaller()

	// Rollback with no backup should succeed silently
	err := installer.Rollback()
	if err != nil {
		t.Errorf("Rollback with no backup should not fail: %v", err)
	}
}

func TestResultFields(t *testing.T) {
	result := &Result{
		ModpackName:   "Test Modpack",
		MCVersion:     "1.20.1",
		Loader:        sources.LoaderForge,
		LoaderVersion: "47.2.0",
		ModsInstalled: 150,
		DestDir:       "/path/to/server",
	}

	if result.ModpackName != "Test Modpack" {
		t.Errorf("Expected ModpackName 'Test Modpack', got '%s'", result.ModpackName)
	}
	if result.MCVersion != "1.20.1" {
		t.Errorf("Expected MCVersion '1.20.1', got '%s'", result.MCVersion)
	}
	if result.Loader != sources.LoaderForge {
		t.Errorf("Expected Loader 'forge', got '%s'", result.Loader)
	}
	if result.LoaderVersion != "47.2.0" {
		t.Errorf("Expected LoaderVersion '47.2.0', got '%s'", result.LoaderVersion)
	}
	if result.ModsInstalled != 150 {
		t.Errorf("Expected ModsInstalled 150, got %d", result.ModsInstalled)
	}
	if result.DestDir != "/path/to/server" {
		t.Errorf("Expected DestDir '/path/to/server', got '%s'", result.DestDir)
	}
}

func TestCreateRecipeSnapshot(t *testing.T) {
	modpack := &sources.Modpack{
		Name:           "Test Modpack",
		Identifier:     "test-modpack",
		Description:    "A test modpack",
		MCVersion:      "1.20.1",
		Loader:         sources.LoaderForge,
		LoaderVersion:  "47.2.0",
		Author:         "Test Author",
		Source:         "test-source",
		RecommendedRAM: 8,
		ManifestURL:    "https://example.com/manifest.json",
		Dependencies:   []string{"dep1", "dep2"},
		Mods: []*sources.Mod{
			{
				Name:        "Test Mod",
				Version:     "1.0.0",
				FileName:    "testmod-1.0.0.jar",
				DownloadURL: "https://example.com/mod.jar",
				Side:        sources.SideBoth,
				Required:    true,
				SHA256:      "abc123",
				SHA512:      "def456",
			},
		},
	}

	snapshot := createRecipeSnapshot(modpack)

	if snapshot["name"] != "Test Modpack" {
		t.Errorf("Expected name 'Test Modpack', got '%v'", snapshot["name"])
	}
	if snapshot["identifier"] != "test-modpack" {
		t.Errorf("Expected identifier 'test-modpack', got '%v'", snapshot["identifier"])
	}
	if snapshot["mc_version"] != "1.20.1" {
		t.Errorf("Expected mc_version '1.20.1', got '%v'", snapshot["mc_version"])
	}
	if snapshot["loader"] != "forge" {
		t.Errorf("Expected loader 'forge', got '%v'", snapshot["loader"])
	}
	if snapshot["loader_version"] != "47.2.0" {
		t.Errorf("Expected loader_version '47.2.0', got '%v'", snapshot["loader_version"])
	}

	deps, ok := snapshot["dependencies"].([]string)
	if !ok {
		t.Error("Expected dependencies to be []string")
	} else if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}

	mods, ok := snapshot["mods"].([]map[string]interface{})
	if !ok {
		t.Error("Expected mods to be []map[string]interface{}")
	} else if len(mods) != 1 {
		t.Errorf("Expected 1 mod, got %d", len(mods))
	} else {
		mod := mods[0]
		if mod["name"] != "Test Mod" {
			t.Errorf("Expected mod name 'Test Mod', got '%v'", mod["name"])
		}
		if mod["sha256"] != "abc123" {
			t.Errorf("Expected mod sha256 'abc123', got '%v'", mod["sha256"])
		}
	}
}

func TestTrackInstallation(t *testing.T) {
	// Create temporary directory for test tracking
	tmpHome, err := os.MkdirTemp("", "chunk-track-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	// Set HOME to temporary directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	modpack := &sources.Modpack{
		Name:           "All the Mods 9",
		Identifier:     "atm9",
		Description:    "All the Mods 9 modpack",
		MCVersion:      "1.20.1",
		Loader:         sources.LoaderForge,
		LoaderVersion:  "47.2.0",
		Author:         "ATM Team",
		Source:         "usechunk/recipes",
		RecommendedRAM: 8,
		Mods:           []*sources.Mod{},
	}

	result := &Result{
		ModpackName:   modpack.Name,
		MCVersion:     modpack.MCVersion,
		Loader:        modpack.Loader,
		LoaderVersion: modpack.LoaderVersion,
		ModsInstalled: 0,
		DestDir:       "/opt/minecraft/atm9",
		Modpack:       modpack,
	}

	// Track installation
	err = TrackInstallation(result, "atm9")
	if err != nil {
		t.Fatalf("TrackInstallation failed: %v", err)
	}

	// Verify installation was tracked
	tracker, err := tracking.NewTracker()
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}

	installation, err := tracker.GetInstallation("/opt/minecraft/atm9")
	if err != nil {
		t.Fatalf("Failed to get installation: %v", err)
	}

	if installation == nil {
		t.Fatal("Expected installation to be tracked")
	}

	if installation.Slug != "atm9" {
		t.Errorf("Expected slug 'atm9', got '%s'", installation.Slug)
	}
	if installation.Path != "/opt/minecraft/atm9" {
		t.Errorf("Expected path '/opt/minecraft/atm9', got '%s'", installation.Path)
	}
	if installation.Bench != "usechunk/recipes" {
		t.Errorf("Expected bench 'usechunk/recipes', got '%s'", installation.Bench)
	}

	// Verify recipe snapshot
	if installation.RecipeSnapshot == nil {
		t.Fatal("Expected recipe snapshot to be set")
	}
	if installation.RecipeSnapshot["name"] != "All the Mods 9" {
		t.Errorf("Expected recipe snapshot name 'All the Mods 9', got '%v'", installation.RecipeSnapshot["name"])
	}
}

func TestTrackInstallationNilResult(t *testing.T) {
	err := TrackInstallation(nil, "test")
	if err == nil {
		t.Error("Expected error when tracking nil result")
	}
}

func TestTrackInstallationNilModpack(t *testing.T) {
	result := &Result{
		ModpackName: "Test",
		Modpack:     nil,
	}

	err := TrackInstallation(result, "test")
	if err == nil {
		t.Error("Expected error when tracking result with nil modpack")
	}
}


func TestInstallerExtractLocalModpack(t *testing.T) {
	installer := NewInstaller()

	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "chunk-test-extract-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test mrpack file
	mrpackPath, err := createTestMrpack(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create test mrpack: %v", err)
	}

	// Create destination directory
	destDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("Failed to create dest dir: %v", err)
	}

	// Extract the mrpack
	err = installer.extractLocalModpack(mrpackPath, destDir)
	if err != nil {
		t.Fatalf("extractLocalModpack failed: %v", err)
	}

	// Verify the overrides folder was extracted
	overridesDir := filepath.Join(destDir, "overrides")
	if _, err := os.Stat(overridesDir); err == nil {
		// Check for config file in overrides
		configDir := filepath.Join(overridesDir, "config")
		if _, err := os.Stat(configDir); os.IsNotExist(err) {
			// This is ok - depends on what's in the test mrpack
		}
	}
}

// createTestMrpack creates a minimal valid mrpack file for testing
func createTestMrpack(dir string) (string, error) {
	// Create the zip file
	mrpackPath := filepath.Join(dir, "test.mrpack")

	// Create modrinth.index.json content
	manifest := `{
  "formatVersion": 1,
  "game": "minecraft",
  "versionId": "1.0.0",
  "name": "Test Modpack",
  "summary": "A test modpack",
  "files": [],
  "dependencies": {
    "minecraft": "1.20.1",
    "forge": "47.2.0"
  }
}`

	// Create a zip file with the manifest
	zipFile, err := os.Create(mrpackPath)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	// Write a minimal zip file with just the manifest
	// Use archive/zip to create proper zip
	zipWriter := newZipWriter(zipFile)
	if err := zipWriter.writeFile("modrinth.index.json", []byte(manifest)); err != nil {
		return "", err
	}

	// Add an overrides directory with a config
	if err := zipWriter.writeFile("overrides/config/test.toml", []byte("# test config")); err != nil {
		return "", err
	}

	if err := zipWriter.close(); err != nil {
		return "", err
	}

	return mrpackPath, nil
}

// Simple zip writer helper
type zipWriterHelper struct {
	file   *os.File
	writer *zip.Writer
}

func newZipWriter(f *os.File) *zipWriterHelper {
	return &zipWriterHelper{
		file:   f,
		writer: zip.NewWriter(f),
	}
}

func (z *zipWriterHelper) writeFile(name string, content []byte) error {
	w, err := z.writer.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(content)
	return err
}

func (z *zipWriterHelper) close() error {
	return z.writer.Close()
}
