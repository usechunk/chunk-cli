package uninstall

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexinslc/chunk/internal/tracking"
)

func TestNewUninstaller(t *testing.T) {
	uninstaller, err := NewUninstaller()
	if err != nil {
		t.Fatalf("NewUninstaller failed: %v", err)
	}

	if uninstaller == nil {
		t.Fatal("Expected uninstaller to be non-nil")
	}

	if uninstaller.tracker == nil {
		t.Error("Expected tracker to be initialized")
	}
}

func TestUninstallNilOptions(t *testing.T) {
	uninstaller, err := NewUninstaller()
	if err != nil {
		t.Fatalf("NewUninstaller failed: %v", err)
	}

	_, err = uninstaller.Uninstall(nil)
	if err == nil {
		t.Error("Expected error when uninstalling with nil options")
	}
	if err.Error() != "options cannot be nil" {
		t.Errorf("Expected 'options cannot be nil' error, got: %v", err)
	}
}

func TestUninstallDirectoryNotExists(t *testing.T) {
	uninstaller, err := NewUninstaller()
	if err != nil {
		t.Fatalf("NewUninstaller failed: %v", err)
	}

	opts := &Options{
		ServerDir: "/non/existent/directory",
		Force:     true,
	}

	_, err = uninstaller.Uninstall(opts)
	if err == nil {
		t.Error("Expected error when uninstalling non-existent directory")
	}
}

