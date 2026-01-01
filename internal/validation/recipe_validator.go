package validation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/alexinslc/chunk/internal/search"
)

// RecipeValidator validates recipe JSON files
type RecipeValidator struct {
	httpClient *http.Client
}

// NewRecipeValidator creates a new recipe validator
func NewRecipeValidator() *RecipeValidator {
	return &RecipeValidator{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ValidationResult represents the result of validating a recipe
type ValidationResult struct {
	Errors   []RecipeValidationError
	Warnings []RecipeValidationWarning
}

// RecipeValidationError represents a validation error
type RecipeValidationError struct {
	Field      string
	Message    string
	Suggestion string
}

// RecipeValidationWarning represents a validation warning
type RecipeValidationWarning struct {
	Field   string
	Message string
}

// ValidateRecipe validates a recipe file
func (v *RecipeValidator) ValidateRecipe(recipe *search.Recipe, filePath string) *ValidationResult {
	result := &ValidationResult{
		Errors:   []RecipeValidationError{},
		Warnings: []RecipeValidationWarning{},
	}

	// Validate required fields
	v.validateRequiredFields(recipe, result)

	// Validate version format
	v.validateVersionFormat(recipe, result)

	// Validate loader
	v.validateLoader(recipe, result)

	// Validate URL format
	v.validateURLFormat(recipe, result)

	// Validate license
	v.validateLicense(recipe, result)

	// Validate naming (slug matches filename)
	if filePath != "" {
		v.validateNaming(recipe, filePath, result)
	}

	return result
}

// ValidateRecipeWithNetwork validates a recipe with network checks (URL reachability and checksum)
func (v *RecipeValidator) ValidateRecipeWithNetwork(recipe *search.Recipe, filePath string) (*ValidationResult, int64) {
	result := v.ValidateRecipe(recipe, filePath)

	var downloadSize int64

	// Validate URL reachability
	if recipe.DownloadURL != "" {
		size, err := v.validateURLReachability(recipe.DownloadURL)
		if err != nil {
			result.Errors = append(result.Errors, RecipeValidationError{
				Field:      "download_url",
				Message:    fmt.Sprintf("URL not reachable: %v", err),
				Suggestion: "Ensure the URL is accessible and returns HTTP 200",
			})
		} else {
			downloadSize = size
		}

		// Validate checksum if URL is reachable
		if recipe.SHA256 != "" && err == nil {
			if checksumErr := v.validateChecksum(recipe.DownloadURL, recipe.SHA256); checksumErr != nil {
				result.Errors = append(result.Errors, RecipeValidationError{
					Field:      "sha256",
					Message:    fmt.Sprintf("Checksum mismatch: %v", checksumErr),
					Suggestion: "Download the file and recalculate the SHA-256 checksum",
				})
			}
		}
	}

	return result, downloadSize
}

func (v *RecipeValidator) validateRequiredFields(recipe *search.Recipe, result *ValidationResult) {
	if recipe.Name == "" {
		result.Errors = append(result.Errors, RecipeValidationError{
			Field:      "name",
			Message:    "Name is required",
			Suggestion: "Add a descriptive name for the recipe",
		})
	}

	if recipe.MCVersion == "" {
		result.Errors = append(result.Errors, RecipeValidationError{
			Field:      "mc_version",
			Message:    "Minecraft version is required",
			Suggestion: "Add mc_version (e.g., \"1.20.1\")",
		})
	}

	if recipe.Loader == "" {
		result.Errors = append(result.Errors, RecipeValidationError{
			Field:      "loader",
			Message:    "Loader is required",
			Suggestion: "Add loader (forge, fabric, or neoforge)",
		})
	}

	if recipe.DownloadURL == "" {
		result.Errors = append(result.Errors, RecipeValidationError{
			Field:      "download_url",
			Message:    "Download URL is required",
			Suggestion: "Add download_url pointing to the modpack archive",
		})
	}

	if recipe.SHA256 == "" {
		result.Warnings = append(result.Warnings, RecipeValidationWarning{
			Field:   "sha256",
			Message: "SHA-256 checksum is recommended for integrity verification",
		})
	}
}

func (v *RecipeValidator) validateVersionFormat(recipe *search.Recipe, result *ValidationResult) {
	// Validate Minecraft version format
	if recipe.MCVersion != "" {
		if !isValidMCVersion(recipe.MCVersion) {
			result.Errors = append(result.Errors, RecipeValidationError{
				Field:      "mc_version",
				Message:    fmt.Sprintf("Invalid Minecraft version format: %s", recipe.MCVersion),
				Suggestion: "Use format: MAJOR.MINOR.PATCH (e.g., \"1.20.1\")",
			})
		}
	}

	// Validate recipe version if present (should be semver)
	if recipe.Version != "" {
		if !isValidSemver(recipe.Version) {
			result.Warnings = append(result.Warnings, RecipeValidationWarning{
				Field:   "version",
				Message: fmt.Sprintf("Version should follow semver format: %s", recipe.Version),
			})
		}
	}
}

func (v *RecipeValidator) validateLoader(recipe *search.Recipe, result *ValidationResult) {
	validLoaders := map[string]bool{
		"forge":    true,
		"fabric":   true,
		"neoforge": true,
	}

	loader := strings.ToLower(recipe.Loader)
	if loader != "" && !validLoaders[loader] {
		result.Errors = append(result.Errors, RecipeValidationError{
			Field:      "loader",
			Message:    fmt.Sprintf("Invalid loader: %s", recipe.Loader),
			Suggestion: "Use one of: forge, fabric, neoforge",
		})
	}

	// Check if loader version is provided
	if recipe.Loader != "" && recipe.LoaderVersion == "" {
		result.Warnings = append(result.Warnings, RecipeValidationWarning{
			Field:   "loader_version",
			Message: "Loader version is recommended",
		})
	}
}

func (v *RecipeValidator) validateURLFormat(recipe *search.Recipe, result *ValidationResult) {
	if recipe.DownloadURL == "" {
		return
	}

	parsedURL, err := url.Parse(recipe.DownloadURL)
	if err != nil {
		result.Errors = append(result.Errors, RecipeValidationError{
			Field:      "download_url",
			Message:    fmt.Sprintf("Invalid URL format: %v", err),
			Suggestion: "Ensure the URL is properly formatted",
		})
		return
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		result.Errors = append(result.Errors, RecipeValidationError{
			Field:      "download_url",
			Message:    fmt.Sprintf("Invalid URL scheme: %s", parsedURL.Scheme),
			Suggestion: "Use http or https scheme",
		})
	}

	if parsedURL.Host == "" {
		result.Errors = append(result.Errors, RecipeValidationError{
			Field:      "download_url",
			Message:    "URL missing host",
			Suggestion: "Ensure the URL includes a valid hostname",
		})
	}
}

func (v *RecipeValidator) validateLicense(recipe *search.Recipe, result *ValidationResult) {
	if recipe.License == "" {
		result.Warnings = append(result.Warnings, RecipeValidationWarning{
			Field:   "license",
			Message: "License is recommended",
		})
		return
	}

	if !isValidSPDXLicense(recipe.License) {
		result.Errors = append(result.Errors, RecipeValidationError{
			Field:      "license",
			Message:    fmt.Sprintf("'%s' is not a valid SPDX identifier", recipe.License),
			Suggestion: "Use a valid SPDX license identifier (e.g., MIT, GPL-3.0, Apache-2.0) or ARR (All Rights Reserved)",
		})
	}
}

func (v *RecipeValidator) validateNaming(recipe *search.Recipe, filePath string, result *ValidationResult) {
	fileName := filepath.Base(filePath)
	fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	if recipe.Slug != "" && recipe.Slug != fileNameWithoutExt {
		result.Warnings = append(result.Warnings, RecipeValidationWarning{
			Field:   "slug",
			Message: fmt.Sprintf("Slug '%s' does not match filename '%s'", recipe.Slug, fileNameWithoutExt),
		})
	}
}

func (v *RecipeValidator) validateURLReachability(downloadURL string) (int64, error) {
	resp, err := v.httpClient.Head(downloadURL)
	if err != nil {
		// Try GET if HEAD fails
		resp, err = v.httpClient.Get(downloadURL)
		if err != nil {
			return 0, fmt.Errorf("failed to reach URL: %w", err)
		}
		defer resp.Body.Close()
	} else {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return resp.ContentLength, nil
}

func (v *RecipeValidator) validateChecksum(downloadURL, expectedSHA256 string) error {
	resp, err := v.httpClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Calculate checksum
	hash := sha256.New()
	// Limit to 2GB to prevent memory issues
	limitedReader := io.LimitReader(resp.Body, 2*1024*1024*1024)
	_, err = io.Copy(hash, limitedReader)
	if err != nil {
		return fmt.Errorf("failed to read download: %w", err)
	}

	actualSHA256 := hex.EncodeToString(hash.Sum(nil))
	if actualSHA256 != expectedSHA256 {
		return fmt.Errorf("expected %s, got %s", expectedSHA256, actualSHA256)
	}

	return nil
}

// isValidMCVersion checks if a Minecraft version is valid
func isValidMCVersion(version string) bool {
	// Match versions like 1.20.1, 1.19.2, etc.
	matched, _ := regexp.MatchString(`^\d+\.\d+(\.\d+)?$`, version)
	return matched
}

// isValidSemver checks if a version follows semantic versioning
func isValidSemver(version string) bool {
	// Basic semver check: MAJOR.MINOR.PATCH
	matched, _ := regexp.MatchString(`^\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$`, version)
	return matched
}

// isValidSPDXLicense checks if a license is a valid SPDX identifier
// This is a subset of common licenses; for full validation, use an SPDX library
func isValidSPDXLicense(license string) bool {
	// Common SPDX licenses
	validLicenses := map[string]bool{
		"ARR":         true, // All Rights Reserved
		"MIT":         true,
		"Apache-2.0":  true,
		"GPL-2.0":     true,
		"GPL-3.0":     true,
		"LGPL-2.1":    true,
		"LGPL-3.0":    true,
		"BSD-2-Clause": true,
		"BSD-3-Clause": true,
		"ISC":         true,
		"MPL-2.0":     true,
		"CC0-1.0":     true,
		"Unlicense":   true,
		"WTFPL":       true,
	}

	return validLicenses[license]
}
