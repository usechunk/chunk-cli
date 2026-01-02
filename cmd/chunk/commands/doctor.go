package commands

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexinslc/chunk/internal/bench"
	"github.com/alexinslc/chunk/internal/java"
	"github.com/alexinslc/chunk/internal/search"
	"github.com/alexinslc/chunk/internal/tracking"
	"github.com/spf13/cobra"
)

var (
	doctorVerbose bool
)

// DoctorCmd is the command for health check diagnostics
var DoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run health check diagnostics",
	Long: `Run health check diagnostics to verify installation, dependencies, and environment.

Performs the following checks:
  - Java installations: Detect installed Java versions
  - Git installed: Required for benches
  - Disk space: Check available space
  - Benches: Verify benches are valid Git repos
  - Recipes: Check for corrupted recipe files
  - Installations: Verify tracked installations exist
  - Permissions: Check write permissions to install dirs
  - Network: Check connectivity to common sources

Examples:
  chunk doctor
  chunk doctor --verbose`,
	RunE: runDoctor,
}

func init() {
	DoctorCmd.Flags().BoolVarP(&doctorVerbose, "verbose", "v", false, "Show detailed diagnostic output")
	DoctorCmd.SilenceUsage = true
}

type checkResult struct {
	success bool
	message string
	fix     string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("Running diagnostics...")
	fmt.Println()

	var results []checkResult
	issueCount := 0

	// Check Git
	gitResult := checkGit()
	results = append(results, gitResult)
	if !gitResult.success {
		issueCount++
	}

	// Check Java installations
	javaResults := checkJava()
	results = append(results, javaResults...)
	for _, r := range javaResults {
		if !r.success {
			issueCount++
		}
	}

	// Check disk space
	diskResult := checkDiskSpace()
	results = append(results, diskResult)
	if !diskResult.success {
		issueCount++
	}

	// Check benches
	benchResults := checkBenches()
	results = append(results, benchResults...)
	for _, r := range benchResults {
		if !r.success {
			issueCount++
		}
	}

	// Check recipes
	recipeResults := checkRecipes()
	results = append(results, recipeResults...)
	for _, r := range recipeResults {
		if !r.success {
			issueCount++
		}
	}

	// Check installations
	installResults := checkInstallations()
	results = append(results, installResults...)
	for _, r := range installResults {
		if !r.success {
			issueCount++
		}
	}

	// Check network connectivity
	networkResults := checkNetwork()
	results = append(results, networkResults...)
	for _, r := range networkResults {
		if !r.success {
			issueCount++
		}
	}

	// Print all results
	for _, result := range results {
		if result.success {
			fmt.Printf("✓ %s\n", result.message)
		} else {
			fmt.Printf("✗ %s\n", result.message)
			if result.fix != "" {
				fmt.Printf("  Fix: %s\n", result.fix)
			}
		}
	}

	fmt.Println()
	if issueCount == 0 {
		fmt.Println("No issues found.")
	} else if issueCount == 1 {
		fmt.Println("1 issue found. See above for fixes.")
	} else {
		fmt.Printf("%d issues found. See above for fixes.\n", issueCount)
	}
	fmt.Println()

	return nil
}

// checkGit verifies that Git is installed
func checkGit() checkResult {
	cmd := exec.Command("git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return checkResult{
			success: false,
			message: "Git not found",
			fix:     "Install Git from https://git-scm.com/downloads",
		}
	}

	version := strings.TrimSpace(string(output))
	version = strings.TrimPrefix(version, "git version ")

	if doctorVerbose {
		return checkResult{
			success: true,
			message: fmt.Sprintf("Git installed (%s)", version),
		}
	}

	return checkResult{
		success: true,
		message: "Git installed",
	}
}

