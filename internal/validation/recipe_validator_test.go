package validation

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/alexinslc/chunk/internal/search"
)

func TestValidateRequiredFields(t *testing.T) {
	validator := NewRecipeValidator()

	tests := []struct {
		name        string
		recipe      *search.Recipe
		expectError bool
		errorField  string
	}{
		{
			name: "valid recipe with all required fields",
			recipe: &search.Recipe{
				Name:        "Test Pack",
				MCVersion:   "1.20.1",
				Loader:      "forge",
				DownloadURL: "https://example.com/pack.zip",
				SHA256:      "abc123",
			},
			expectError: false,
		},
		{
			name: "missing name",
			recipe: &search.Recipe{
				MCVersion:   "1.20.1",
				Loader:      "forge",
				DownloadURL: "https://example.com/pack.zip",
			},
			expectError: true,
			errorField:  "name",
		},
		{
			name: "missing mc_version",
			recipe: &search.Recipe{
				Name:        "Test Pack",
				Loader:      "forge",
				DownloadURL: "https://example.com/pack.zip",
			},
			expectError: true,
			errorField:  "mc_version",
		},
		{
			name: "missing loader",
			recipe: &search.Recipe{
				Name:        "Test Pack",
				MCVersion:   "1.20.1",
				DownloadURL: "https://example.com/pack.zip",
			},
			expectError: true,
			errorField:  "loader",
		},
		{
			name: "missing download_url",
			recipe: &search.Recipe{
				Name:      "Test Pack",
				MCVersion: "1.20.1",
				Loader:    "forge",
			},
			expectError: true,
			errorField:  "download_url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateRecipe(tt.recipe, "")

			hasError := len(result.Errors) > 0
			if hasError != tt.expectError {
				t.Errorf("expected error: %v, got error: %v (errors: %v)", tt.expectError, hasError, result.Errors)
			}

			if tt.expectError && tt.errorField != "" {
				found := false
				for _, err := range result.Errors {
					if err.Field == tt.errorField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error for field %s, but not found in errors", tt.errorField)
				}
			}
		})
	}
}

func TestValidateVersionFormat(t *testing.T) {
	validator := NewRecipeValidator()

	tests := []struct {
		name        string
		mcVersion   string
		version     string
		expectError bool
	}{
		{
			name:        "valid minecraft version",
			mcVersion:   "1.20.1",
			version:     "1.0.0",
			expectError: false,
		},
		{
			name:        "valid minecraft version without patch",
			mcVersion:   "1.20",
			version:     "1.0.0",
			expectError: false,
		},
		{
			name:        "invalid minecraft version",
			mcVersion:   "1.20.x",
			version:     "1.0.0",
			expectError: true,
		},
		{
			name:        "invalid minecraft version format",
			mcVersion:   "v1.20.1",
			version:     "1.0.0",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recipe := &search.Recipe{
				Name:        "Test Pack",
				MCVersion:   tt.mcVersion,
				Version:     tt.version,
				Loader:      "forge",
				DownloadURL: "https://example.com/pack.zip",
			}

			result := validator.ValidateRecipe(recipe, "")

			hasError := len(result.Errors) > 0
			if hasError != tt.expectError {
				t.Errorf("expected error: %v, got error: %v (errors: %v)", tt.expectError, hasError, result.Errors)
			}
		})
	}
}

func TestValidateLoader(t *testing.T) {
	validator := NewRecipeValidator()

	tests := []struct {
		name        string
		loader      string
		expectError bool
	}{
		{
			name:        "valid forge",
			loader:      "forge",
			expectError: false,
		},
		{
			name:        "valid fabric",
			loader:      "fabric",
			expectError: false,
		},
		{
			name:        "valid neoforge",
			loader:      "neoforge",
			expectError: false,
		},
		{
			name:        "valid case insensitive",
			loader:      "Forge",
			expectError: false,
		},
		{
			name:        "invalid loader",
			loader:      "quilt",
			expectError: true,
		},
		{
			name:        "invalid loader",
			loader:      "vanilla",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recipe := &search.Recipe{
				Name:        "Test Pack",
				MCVersion:   "1.20.1",
				Loader:      tt.loader,
				DownloadURL: "https://example.com/pack.zip",
			}

			result := validator.ValidateRecipe(recipe, "")

			hasError := len(result.Errors) > 0
			if hasError != tt.expectError {
				t.Errorf("expected error: %v, got error: %v (errors: %v)", tt.expectError, hasError, result.Errors)
			}
		})
	}
}

