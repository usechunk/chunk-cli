package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexinslc/chunk/internal/cache"
	"github.com/spf13/cobra"
)

var (
	cleanupAll    bool
	cleanupDryRun bool
	cleanupStats  bool
)

var CleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up cached downloads",
	Long: `Remove old cached downloads and outdated recipe snapshots.

This command will:
  - Scan ~/.chunk/downloads/ for cached files
  - Identify downloads for outdated versions
  - Identify downloads for uninstalled modpacks
  - Identify failed downloads (partial files)
  - Calculate space that will be freed
  - Prompt before deletion

Examples:
  chunk cleanup              # Interactive cleanup
  chunk cleanup --all        # Remove all cached downloads
  chunk cleanup --dry-run    # Show what would be removed
  chunk cleanup -s           # Show cache statistics only`,
	RunE: runCleanup,
}

func init() {
	CleanupCmd.Flags().BoolVar(&cleanupAll, "all", false, "Remove all cached downloads")
	CleanupCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false, "Show what would be removed without deleting")
	CleanupCmd.Flags().BoolVarP(&cleanupStats, "stats", "s", false, "Show cache statistics only")

	// Suppress usage printing on errors
	CleanupCmd.SilenceUsage = true
	CleanupCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		cmd.Usage()
		return err
	})
}

func runCleanup(cmd *cobra.Command, args []string) error {
	manager, err := cache.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize cache manager: %w", err)
	}

	// Handle stats-only mode
	if cleanupStats {
		return showCacheStats(manager)
	}

	// Handle cleanup
	if cleanupAll {
		return cleanupAllFiles(manager)
	}

	return cleanupRemovableFiles(manager)
}

// showCacheStats displays cache statistics
func showCacheStats(manager *cache.Manager) error {
	totalSize, fileCount, err := manager.GetCacheSize()
	if err != nil {
		return fmt.Errorf("failed to get cache size: %w", err)
	}

	fmt.Println()
	fmt.Println("📊 Cache Statistics")
	fmt.Println()
	fmt.Printf("  Files:      %d\n", fileCount)
	fmt.Printf("  Total size: %s\n", formatSize(totalSize))
	fmt.Println()

	// Get detailed analysis
	stats, err := manager.AnalyzeCache()
	if err != nil {
		return fmt.Errorf("failed to analyze cache: %w", err)
	}

	if len(stats.FilesToRemove) > 0 {
		fmt.Println("Removable files:")
		if stats.OutdatedFiles > 0 {
			fmt.Printf("  Outdated versions:     %d\n", stats.OutdatedFiles)
		}
		if stats.UninstalledFiles > 0 {
			fmt.Printf("  Uninstalled modpacks:  %d\n", stats.UninstalledFiles)
		}
		if stats.PartialFiles > 0 {
			fmt.Printf("  Failed downloads:      %d\n", stats.PartialFiles)
		}
		fmt.Printf("  Reclaimable space:     %s\n", formatSize(stats.TotalSize))
		fmt.Println()
		fmt.Println("Run 'chunk cleanup' to remove these files.")
	} else {
		fmt.Println("✓ No removable files found")
	}
	fmt.Println()

	return nil
}

// cleanupAllFiles removes all cached files
func cleanupAllFiles(manager *cache.Manager) error {
	cachedFiles, err := manager.ListCachedFiles()
	if err != nil {
		return fmt.Errorf("failed to list cached files: %w", err)
	}

	if len(cachedFiles) == 0 {
		fmt.Println()
		fmt.Println("✓ Cache is already empty")
		fmt.Println()
		return nil
	}

	var totalSize int64
	for _, file := range cachedFiles {
		totalSize += file.Size
	}

	fmt.Println()
	fmt.Printf("The following will be removed:\n\n")

	for _, file := range cachedFiles {
		displayCachedFile(file)
	}

	fmt.Printf("\nTotal: %s (%d files)\n\n", formatSize(totalSize), len(cachedFiles))

	// Dry run mode
	if cleanupDryRun {
		fmt.Println("(dry-run mode - no files were deleted)")
		fmt.Println()
		return nil
	}

	// Prompt for confirmation
	if !promptConfirm("Proceed?") {
		fmt.Println("Cleanup cancelled")
		fmt.Println()
		return nil
	}

	// Perform cleanup
	fmt.Printf("\nRemoving %d files...\n", len(cachedFiles))
	if err := manager.CleanupAll(); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	fmt.Printf("✓ Freed %s\n", formatSize(totalSize))
	fmt.Println()

	return nil
}

// cleanupRemovableFiles removes outdated and uninstalled files
func cleanupRemovableFiles(manager *cache.Manager) error {
	stats, err := manager.AnalyzeCache()
	if err != nil {
		return fmt.Errorf("failed to analyze cache: %w", err)
	}

	if len(stats.FilesToRemove) == 0 {
		fmt.Println()
		fmt.Println("✓ No removable files found")
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Printf("The following will be removed:\n\n")

	for _, file := range stats.FilesToRemove {
		displayCachedFile(file)
	}

	fmt.Printf("\nTotal: %s (%d files)\n\n", formatSize(stats.TotalSize), len(stats.FilesToRemove))

	// Dry run mode
	if cleanupDryRun {
		fmt.Println("(dry-run mode - no files were deleted)")
		fmt.Println()
		return nil
	}

	// Prompt for confirmation
	if !promptConfirm("Proceed?") {
		fmt.Println("Cleanup cancelled")
		fmt.Println()
		return nil
	}

	// Perform cleanup
	fmt.Printf("\nRemoving %d files...\n", len(stats.FilesToRemove))
	if err := manager.CleanupFiles(stats.FilesToRemove); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	fmt.Printf("✓ Freed %s\n", formatSize(stats.TotalSize))
	fmt.Println()

	return nil
}

// displayCachedFile displays information about a cached file
func displayCachedFile(file *cache.CachedFile) {
	filename := filepath.Base(file.Path)
	sizeStr := formatSize(file.Size)

	var reason string
	switch file.Reason {
	case "partial":
		reason = "failed download"
	case "outdated":
		reason = "outdated"
	case "uninstalled":
		reason = "no longer installed"
	default:
		// This should not happen as Reason is always set by AnalyzeCache
		reason = "unknown"
	}

	fmt.Printf("  %s (%s) - %s\n", filename, sizeStr, reason)
}

// formatSize formats a byte size in human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	// Calculate appropriate unit
	sizeKB := float64(bytes) / unit
	sizeMB := sizeKB / unit
	sizeGB := sizeMB / unit

	if sizeGB >= 1.0 {
		return fmt.Sprintf("%.2f GB", sizeGB)
	} else if sizeMB >= 1.0 {
		return fmt.Sprintf("%.0f MB", sizeMB)
	}
	return fmt.Sprintf("%.0f KB", sizeKB)
}

// promptConfirm prompts the user for confirmation
func promptConfirm(message string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [Y/n]: ", message)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "" || response == "y" || response == "yes"
}
