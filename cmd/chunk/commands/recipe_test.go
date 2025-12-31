package commands

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexinslc/chunk/internal/search"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "My Modpack",
			expected: "my-modpack",
		},
		{
			name:     "name with special characters",
			input:    "All The Mods 9!",
			expected: "all-the-mods-9",
		},
		{
			name:     "name with multiple spaces",
			input:    "Cool   Modpack   Name",
			expected: "cool-modpack-name",
		},
		{
			name:     "name with underscores",
			input:    "my_cool_modpack",
			expected: "my-cool-modpack",
		},
		{
			name:     "name with leading/trailing spaces",
			input:    "  Test Modpack  ",
			expected: "test-modpack",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "modpack",
		},
		{
			name:     "only special characters",
			input:    "!!!",
			expected: "modpack",
		},
		{
			name:     "only numbers",
			input:    "123",
			expected: "modpack-123",
		},
		{
			name:     "starts with number",
			input:    "9 Tech Mods",
			expected: "modpack-9-tech-mods",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.input)
			if result != tt.expected {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSaveRecipe(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	recipe := &search.Recipe{
		Name:             "Test Modpack",
		Slug:             "test-modpack",
		Description:      "A test modpack",
		MCVersion:        "1.20.1",
		Loader:           "forge",
		LoaderVersion:    "47.3.0",
		DownloadURL:      "https://example.com/modpack.zip",
		SHA256:           "abc123def456",
		RecommendedRAMGB: 6,
		DiskSpaceGB:      8,
		License:          "MIT",
	}

	outputPath := filepath.Join(tmpDir, "test-modpack.json")
	err := saveRecipe(recipe, outputPath)
	if err != nil {
		t.Fatalf("saveRecipe() failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("recipe file was not created: %s", outputPath)
	}

	// Verify content can be read back
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read recipe file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("recipe file is empty")
	}
}

func TestSaveRecipeCreatesDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	recipe := &search.Recipe{
		Name:        "Test Modpack",
		Slug:        "test-modpack",
		Description: "A test modpack",
		MCVersion:   "1.20.1",
		Loader:      "forge",
	}

	// Use a subdirectory that doesn't exist
	outputPath := filepath.Join(tmpDir, "subdir", "test-modpack.json")
	err := saveRecipe(recipe, outputPath)
	if err != nil {
		t.Fatalf("saveRecipe() failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("recipe file was not created: %s", outputPath)
	}
}

func TestRecipeCommandExists(t *testing.T) {
	if RecipeCmd == nil {
		t.Fatal("RecipeCmd is nil")
	}

	if RecipeCmd.Use != "recipe" {
		t.Errorf("RecipeCmd.Use = %q, want %q", RecipeCmd.Use, "recipe")
	}

	// Check that create subcommand exists
	createCmd := RecipeCmd.Commands()
	if len(createCmd) == 0 {
		t.Fatal("RecipeCmd has no subcommands")
	}

	found := false
	for _, cmd := range createCmd {
		if cmd.Use == "create" {
			found = true
			break
		}
	}

	if !found {
		t.Error("create subcommand not found")
	}
}

func TestPromptString(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue string
		required     bool
		expected     string
		expectError  bool
	}{
		{
			name:         "valid input",
			input:        "test value\n",
			defaultValue: "",
			required:     true,
			expected:     "test value",
			expectError:  false,
		},
		{
			name:         "empty with default",
			input:        "\n",
			defaultValue: "default",
			required:     false,
			expected:     "default",
			expectError:  false,
		},
		{
			name:         "empty without default not required",
			input:        "\n",
			defaultValue: "",
			required:     false,
			expected:     "",
			expectError:  false,
		},
		{
			name:         "empty without default required",
			input:        "\n",
			defaultValue: "",
			required:     true,
			expected:     "",
			expectError:  true,
		},
		{
			name:         "whitespace trimmed",
			input:        "  spaces  \n",
			defaultValue: "",
			required:     true,
			expected:     "spaces",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			result, err := promptString(reader, "test", tt.defaultValue, tt.required)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectError && result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPromptInt(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue int
		required     bool
		expected     int
		expectError  bool
	}{
		{
			name:         "valid number",
			input:        "42\n",
			defaultValue: 0,
			required:     true,
			expected:     42,
			expectError:  false,
		},
		{
			name:         "empty with default",
			input:        "\n",
			defaultValue: 10,
			required:     false,
			expected:     10,
			expectError:  false,
		},
		{
			name:         "invalid number",
			input:        "abc\n",
			defaultValue: 0,
			required:     true,
			expected:     0,
			expectError:  true,
		},
		{
			name:         "zero value",
			input:        "0\n",
			defaultValue: 5,
			required:     false,
			expected:     0,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			result, err := promptInt(reader, "test", tt.defaultValue, tt.required)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectError && result != tt.expected {
				t.Errorf("got %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestPromptLoader(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue string
		expected     string
		expectError  bool
	}{
		{
			name:         "valid forge",
			input:        "forge\n",
			defaultValue: "",
			expected:     "forge",
			expectError:  false,
		},
		{
			name:         "valid fabric",
			input:        "fabric\n",
			defaultValue: "",
			expected:     "fabric",
			expectError:  false,
		},
		{
			name:         "valid neoforge",
			input:        "neoforge\n",
			defaultValue: "",
			expected:     "neoforge",
			expectError:  false,
		},
		{
			name:         "invalid loader",
			input:        "invalid\n",
			defaultValue: "",
			expected:     "",
			expectError:  true,
		},
		{
			name:         "empty with default",
			input:        "\n",
			defaultValue: "forge",
			expected:     "forge",
			expectError:  false,
		},
		{
			name:         "empty without default",
			input:        "\n",
			defaultValue: "",
			expected:     "",
			expectError:  true,
		},
		{
			name:         "case insensitive",
			input:        "FORGE\n",
			defaultValue: "",
			expected:     "forge",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			result, err := promptLoader(reader, tt.defaultValue)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectError && result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPromptLicense(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue string
		expected     string
	}{
		{
			name:         "custom license",
			input:        "MIT\n",
			defaultValue: "",
			expected:     "MIT",
		},
		{
			name:         "empty defaults to ARR",
			input:        "\n",
			defaultValue: "",
			expected:     "ARR",
		},
		{
			name:         "empty with template default",
			input:        "\n",
			defaultValue: "GPL-3.0",
			expected:     "GPL-3.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			result, err := promptLicense(reader, tt.defaultValue)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPromptURL(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue string
		expected     string
		expectError  bool
	}{
		{
			name:         "valid https URL",
			input:        "https://example.com/file.zip\n",
			defaultValue: "",
			expected:     "https://example.com/file.zip",
			expectError:  false,
		},
		{
			name:         "valid http URL",
			input:        "http://example.com/file.zip\n",
			defaultValue: "",
			expected:     "http://example.com/file.zip",
			expectError:  false,
		},
		{
			name:         "invalid scheme ftp",
			input:        "ftp://example.com/file.zip\n",
			defaultValue: "",
			expected:     "",
			expectError:  true,
		},
		{
			name:         "invalid scheme file",
			input:        "file:///tmp/file.zip\n",
			defaultValue: "",
			expected:     "",
			expectError:  true,
		},
		{
			name:         "missing scheme",
			input:        "example.com/file.zip\n",
			defaultValue: "",
			expected:     "",
			expectError:  true,
		},
		{
			name:         "missing host",
			input:        "https:///file.zip\n",
			defaultValue: "",
			expected:     "",
			expectError:  true,
		},
		{
			name:         "empty with default",
			input:        "\n",
			defaultValue: "https://example.com/default.zip",
			expected:     "https://example.com/default.zip",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			result, err := promptURL(reader, "test", tt.defaultValue)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectError && result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDownloadAndCalculateChecksum(t *testing.T) {
	t.Run("valid download with known size", func(t *testing.T) {
		// Create test server
		testData := []byte("test data for checksum")
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testData)))
			w.WriteHeader(http.StatusOK)
			w.Write(testData)
		}))
		defer server.Close()

		checksum, size, err := downloadAndCalculateChecksum(server.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if size != int64(len(testData)) {
			t.Errorf("size = %d, want %d", size, len(testData))
		}

		if checksum == "" {
			t.Error("checksum is empty")
		}

		// Verify checksum length (SHA-256 produces 64 hex chars)
		if len(checksum) != 64 {
			t.Errorf("checksum length = %d, want 64", len(checksum))
		}
	})

	t.Run("valid download with unknown size", func(t *testing.T) {
		testData := []byte("test data")
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Don't set Content-Length
			w.WriteHeader(http.StatusOK)
			w.Write(testData)
		}))
		defer server.Close()

		checksum, size, err := downloadAndCalculateChecksum(server.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if size != int64(len(testData)) {
			t.Errorf("size = %d, want %d", size, len(testData))
		}

		if checksum == "" {
			t.Error("checksum is empty")
		}
	})

	t.Run("rejects invalid URL scheme", func(t *testing.T) {
		_, _, err := downloadAndCalculateChecksum("ftp://example.com/file.zip")
		if err == nil {
			t.Error("expected error for invalid scheme")
		}
	})

	t.Run("rejects file scheme", func(t *testing.T) {
		_, _, err := downloadAndCalculateChecksum("file:///tmp/file.zip")
		if err == nil {
			t.Error("expected error for file scheme")
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		_, _, err := downloadAndCalculateChecksum(server.URL)
		if err == nil {
			t.Error("expected error for 404 response")
		}
	})

	t.Run("rejects file exceeding size limit", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set Content-Length to exceed 2GB limit
			w.Header().Set("Content-Length", "3000000000")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		_, _, err := downloadAndCalculateChecksum(server.URL)
		if err == nil {
			t.Error("expected error for file exceeding size limit")
		}
	})

	t.Run("handles negative Content-Length", func(t *testing.T) {
		testData := []byte("test data")
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "-1")
			w.WriteHeader(http.StatusOK)
			w.Write(testData)
		}))
		defer server.Close()

		checksum, size, err := downloadAndCalculateChecksum(server.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if size != int64(len(testData)) {
			t.Errorf("size = %d, want %d", size, len(testData))
		}

		if checksum == "" {
			t.Error("checksum is empty")
		}
	})
}

func TestLoadTemplateRecipe(t *testing.T) {
	t.Run("loads from valid file path", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create a test recipe file
		recipe := &search.Recipe{
			Name:        "Test Template",
			Slug:        "test-template",
			MCVersion:   "1.20.1",
			Loader:      "forge",
			Description: "A test template",
		}
		
		filePath := filepath.Join(tmpDir, "template.json")
		err := saveRecipe(recipe, filePath)
		if err != nil {
			t.Fatalf("failed to create test recipe: %v", err)
		}

		loaded, err := loadTemplateRecipe(filePath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if loaded.Name != recipe.Name {
			t.Errorf("name = %q, want %q", loaded.Name, recipe.Name)
		}
		if loaded.Slug != recipe.Slug {
			t.Errorf("slug = %q, want %q", loaded.Slug, recipe.Slug)
		}
	})

	t.Run("returns error for non-existent file without benches", func(t *testing.T) {
		_, err := loadTemplateRecipe("non-existent-recipe")
		if err == nil {
			t.Error("expected error for non-existent recipe")
		}
	})

	t.Run("returns error for invalid JSON file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "invalid.json")
		
		// Write invalid JSON
		err := os.WriteFile(filePath, []byte("invalid json {"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		_, err = loadTemplateRecipe(filePath)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}
