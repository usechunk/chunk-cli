package commands

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alexinslc/chunk/internal/bench"
	"github.com/alexinslc/chunk/internal/search"
	"github.com/alexinslc/chunk/internal/ui"
	"github.com/alexinslc/chunk/internal/validation"
	"github.com/spf13/cobra"
)

var (
	templateRecipe string
	outputDir      string
	validateAll    bool
)

var RecipeCmd = &cobra.Command{
	Use:   "recipe",
	Short: "Manage modpack recipes",
	Long: `Create and manage modpack recipe JSON files.

Recipes are JSON files that describe how to install a modpack.
They are stored in recipe benches (Git repositories).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var recipeCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new recipe interactively",
	Long: `Create a new recipe JSON file through an interactive wizard.

The command will guide you through entering all required fields,
download the modpack file to calculate its checksum, and generate
a properly formatted recipe JSON file.

Example:
  chunk recipe create
  chunk recipe create --template atm9
  chunk recipe create --output ./my-recipes`,
	RunE: runRecipeCreate,
}

var recipeValidateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate a recipe JSON file",
	Long: `Validate a recipe JSON file against the schema and check for common issues.

This command performs the following checks:
- JSON schema validation
- Required fields presence
- URL reachability (download URL returns 200)
- Checksum verification (SHA-256 matches download)
- Version format validation (semver, Minecraft version)
- Loader compatibility
- License SPDX identifier validation
- Naming consistency (slug matches filename)

Example:
  chunk recipe validate my-pack.json
  chunk recipe validate .`,
	Args: cobra.ExactArgs(1),
	RunE: runRecipeValidate,
}

func runRecipeValidate(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Check if path is a directory
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to access path: %w", err)
	}

	var filesToValidate []string
	if fileInfo.IsDir() {
		// Validate all JSON files in directory
		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".json" || ext == ".yaml" || ext == ".yml" {
				filesToValidate = append(filesToValidate, filepath.Join(path, entry.Name()))
			}
		}

		if len(filesToValidate) == 0 {
			return fmt.Errorf("no recipe files found in directory: %s", path)
		}
	} else {
		// Validate single file
		filesToValidate = []string{path}
	}

	validator := validation.NewRecipeValidator()
	totalErrors := 0
	totalWarnings := 0

	for _, filePath := range filesToValidate {
		fmt.Printf("\nValidating %s...\n", filepath.Base(filePath))
		fmt.Println()

		// Load recipe
		recipe, err := search.LoadRecipe(filePath, "")
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to load recipe: %v", err))
			totalErrors++
			continue
		}

		// Validate recipe
		result, downloadSize := validator.ValidateRecipeWithNetwork(recipe, filePath)

		// Print validation results
		printValidationResults(recipe, result, downloadSize)

		totalErrors += len(result.Errors)
		totalWarnings += len(result.Warnings)
	}

	// Print summary if multiple files
	if len(filesToValidate) > 1 {
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("Validated %d file(s)\n", len(filesToValidate))
		if totalErrors > 0 {
			ui.PrintError(fmt.Sprintf("%d error(s) found", totalErrors))
		}
		if totalWarnings > 0 {
			ui.PrintWarning(fmt.Sprintf("%d warning(s) found", totalWarnings))
		}
		if totalErrors == 0 && totalWarnings == 0 {
			ui.PrintSuccess("All files passed validation!")
		}
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	}

	if totalErrors > 0 {
		return fmt.Errorf("validation failed with %d error(s)", totalErrors)
	}

	return nil
}

func printValidationResults(recipe *search.Recipe, result *validation.ValidationResult, downloadSize int64) {
	// Track checks performed
	checks := []struct {
		name    string
		passed  bool
		message string
	}{
		{"Required fields present", !hasErrorForField(result, "name", "mc_version", "loader", "download_url"), ""},
	}

	// Check URL reachability
	urlError := hasErrorForField(result, "download_url")
	if !urlError && recipe.DownloadURL != "" {
		sizeStr := ""
		if downloadSize > 0 {
			sizeMB := downloadSize / (1024 * 1024)
			sizeStr = fmt.Sprintf(" (%d MB)", sizeMB)
		}
		checks = append(checks, struct {
			name    string
			passed  bool
			message string
		}{"Download URL reachable", true, sizeStr})
	} else if !urlError && recipe.DownloadURL == "" {
		checks = append(checks, struct {
			name    string
			passed  bool
			message string
		}{"Download URL reachable", false, " (missing)"})
	} else {
		checks = append(checks, struct {
			name    string
			passed  bool
			message string
		}{"Download URL reachable", false, ""})
	}

	// Check checksum
	checksumError := hasErrorForField(result, "sha256")
	if !checksumError && recipe.SHA256 != "" {
		checks = append(checks, struct {
			name    string
			passed  bool
			message string
		}{"Checksum matches", true, ""})
	} else if !checksumError && recipe.SHA256 == "" {
		checks = append(checks, struct {
			name    string
			passed  bool
			message string
		}{"Checksum provided", false, " (recommended)"})
	} else {
		checks = append(checks, struct {
			name    string
			passed  bool
			message string
		}{"Checksum matches", false, ""})
	}

	// Check Minecraft version
	mcVersionError := hasErrorForField(result, "mc_version")
	if !mcVersionError && recipe.MCVersion != "" {
		checks = append(checks, struct {
			name    string
			passed  bool
			message string
		}{"Minecraft version valid", true, fmt.Sprintf(" (%s)", recipe.MCVersion)})
	} else {
		checks = append(checks, struct {
			name    string
			passed  bool
			message string
		}{"Minecraft version valid", false, ""})
	}

	// Check loader
	loaderError := hasErrorForField(result, "loader")
	if !loaderError && recipe.Loader != "" {
		loaderStr := recipe.Loader
		if recipe.LoaderVersion != "" {
			loaderStr += " " + recipe.LoaderVersion
		}
		checks = append(checks, struct {
			name    string
			passed  bool
			message string
		}{"Loader version valid", true, fmt.Sprintf(" (%s)", loaderStr)})
	} else {
		checks = append(checks, struct {
			name    string
			passed  bool
			message string
		}{"Loader version valid", false, ""})
	}

	// Check license
	licenseError := hasErrorForField(result, "license")
	if !licenseError && recipe.License != "" {
		checks = append(checks, struct {
			name    string
			passed  bool
			message string
		}{"License valid SPDX", true, fmt.Sprintf(" (%s)", recipe.License)})
	} else if recipe.License == "" {
		checks = append(checks, struct {
			name    string
			passed  bool
			message string
		}{"License valid SPDX", false, " (recommended)"})
	} else {
		checks = append(checks, struct {
			name    string
			passed  bool
			message string
		}{"License valid SPDX", false, ""})
	}

	// Print checks
	for _, check := range checks {
		if check.passed {
			ui.PrintSuccess(check.name + check.message)
		} else {
			ui.PrintError(check.name + check.message)
		}
	}

	// Print specific errors
	if len(result.Errors) > 0 {
		fmt.Println()
		for _, err := range result.Errors {
			ui.PrintError(fmt.Sprintf("%s: %s", err.Field, err.Message))
			if err.Suggestion != "" {
				fmt.Printf("  Suggested: %s\n", err.Suggestion)
			}
		}
	}

	// Print warnings
	if len(result.Warnings) > 0 {
		fmt.Println()
		for _, warn := range result.Warnings {
			ui.PrintWarning(fmt.Sprintf("%s: %s", warn.Field, warn.Message))
		}
	}

	// Print summary
	fmt.Println()
	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		ui.PrintSuccess("All checks passed!")
	} else {
		fmt.Printf("%d error(s), %d warning(s)\n", len(result.Errors), len(result.Warnings))
		if len(result.Errors) > 0 {
			fmt.Println()
			fmt.Println("Fix errors and try again.")
		}
	}
}

func hasErrorForField(result *validation.ValidationResult, fields ...string) bool {
	for _, field := range fields {
		for _, err := range result.Errors {
			if err.Field == field {
				return true
			}
		}
	}
	return false
}

func runRecipeCreate(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("🧪 Chunk Recipe Creator")
	fmt.Println()
	fmt.Println("This wizard will guide you through creating a recipe JSON file.")
	fmt.Println()

	var recipe search.Recipe

	// Load template if specified
	if templateRecipe != "" {
		template, err := loadTemplateRecipe(templateRecipe)
		if err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not load template: %v", err))
			fmt.Println()
		} else {
			recipe = *template
			fmt.Printf("✓ Loaded template from: %s\n\n", templateRecipe)
		}
	}

	// Prompt for name
	name, err := promptString(reader, "Name", recipe.Name, true)
	if err != nil {
		return err
	}
	recipe.Name = name

	// Auto-generate slug from name
	defaultSlug := generateSlug(name)
	if recipe.Slug == "" {
		recipe.Slug = defaultSlug
	}
	slug, err := promptString(reader, "Slug", recipe.Slug, false)
	if err != nil {
		return err
	}
	if slug == "" {
		slug = defaultSlug
	}
	recipe.Slug = slug

	// Prompt for description
	description, err := promptString(reader, "Description", recipe.Description, true)
	if err != nil {
		return err
	}
	recipe.Description = description

	// Prompt for Minecraft version
	mcVersion, err := promptString(reader, "Minecraft version", recipe.MCVersion, true)
	if err != nil {
		return err
	}
	recipe.MCVersion = mcVersion

	// Prompt for loader type
	loader, err := promptLoader(reader, recipe.Loader)
	if err != nil {
		return err
	}
	recipe.Loader = loader

	// Prompt for loader version
	loaderVersion, err := promptString(reader, "Loader version", recipe.LoaderVersion, true)
	if err != nil {
		return err
	}
	recipe.LoaderVersion = loaderVersion

	// Prompt for download URL
	downloadURL, err := promptURL(reader, "Download URL", recipe.DownloadURL)
	if err != nil {
		return err
	}
	recipe.DownloadURL = downloadURL

	// Calculate checksum from download URL
	fmt.Println()
	fmt.Println("Downloading to calculate checksum...")
	checksum, size, err := downloadAndCalculateChecksum(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}
	recipe.SHA256 = checksum
	recipe.DownloadSizeMB = int(size / (1024 * 1024))
	ui.PrintSuccess(fmt.Sprintf("SHA-256: %s", checksum))
	fmt.Println()

	// Prompt for RAM requirements
	ram, err := promptInt(reader, "RAM required (GB)", recipe.RecommendedRAMGB, false)
	if err != nil {
		return err
	}
	recipe.RecommendedRAMGB = ram

	// Prompt for disk space
	disk, err := promptInt(reader, "Disk space (GB)", recipe.DiskSpaceGB, false)
	if err != nil {
		return err
	}
	recipe.DiskSpaceGB = disk

	// Prompt for license
	license, err := promptLicense(reader, recipe.License)
	if err != nil {
		return err
	}
	recipe.License = license

	// Prompt for optional homepage
	homepage, err := promptString(reader, "Homepage (optional)", recipe.Homepage, false)
	if err != nil {
		return err
	}
	recipe.Homepage = homepage

	// Prompt for optional author
	author, err := promptString(reader, "Author (optional)", recipe.Author, false)
	if err != nil {
		return err
	}
	recipe.Author = author

	// Generate the recipe JSON
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📝 Recipe Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Name:         %s\n", recipe.Name)
	fmt.Printf("Slug:         %s\n", recipe.Slug)
	fmt.Printf("Description:  %s\n", recipe.Description)
	fmt.Printf("Minecraft:    %s\n", recipe.MCVersion)
	fmt.Printf("Loader:       %s %s\n", recipe.Loader, recipe.LoaderVersion)
	fmt.Printf("Download:     %s\n", recipe.DownloadURL)
	fmt.Printf("Size:         %dMB\n", recipe.DownloadSizeMB)
	fmt.Printf("SHA-256:      %s\n", recipe.SHA256)
	if recipe.RecommendedRAMGB > 0 {
		fmt.Printf("RAM:          %dGB\n", recipe.RecommendedRAMGB)
	}
	if recipe.DiskSpaceGB > 0 {
		fmt.Printf("Disk:         %dGB\n", recipe.DiskSpaceGB)
	}
	fmt.Printf("License:      %s\n", recipe.License)
	if recipe.Homepage != "" {
		fmt.Printf("Homepage:     %s\n", recipe.Homepage)
	}
	if recipe.Author != "" {
		fmt.Printf("Author:       %s\n", recipe.Author)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Save the recipe
	filename := recipe.Slug + ".json"
	outDir := outputDir
	if outDir == "" {
		outDir = "."
	}

	outputPath := filepath.Join(outDir, filename)
	if err := saveRecipe(&recipe, outputPath); err != nil {
		return fmt.Errorf("failed to save recipe: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Recipe created: %s", outputPath))
	fmt.Println()
	fmt.Println("To submit this recipe:")
	fmt.Println("  1. Fork the repository: https://github.com/usechunk/recipes")
	fmt.Println("  2. Add your recipe to the Recipes/ directory")
	fmt.Println("  3. Open a pull request")
	fmt.Println()

	return nil
}

func promptString(reader *bufio.Reader, prompt string, defaultValue string, required bool) (string, error) {
	suffix := ""
	if defaultValue != "" {
		suffix = fmt.Sprintf(" [%s]", defaultValue)
	}
	fmt.Printf("%s%s: ", prompt, suffix)

	text, err := reader.ReadString('\n')
	if err != nil {
		// If EOF and we have a default value, use it
		if err == io.EOF && defaultValue != "" {
			fmt.Println() // Add newline
			return defaultValue, nil
		}
		// If EOF and not required, return empty
		if err == io.EOF && !required {
			fmt.Println() // Add newline
			return "", nil
		}
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	text = strings.TrimSpace(text)
	if text == "" {
		if defaultValue != "" {
			return defaultValue, nil
		}
		if required {
			return "", fmt.Errorf("required field: %s", prompt)
		}
		return "", nil
	}

	return text, nil
}

func promptInt(reader *bufio.Reader, prompt string, defaultValue int, required bool) (int, error) {
	suffix := ""
	if defaultValue > 0 {
		suffix = fmt.Sprintf(" [%d]", defaultValue)
	}
	fmt.Printf("%s%s: ", prompt, suffix)

	text, err := reader.ReadString('\n')
	if err != nil {
		// If EOF and we have a default value, use it
		if err == io.EOF && defaultValue > 0 {
			fmt.Println() // Add newline
			return defaultValue, nil
		}
		// If EOF and not required, return 0
		if err == io.EOF && !required {
			fmt.Println() // Add newline
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read input: %w", err)
	}

	text = strings.TrimSpace(text)
	if text == "" {
		if defaultValue > 0 {
			return defaultValue, nil
		}
		if required {
			return 0, fmt.Errorf("required field: %s", prompt)
		}
		return 0, nil
	}

	value, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", text)
	}

	return value, nil
}

func promptLoader(reader *bufio.Reader, defaultValue string) (string, error) {
	suffix := ""
	if defaultValue != "" {
		suffix = fmt.Sprintf(" [%s]", defaultValue)
	}
	fmt.Printf("Loader [forge/fabric/neoforge]%s: ", suffix)

	text, err := reader.ReadString('\n')
	if err != nil {
		// If EOF and we have a default value, use it
		if err == io.EOF && defaultValue != "" {
			fmt.Println() // Add newline
			return defaultValue, nil
		}
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		if defaultValue != "" {
			return defaultValue, nil
		}
		return "", fmt.Errorf("loader type is required")
	}

	validLoaders := map[string]bool{
		"forge":    true,
		"fabric":   true,
		"neoforge": true,
	}

	if !validLoaders[text] {
		return "", fmt.Errorf("invalid loader: %s (must be forge, fabric, or neoforge)", text)
	}

	return text, nil
}

func promptLicense(reader *bufio.Reader, defaultValue string) (string, error) {
	suffix := ""
	if defaultValue != "" {
		suffix = fmt.Sprintf(" [%s]", defaultValue)
	}
	fmt.Printf("License [MIT/GPL-3.0/ARR]%s: ", suffix)

	text, err := reader.ReadString('\n')
	if err != nil {
		// If EOF, use default or ARR
		if err == io.EOF {
			fmt.Println() // Add newline
			if defaultValue != "" {
				return defaultValue, nil
			}
			return "ARR", nil
		}
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	text = strings.TrimSpace(text)
	if text == "" {
		if defaultValue != "" {
			return defaultValue, nil
		}
		return "ARR", nil // Default to All Rights Reserved
	}

	return text, nil
}

func promptURL(reader *bufio.Reader, prompt string, defaultValue string) (string, error) {
	suffix := ""
	if defaultValue != "" {
		suffix = fmt.Sprintf(" [%s]", defaultValue)
	}
	fmt.Printf("%s%s: ", prompt, suffix)

	text, err := reader.ReadString('\n')
	if err != nil {
		// If EOF and we have a default value, use it
		if err == io.EOF && defaultValue != "" {
			fmt.Println() // Add newline
			return defaultValue, nil
		}
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	text = strings.TrimSpace(text)
	if text == "" {
		if defaultValue != "" {
			return defaultValue, nil
		}
		return "", fmt.Errorf("URL is required")
	}

	// Validate URL format and enforce http/https scheme
	parsedURL, err := url.Parse(text)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %s", text)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("invalid URL scheme (must be http or https): %s", text)
	}
	if parsedURL.Host == "" {
		return "", fmt.Errorf("invalid URL (missing host): %s", text)
	}

	return text, nil
}

func generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Ensure slug is not empty; fall back to a safe default
	if slug == "" {
		slug = "modpack"
	}

	// Ensure slug starts with a letter to avoid numeric-only slugs
	if len(slug) > 0 && (slug[0] < 'a' || slug[0] > 'z') {
		slug = "modpack-" + slug
	}

	return slug
}

func downloadAndCalculateChecksum(downloadURL string) (string, int64, error) {
	// Validate URL scheme (only allow HTTPS and HTTP)
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return "", 0, fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return "", 0, fmt.Errorf("unsupported URL scheme: %s (only http and https are allowed)", parsedURL.Scheme)
	}

	// Create HTTP client with reasonable timeout
	client := &http.Client{
		Timeout: 3 * time.Minute, // Reasonable timeout for most modpack downloads
	}

	// Download the file
	resp, err := client.Get(downloadURL)
	if err != nil {
		return "", 0, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Check content length to prevent memory exhaustion
	// Limit to 2GB to be safe (most modpacks are much smaller)
	const maxSize = 2 * 1024 * 1024 * 1024 // 2GB
	
	// Handle missing or negative Content-Length
	totalSize := resp.ContentLength
	if totalSize < 0 {
		totalSize = 0 // Treat as unknown size
	}
	
	if totalSize > maxSize {
		return "", 0, fmt.Errorf("file too large: %d bytes (max %d bytes)", totalSize, maxSize)
	}

	// Calculate checksum while downloading with size limit
	hash := sha256.New()

	// Create a limited reader to prevent reading more than maxSize
	limitedReader := io.LimitReader(resp.Body, maxSize)

	// Create progress bar if we know the size
	var written int64
	if totalSize > 0 {
		pb := ui.NewProgressBar(totalSize, "Downloading")
		_, err = io.Copy(io.MultiWriter(hash, &progressWriter{pb: pb, written: &written}), limitedReader)
		pb.Finish()
	} else {
		// No progress bar if size unknown, but still apply size limit
		written, err = io.Copy(hash, limitedReader)
	}

	if err != nil {
		return "", 0, fmt.Errorf("failed to read download: %w", err)
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	return checksum, written, nil
}

// progressWriter wraps a progress bar for io.Writer interface
type progressWriter struct {
	pb      *ui.ProgressBar
	written *int64
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	*pw.written += int64(n)
	pw.pb.Set(*pw.written)
	return n, nil
}

func loadTemplateRecipe(identifier string) (*search.Recipe, error) {
	// Try to load from a file path first
	if _, err := os.Stat(identifier); err == nil {
		data, err := os.ReadFile(identifier)
		if err != nil {
			return nil, fmt.Errorf("failed to read template file: %w", err)
		}

		var recipe search.Recipe
		if err := json.Unmarshal(data, &recipe); err != nil {
			return nil, fmt.Errorf("failed to parse template JSON: %w", err)
		}

		return &recipe, nil
	}

	// Try to load from installed benches
	manager, err := bench.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize bench manager: %w", err)
	}

	benches := manager.List()
	if len(benches) == 0 {
		return nil, fmt.Errorf("no benches installed")
	}

	// Search for recipe in benches
	for _, b := range benches {
		recipes, err := search.LoadRecipesFromBench(b.Path, b.Name)
		if err != nil {
			continue
		}

		for _, recipe := range recipes {
			if recipe.Slug == identifier || strings.EqualFold(recipe.Name, identifier) {
				return recipe, nil
			}
		}
	}

	return nil, fmt.Errorf("template recipe not found: %s", identifier)
}

func saveRecipe(recipe *search.Recipe, outputPath string) error {
	// Create output directory if needed
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Create file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Encode JSON with nice formatting
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(recipe); err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	return nil
}

func init() {
	// Add subcommands
	RecipeCmd.AddCommand(recipeCreateCmd)
	RecipeCmd.AddCommand(recipeValidateCmd)

	// Flags for create command
	recipeCreateCmd.Flags().StringVar(&templateRecipe, "template", "", "Start from an existing recipe (name, slug, or file path)")
	recipeCreateCmd.Flags().StringVar(&outputDir, "output", "", "Output directory for the recipe file (default: current directory)")

	// Suppress usage printing on errors
	RecipeCmd.SilenceUsage = true
	RecipeCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		cmd.Usage()
		return err
	})
}
