package uninstall

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexinslc/chunk/internal/tracking"
	"github.com/alexinslc/chunk/internal/ui"
)

// Options contains configuration for uninstalling a modpack
type Options struct {
	// ServerDir is the directory containing the modpack installation
	ServerDir string
	// KeepWorlds preserves world data during uninstall
	KeepWorlds bool
	// Force skips confirmation prompts
	Force bool
}

// Result contains information about the uninstall operation
type Result struct {
	RemovedPaths   []string
	PreservedPaths []string
}

// Uninstaller handles modpack uninstallation
type Uninstaller struct {
	tracker *tracking.Tracker
}

// NewUninstaller creates a new uninstaller
func NewUninstaller() (*Uninstaller, error) {
	tracker, err := tracking.NewTracker()
	if err != nil {
		return nil, fmt.Errorf("failed to create tracker: %w", err)
	}

	return &Uninstaller{
		tracker: tracker,
	}, nil
}

// Uninstall removes a modpack installation
func (u *Uninstaller) Uninstall(opts *Options) (*Result, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	// Normalize server directory path
	serverDir, err := filepath.Abs(opts.ServerDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server directory: %w", err)
	}

	// Check if directory exists
	if _, err := os.Stat(serverDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("server directory does not exist: %s", serverDir)
	}

	// Get installation info from tracker
	installation, err := u.tracker.GetInstallation(serverDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get installation info: %w", err)
	}

	// Determine what to remove and what to preserve
	toRemove, toPreserve := u.determinePathsToRemoveAndPreserve(serverDir, opts.KeepWorlds)

	// If not force mode and no explicit keep-worlds flag, prompt user
	keepWorlds := opts.KeepWorlds
	if !opts.Force && !opts.KeepWorlds {
		keepWorlds, err = u.promptKeepWorlds()
		if err != nil {
			return nil, fmt.Errorf("failed to get user input: %w", err)
		}

		// Recalculate paths based on user choice
		toRemove, toPreserve = u.determinePathsToRemoveAndPreserve(serverDir, keepWorlds)
	}

	// Show what will happen
	u.displayUninstallPlan(installation, toRemove, toPreserve)

	// Confirm if not in force mode
	if !opts.Force {
		confirmed, err := u.promptConfirmation()
		if err != nil {
			return nil, fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			return nil, fmt.Errorf("uninstall cancelled by user")
		}
	}

	// Perform the uninstall
	result := &Result{
		RemovedPaths:   []string{},
		PreservedPaths: toPreserve,
	}

	for _, relPath := range toRemove {
		fullPath := filepath.Join(serverDir, relPath)
		if err := os.RemoveAll(fullPath); err != nil {
			ui.PrintWarning(fmt.Sprintf("Failed to remove %s: %v", relPath, err))
		} else {
			result.RemovedPaths = append(result.RemovedPaths, relPath)
			ui.PrintSuccess(fmt.Sprintf("Removed: %s", relPath))
		}
	}

	// Remove from tracking
	if err := u.tracker.RemoveInstallation(serverDir); err != nil {
		ui.PrintWarning(fmt.Sprintf("Failed to update installation tracking: %v", err))
	}

	return result, nil
}

// determinePathsToRemoveAndPreserve decides which paths to remove and preserve
func (u *Uninstaller) determinePathsToRemoveAndPreserve(serverDir string, keepWorlds bool) (toRemove, toPreserve []string) {
	// Paths that are always considered for removal
	modpackPaths := []string{
		"mods",
		"config",
		"libraries",
		"defaultconfigs",
		"kubejs",
		"scripts",
		"resources",
		"resourcepacks",
		"shaderpacks",
		"start.sh",
		"start.bat",
		"forge-installer.jar",
		"fabric-installer.jar",
		"neoforge-installer.jar",
		"server.jar",
		"forge.jar",
		"fabric-server-launch.jar",
		"run.sh",
		"run.bat",
		"user_jvm_args.txt",
		"eula.txt",
	}

	// Paths to preserve if keepWorlds is true
	preservePaths := []string{
		"world",
		"world_nether",
		"world_the_end",
		"server.properties",
		"whitelist.json",
		"ops.json",
		"banned-players.json",
		"banned-ips.json",
		"usercache.json",
	}

	toRemove = []string{}
	toPreserve = []string{}

	// Check which modpack files exist
	for _, relPath := range modpackPaths {
		fullPath := filepath.Join(serverDir, relPath)
		if _, err := os.Stat(fullPath); err == nil {
			toRemove = append(toRemove, relPath)
		}
	}

	// If keeping worlds, check which preserve paths exist and exclude from removal
	if keepWorlds {
		for _, relPath := range preservePaths {
			fullPath := filepath.Join(serverDir, relPath)
			if _, err := os.Stat(fullPath); err == nil {
				toPreserve = append(toPreserve, relPath)
			}
		}
	}

	return toRemove, toPreserve
}

// promptKeepWorlds asks the user if they want to keep world data
func (u *Uninstaller) promptKeepWorlds() (bool, error) {
	fmt.Println()
	fmt.Print("Keep world and player data? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input = strings.TrimSpace(strings.ToLower(input))

	// Default to yes if user just presses enter
	if input == "" || input == "y" || input == "yes" {
		return true, nil
	}

	return false, nil
}

// promptConfirmation asks for final confirmation before uninstalling
func (u *Uninstaller) promptConfirmation() (bool, error) {
	fmt.Println()
	fmt.Print("Proceed with uninstall? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" {
		return true, nil
	}

	return false, nil
}

// displayUninstallPlan shows what will be removed and preserved
func (u *Uninstaller) displayUninstallPlan(installation *tracking.Installation, toRemove, toPreserve []string) {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Uninstall Plan")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if installation != nil {
		fmt.Printf("Modpack: %s\n", installation.Slug)
		fmt.Printf("Version: %s\n", installation.Version)
		fmt.Printf("Path:    %s\n", installation.Path)
		fmt.Println()
	}

	if len(toRemove) > 0 {
		fmt.Println("Removing:")
		for _, path := range toRemove {
			fmt.Printf("  - %s\n", path)
		}
	}

	if len(toPreserve) > 0 {
		fmt.Println()
		fmt.Println("Preserving:")
		for _, path := range toPreserve {
			fmt.Printf("  - %s\n", path)
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