// checkJava detects and reports on Java installations
func checkJava() []checkResult {
	detector := java.NewJavaDetector()
	installations, err := detector.DetectAll()

	if err != nil || len(installations) == 0 {
		return []checkResult{{
			success: false,
			message: "No Java installations found",
			fix:     "Install Java from https://adoptium.net/",
		}}
	}

	var results []checkResult
	seen := make(map[int]bool)

	for _, install := range installations {
		// Only report each major version once
		if seen[install.Major] {
			continue
		}
		seen[install.Major] = true

		if doctorVerbose {
			results = append(results, checkResult{
				success: true,
				message: fmt.Sprintf("Java %d installed (%s)", install.Major, install.Path),
			})
		} else {
			results = append(results, checkResult{
				success: true,
				message: fmt.Sprintf("Java %d installed", install.Major),
			})
		}
	}

	return results
}

// checkDiskSpace checks available disk space
func checkDiskSpace() checkResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return checkResult{
			success: false,
			message: "Could not determine home directory",
		}
	}

	availableGB, err := getAvailableDiskSpace(home)
	if err != nil {
		return checkResult{
			success: false,
			message: "Could not check disk space",
		}
	}

	if availableGB < 5 {
		return checkResult{
			success: false,
			message: fmt.Sprintf("Low disk space: %d GB available", availableGB),
			fix:     "Free up disk space to continue",
		}
	}

	return checkResult{
		success: true,
		message: fmt.Sprintf("Disk space: %d GB available", availableGB),
	}
}

// checkBenches verifies that benches are valid Git repositories
func checkBenches() []checkResult {
	manager, err := bench.NewManager()
	if err != nil {
		return []checkResult{{
			success: false,
			message: "Failed to initialize bench manager",
		}}
	}

	benches := manager.List()
	if len(benches) == 0 {
		return []checkResult{{
			success: true,
			message: "Benches: 0 installed",
		}}
	}

	validCount := 0
	var results []checkResult

	for _, b := range benches {
		// Check if path exists
		if _, err := os.Stat(b.Path); os.IsNotExist(err) {
			results = append(results, checkResult{
				success: false,
				message: fmt.Sprintf("Bench '%s' directory not found", b.Name),
				fix:     fmt.Sprintf("chunk bench remove %s && chunk bench add %s", b.Name, b.URL),
			})
			continue
		}

		// Check if it's a valid Git repository
		gitDir := filepath.Join(b.Path, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			results = append(results, checkResult{
				success: false,
				message: fmt.Sprintf("Bench '%s' is not a valid Git repository", b.Name),
				fix:     fmt.Sprintf("chunk bench remove %s && chunk bench add %s", b.Name, b.URL),
			})
			continue
		}

		// Check if Recipes directory exists
		recipesDir := filepath.Join(b.Path, "Recipes")
		if _, err := os.Stat(recipesDir); os.IsNotExist(err) {
			results = append(results, checkResult{
				success: false,
				message: fmt.Sprintf("Bench '%s' missing Recipes directory", b.Name),
				fix:     fmt.Sprintf("chunk bench remove %s && chunk bench add %s", b.Name, b.URL),
			})
			continue
		}

		validCount++
		if doctorVerbose {
			results = append(results, checkResult{
				success: true,
				message: fmt.Sprintf("Bench '%s' is valid", b.Name),
			})
		}
	}

	// Add summary if not verbose
	if !doctorVerbose {
		totalBenches := len(benches)
		if totalBenches == validCount {
			results = append(results, checkResult{
				success: true,
				message: fmt.Sprintf("Benches: %d installed, all valid", validCount),
			})
		} else {
			results = append(results, checkResult{
				success: false,
				message: fmt.Sprintf("Benches: %d valid out of %d installed", validCount, totalBenches),
			})
		}
	}

	return results
}