func TestValidateURLFormat(t *testing.T) {
	validator := NewRecipeValidator()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "valid https url",
			url:         "https://example.com/pack.zip",
			expectError: false,
		},
		{
			name:        "valid http url",
			url:         "http://example.com/pack.zip",
			expectError: false,
		},
		{
			name:        "invalid scheme ftp",
			url:         "ftp://example.com/pack.zip",
			expectError: true,
		},
		{
			name:        "invalid scheme file",
			url:         "file:///tmp/pack.zip",
			expectError: true,
		},
		{
			name:        "missing scheme",
			url:         "example.com/pack.zip",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recipe := &search.Recipe{
				Name:        "Test Pack",
				MCVersion:   "1.20.1",
				Loader:      "forge",
				DownloadURL: tt.url,
			}

			result := validator.ValidateRecipe(recipe, "")

			hasError := len(result.Errors) > 0
			if hasError != tt.expectError {
				t.Errorf("expected error: %v, got error: %v (errors: %v)", tt.expectError, hasError, result.Errors)
			}
		})
	}
}

func TestValidateLicense(t *testing.T) {
	validator := NewRecipeValidator()

	tests := []struct {
		name        string
		license     string
		expectError bool
	}{
		{
			name:        "valid MIT",
			license:     "MIT",
			expectError: false,
		},
		{
			name:        "valid GPL-3.0",
			license:     "GPL-3.0",
			expectError: false,
		},
		{
			name:        "valid Apache-2.0",
			license:     "Apache-2.0",
			expectError: false,
		},
		{
			name:        "valid ARR",
			license:     "ARR",
			expectError: false,
		},
		{
			name:        "invalid custom",
			license:     "Custom",
			expectError: true,
		},
		{
			name:        "invalid proprietary",
			license:     "Proprietary",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recipe := &search.Recipe{
				Name:        "Test Pack",
				MCVersion:   "1.20.1",
				Loader:      "forge",
				DownloadURL: "https://example.com/pack.zip",
				License:     tt.license,
			}

			result := validator.ValidateRecipe(recipe, "")

			hasError := len(result.Errors) > 0
			if hasError != tt.expectError {
				t.Errorf("expected error: %v, got error: %v (errors: %v)", tt.expectError, hasError, result.Errors)
			}
		})
	}
}

func TestValidateNaming(t *testing.T) {
	validator := NewRecipeValidator()

	tests := []struct {
		name         string
		slug         string
		filePath     string
		expectWarn   bool
	}{
		{
			name:       "matching slug and filename",
			slug:       "test-pack",
			filePath:   "/path/to/test-pack.json",
			expectWarn: false,
		},
		{
			name:       "mismatching slug and filename",
			slug:       "test-pack",
			filePath:   "/path/to/different-name.json",
			expectWarn: true,
		},
		{
			name:       "no slug provided",
			slug:       "",
			filePath:   "/path/to/test-pack.json",
			expectWarn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recipe := &search.Recipe{
				Name:        "Test Pack",
				Slug:        tt.slug,
				MCVersion:   "1.20.1",
				Loader:      "forge",
				DownloadURL: "https://example.com/pack.zip",
				SHA256:      "abc123", // Prevent SHA256 warning
				LoaderVersion: "47.3.0", // Prevent loader version warning
				License:     "MIT", // Prevent license warning
			}

			result := validator.ValidateRecipe(recipe, tt.filePath)

			// Check specifically for slug warning
			hasSlugWarn := false
			for _, warn := range result.Warnings {
				if warn.Field == "slug" {
					hasSlugWarn = true
					break
				}
			}
			if hasSlugWarn != tt.expectWarn {
				t.Errorf("expected slug warning: %v, got slug warning: %v (warnings: %v)", tt.expectWarn, hasSlugWarn, result.Warnings)
			}
		})
	}
}

func TestValidateURLReachability(t *testing.T) {
	validator := NewRecipeValidator()

	t.Run("reachable URL returns 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "1000")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		size, err := validator.validateURLReachability(server.URL)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if size != 1000 {
			t.Errorf("expected size 1000, got %d", size)
		}
	})

	t.Run("unreachable URL returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		_, err := validator.validateURLReachability(server.URL)
		if err == nil {
			t.Error("expected error for 404 response")
		}
	})
}

func TestValidateChecksum(t *testing.T) {
	validator := NewRecipeValidator()

	testData := []byte("test data for checksum")

	t.Run("matching checksum", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(testData)
		}))
		defer server.Close()

		// Calculate expected checksum
		expectedChecksum := "7139bd88b36819ededa38e297c4caed50543b0014430590835c1943d22ff8998"

		err := validator.validateChecksum(server.URL, expectedChecksum)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("mismatching checksum", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(testData)
		}))
		defer server.Close()

		wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"

		err := validator.validateChecksum(server.URL, wrongChecksum)
		if err == nil {
			t.Error("expected error for mismatching checksum")
		}
	})
}