func TestUninstallWithForceAndKeepWorlds(t *testing.T) {
	// Create test server directory
	tmpDir := t.TempDir()
	serverDir := filepath.Join(tmpDir, "server")

	// Create test structure
	if err := os.MkdirAll(filepath.Join(serverDir, "mods"), 0755); err != nil {
		t.Fatalf("Failed to create mods directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(serverDir, "config"), 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(serverDir, "world"), 0755); err != nil {
		t.Fatalf("Failed to create world directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "mods", "test.jar"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "world", "level.dat"), []byte("world"), 0644); err != nil {
		t.Fatalf("Failed to create world file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "ops.json"), []byte("[]"), 0644); err != nil {
		t.Fatalf("Failed to create ops.json: %v", err)
	}

	// Create uninstaller
	uninstaller, err := NewUninstaller()
	if err != nil {
		t.Fatalf("NewUninstaller failed: %v", err)
	}

	// Uninstall with force and keep-worlds
	opts := &Options{
		ServerDir:  serverDir,
		KeepWorlds: true,
		Force:      true,
	}

	result, err := uninstaller.Uninstall(opts)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Verify mods were removed
	if _, err := os.Stat(filepath.Join(serverDir, "mods")); !os.IsNotExist(err) {
		t.Error("Expected mods directory to be removed")
	}

	// Verify config was removed
	if _, err := os.Stat(filepath.Join(serverDir, "config")); !os.IsNotExist(err) {
		t.Error("Expected config directory to be removed")
	}

	// Verify world was preserved
	if _, err := os.Stat(filepath.Join(serverDir, "world")); os.IsNotExist(err) {
		t.Error("Expected world directory to be preserved")
	}

	// Verify ops.json was preserved
	if _, err := os.Stat(filepath.Join(serverDir, "ops.json")); os.IsNotExist(err) {
		t.Error("Expected ops.json to be preserved")
	}

	// Check result
	if len(result.RemovedPaths) == 0 {
		t.Error("Expected some paths to be removed")
	}

	if len(result.PreservedPaths) == 0 {
		t.Error("Expected some paths to be preserved")
	}
}

func TestUninstallWithForceNoKeepWorlds(t *testing.T) {
	// Create test server directory
	tmpDir := t.TempDir()
	serverDir := filepath.Join(tmpDir, "server")

	// Create test structure
	if err := os.MkdirAll(filepath.Join(serverDir, "mods"), 0755); err != nil {
		t.Fatalf("Failed to create mods directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(serverDir, "config"), 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "mods", "test.jar"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test mod: %v", err)
	}

	// Create uninstaller
	uninstaller, err := NewUninstaller()
	if err != nil {
		t.Fatalf("NewUninstaller failed: %v", err)
	}

	// Uninstall with force, no keep-worlds
	opts := &Options{
		ServerDir:  serverDir,
		KeepWorlds: false,
		Force:      true,
	}

	result, err := uninstaller.Uninstall(opts)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Verify mods were removed
	if _, err := os.Stat(filepath.Join(serverDir, "mods")); !os.IsNotExist(err) {
		t.Error("Expected mods directory to be removed")
	}

	// Verify config was removed
	if _, err := os.Stat(filepath.Join(serverDir, "config")); !os.IsNotExist(err) {
		t.Error("Expected config directory to be removed")
	}

	// Check result
	if len(result.RemovedPaths) == 0 {
		t.Error("Expected some paths to be removed")
	}

	if len(result.PreservedPaths) != 0 {
		t.Error("Expected no paths to be preserved")
	}
}

func TestUninstallValidateModpackSlug(t *testing.T) {
	// Create test server directory
	tmpDir := t.TempDir()
	serverDir := filepath.Join(tmpDir, "server")

	if err := os.MkdirAll(filepath.Join(serverDir, "mods"), 0755); err != nil {
		t.Fatalf("Failed to create mods directory: %v", err)
	}

	// Create tracker and add installation
	tracker, err := tracking.NewTracker()
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}

	installation := &tracking.Installation{
		Slug:        "atm9",
		Version:     "1.0.0",
		Bench:       "test",
		Path:        serverDir,
		InstalledAt: time.Now(),
		RecipeSnapshot: map[string]interface{}{
			"name": "All the Mods 9",
		},
	}

	if err := tracker.AddInstallation(installation); err != nil {
		t.Fatalf("Failed to add installation: %v", err)
	}

	// Create uninstaller
	uninstaller, err := NewUninstaller()
	if err != nil {
		t.Fatalf("NewUninstaller failed: %v", err)
	}

	// Try to uninstall with wrong modpack slug
	opts := &Options{
		ServerDir:   serverDir,
		ModpackSlug: "different-modpack",
		Force:       true,
	}

	_, err = uninstaller.Uninstall(opts)
	if err == nil {
		t.Error("Expected error when modpack slug doesn't match")
	}
	if err != nil && err.Error() != `requested modpack "different-modpack" does not match installed modpack "atm9" in directory `+serverDir {
		t.Errorf("Unexpected error message: %v", err)
	}

	// Clean up tracker
	tracker.RemoveInstallation(serverDir)
}

func TestUninstallValidateModpackSlugMatches(t *testing.T) {
	// Create test server directory
	tmpDir := t.TempDir()
	serverDir := filepath.Join(tmpDir, "server")

	if err := os.MkdirAll(filepath.Join(serverDir, "mods"), 0755); err != nil {
		t.Fatalf("Failed to create mods directory: %v", err)
	}

	// Create tracker and add installation
	tracker, err := tracking.NewTracker()
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}

	installation := &tracking.Installation{
		Slug:        "atm9",
		Version:     "1.0.0",
		Bench:       "test",
		Path:        serverDir,
		InstalledAt: time.Now(),
		RecipeSnapshot: map[string]interface{}{
			"name": "All the Mods 9",
		},
	}

	if err := tracker.AddInstallation(installation); err != nil {
		t.Fatalf("Failed to add installation: %v", err)
	}

	// Create uninstaller
	uninstaller, err := NewUninstaller()
	if err != nil {
		t.Fatalf("NewUninstaller failed: %v", err)
	}

	// Uninstall with matching modpack slug (case insensitive)
	opts := &Options{
		ServerDir:   serverDir,
		ModpackSlug: "ATM9",
		Force:       true,
	}

	_, err = uninstaller.Uninstall(opts)
	if err != nil {
		t.Errorf("Expected successful uninstall with matching slug, got error: %v", err)
	}

	// Clean up tracker
	tracker.RemoveInstallation(serverDir)
}

func TestDeterminePathsToRemoveAndPreserve(t *testing.T) {
	// Create test server directory
	tmpDir := t.TempDir()
	serverDir := filepath.Join(tmpDir, "server")

	// Create test structure
	if err := os.MkdirAll(filepath.Join(serverDir, "mods"), 0755); err != nil {
		t.Fatalf("Failed to create mods directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(serverDir, "world"), 0755); err != nil {
		t.Fatalf("Failed to create world directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "ops.json"), []byte("[]"), 0644); err != nil {
		t.Fatalf("Failed to create ops.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "start.sh"), []byte("#!/bin/bash"), 0755); err != nil {
		t.Fatalf("Failed to create start.sh: %v", err)
	}

	// Create uninstaller
	uninstaller, err := NewUninstaller()
	if err != nil {
		t.Fatalf("NewUninstaller failed: %v", err)
	}

	// Test with keepWorlds = true
	toRemove, toPreserve := uninstaller.determinePathsToRemoveAndPreserve(serverDir, true)

	// Check that mods and start.sh are in toRemove
	foundMods := false
	foundStartSh := false
	for _, path := range toRemove {
		if path == "mods" {
			foundMods = true
		}
		if path == "start.sh" {
			foundStartSh = true
		}
	}
	if !foundMods {
		t.Error("Expected 'mods' in toRemove")
	}
	if !foundStartSh {
		t.Error("Expected 'start.sh' in toRemove")
	}

	// Check that world and ops.json are in toPreserve
	foundWorld := false
	foundOps := false
	for _, path := range toPreserve {
		if path == "world" {
			foundWorld = true
		}
		if path == "ops.json" {
			foundOps = true
		}
	}
	if !foundWorld {
		t.Error("Expected 'world' in toPreserve")
	}
	if !foundOps {
		t.Error("Expected 'ops.json' in toPreserve")
	}

	// Test with keepWorlds = false
	toRemove, toPreserve = uninstaller.determinePathsToRemoveAndPreserve(serverDir, false)

	if len(toPreserve) != 0 {
		t.Errorf("Expected no preserved paths when keepWorlds=false, got %d", len(toPreserve))
	}
}

func TestUninstallEmptyDirectory(t *testing.T) {
	// Create empty test server directory
	tmpDir := t.TempDir()
	serverDir := filepath.Join(tmpDir, "server")

	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatalf("Failed to create server directory: %v", err)
	}

	// Create uninstaller
	uninstaller, err := NewUninstaller()
	if err != nil {
		t.Fatalf("NewUninstaller failed: %v", err)
	}

	// Uninstall empty directory
	opts := &Options{
		ServerDir: serverDir,
		Force:     true,
	}

	result, err := uninstaller.Uninstall(opts)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Should complete successfully with no files removed
	if len(result.RemovedPaths) != 0 {
		t.Error("Expected no paths to be removed from empty directory")
	}
}