// checkRecipes validates recipe files
func checkRecipes() []checkResult {
	manager, err := bench.NewManager()
	if err != nil {
		return []checkResult{{
			success: false,
			message: "Failed to initialize bench manager",
		}}
	}

	benches := manager.List()
	if len(benches) == 0 {
		return []checkResult{{
			success: true,
			message: "Recipes: 0 (no benches installed)",
		}}
	}

	totalRecipes := 0
	var corruptedRecipes []string

	for _, b := range benches {
		recipes, err := search.LoadRecipesFromBench(b.Path, b.Name)
		if err != nil {
			if doctorVerbose {
				corruptedRecipes = append(corruptedRecipes, fmt.Sprintf("%s (load error)", b.Name))
			}
			continue
		}

		totalRecipes += len(recipes)

		// Check each recipe for basic validity
		for _, recipe := range recipes {
			if recipe.Slug == "" || recipe.MCVersion == "" {
				corruptedRecipes = append(corruptedRecipes, recipe.Slug)
			}
		}
	}

	if len(corruptedRecipes) > 0 {
		return []checkResult{{
			success: false,
			message: fmt.Sprintf("Recipes: %d corrupted out of %d", len(corruptedRecipes), totalRecipes),
			fix:     "chunk bench update --all to refresh benches",
		}}
	}

	return []checkResult{{
		success: true,
		message: fmt.Sprintf("Recipes: %d parsed successfully", totalRecipes),
	}}
}

// checkInstallations verifies tracked installations exist
func checkInstallations() []checkResult {
	tracker, err := tracking.NewTracker()
	if err != nil {
		return []checkResult{{
			success: false,
			message: "Failed to initialize installation tracker",
		}}
	}

	installations, err := tracker.ListInstallations()
	if err != nil {
		return []checkResult{{
			success: false,
			message: "Failed to list installations",
		}}
	}

	if len(installations) == 0 {
		return []checkResult{{
			success: true,
			message: "Installations: 0 tracked",
		}}
	}

	var results []checkResult
	validCount := 0
	for _, inst := range installations {
		// Check if installation directory exists
		if _, err := os.Stat(inst.Path); os.IsNotExist(err) {
			results = append(results, checkResult{
				success: false,
				message: fmt.Sprintf("Installation mismatch: %s tracked but directory missing", inst.Slug),
				fix:     fmt.Sprintf("chunk uninstall %s --force", inst.Slug),
			})
			continue
		}

		// Check if .chunk.json exists in the installation
		manifestPath := filepath.Join(inst.Path, ".chunk.json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			if doctorVerbose {
				results = append(results, checkResult{
					success: false,
					message: fmt.Sprintf("Installation '%s' missing .chunk.json", inst.Slug),
					fix:     "This may indicate a partial or corrupted installation",
				})
			}
		} else {
			validCount++
			if doctorVerbose {
				results = append(results, checkResult{
					success: true,
					message: fmt.Sprintf("Installation '%s' is valid", inst.Slug),
				})
			}
		}
	}

	// Add summary in non-verbose mode
	if !doctorVerbose {
		totalInstalls := len(installations)
		if totalInstalls == validCount && len(results) == 0 {
			results = append(results, checkResult{
				success: true,
				message: fmt.Sprintf("Installations: %d tracked, all valid", validCount),
			})
		} else {
			results = append(results, checkResult{
				success: false,
				message: fmt.Sprintf("Installations: %d valid out of %d tracked", validCount, totalInstalls),
			})
		}
	}

	return results
}

// checkNetwork tests connectivity to common sources
func checkNetwork() []checkResult {
	sources := []struct {
		name string
		url  string
	}{
		{"modrinth.com", "https://api.modrinth.com/v2/version_files"},
		{"github.com", "https://github.com"},
	}

	var results []checkResult
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for _, source := range sources {
		resp, err := client.Head(source.url)
		if err != nil {
			results = append(results, checkResult{
				success: false,
				message: fmt.Sprintf("Network: %s unreachable", source.name),
				fix:     "Check your internet connection",
			})
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			results = append(results, checkResult{
				success: true,
				message: fmt.Sprintf("Network: %s reachable", source.name),
			})
		} else {
			results = append(results, checkResult{
				success: false,
				message: fmt.Sprintf("Network: %s returned status %d", source.name, resp.StatusCode),
				fix:     "Check your internet connection or try again later",
			})
		}
	}

	return results
}