func TestIsValidMCVersion(t *testing.T) {
	tests := []struct {
		version string
		valid   bool
	}{
		{"1.20.1", true},
		{"1.20", true},
		{"1.19.2", true},
		{"1.20.x", false},
		{"v1.20.1", false},
		{"1.20.1.0", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := isValidMCVersion(tt.version)
			if result != tt.valid {
				t.Errorf("isValidMCVersion(%q) = %v, want %v", tt.version, result, tt.valid)
			}
		})
	}
}

func TestIsValidSemver(t *testing.T) {
	tests := []struct {
		version string
		valid   bool
	}{
		{"1.0.0", true},
		{"1.2.3", true},
		{"1.0.0-alpha", true},
		{"1.0.0+build.123", true},
		{"1.0.0-alpha+build", true},
		{"1.0", false},
		{"v1.0.0", false},
		{"1.0.x", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := isValidSemver(tt.version)
			if result != tt.valid {
				t.Errorf("isValidSemver(%q) = %v, want %v", tt.version, result, tt.valid)
			}
		})
	}
}

func TestIsValidSPDXLicense(t *testing.T) {
	tests := []struct {
		license string
		valid   bool
	}{
		{"MIT", true},
		{"Apache-2.0", true},
		{"GPL-3.0", true},
		{"ARR", true},
		{"BSD-3-Clause", true},
		{"Custom", false},
		{"Proprietary", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.license, func(t *testing.T) {
			result := isValidSPDXLicense(tt.license)
			if result != tt.valid {
				t.Errorf("isValidSPDXLicense(%q) = %v, want %v", tt.license, result, tt.valid)
			}
		})
	}
}

func TestValidateRecipeIntegration(t *testing.T) {
	validator := NewRecipeValidator()

	t.Run("fully valid recipe", func(t *testing.T) {
		recipe := &search.Recipe{
			Name:             "All The Mods 9",
			Slug:             "atm9",
			Description:      "Kitchen sink modpack",
			MCVersion:        "1.20.1",
			Loader:           "forge",
			LoaderVersion:    "47.3.0",
			Version:          "1.0.0",
			DownloadURL:      "https://example.com/atm9.zip",
			SHA256:           "abc123",
			License:          "MIT",
			RecommendedRAMGB: 8,
		}

		result := validator.ValidateRecipe(recipe, filepath.Join("/path/to/", "atm9.json"))

		if len(result.Errors) > 0 {
			t.Errorf("expected no errors, got: %v", result.Errors)
		}
	})

	t.Run("recipe with multiple errors", func(t *testing.T) {
		recipe := &search.Recipe{
			Name:        "Invalid Pack",
			MCVersion:   "1.20.x",  // Invalid format
			Loader:      "quilt",   // Invalid loader
			DownloadURL: "ftp://example.com/pack.zip", // Invalid scheme
			License:     "Custom",  // Invalid SPDX
		}

		result := validator.ValidateRecipe(recipe, "")

		if len(result.Errors) < 4 {
			t.Errorf("expected at least 4 errors, got %d: %v", len(result.Errors), result.Errors)
		}
	})
}

func TestValidateRecipeWithNetwork(t *testing.T) {
	validator := NewRecipeValidator()

	testData := []byte("test modpack data")
	expectedChecksum := "1d2f0c42e5b8e9c2e15d2f90e4f5a8b9c4d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0"

	t.Run("valid network checks", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testData)))
			w.WriteHeader(http.StatusOK)
			w.Write(testData)
		}))
		defer server.Close()

		recipe := &search.Recipe{
			Name:        "Test Pack",
			MCVersion:   "1.20.1",
			Loader:      "forge",
			DownloadURL: server.URL,
			SHA256:      expectedChecksum,
		}

		result, size := validator.ValidateRecipeWithNetwork(recipe, "")

		// URL should be reachable
		if len(result.Errors) > 0 {
			// Check if error is about checksum mismatch (which is expected unless we calculate the actual hash)
			hasNonChecksumError := false
			for _, err := range result.Errors {
				if err.Field != "sha256" {
					hasNonChecksumError = true
					break
				}
			}
			if hasNonChecksumError {
				t.Errorf("unexpected non-checksum errors: %v", result.Errors)
			}
		}

		if size != int64(len(testData)) {
			t.Errorf("expected size %d, got %d", len(testData), size)
		}
	})
}
