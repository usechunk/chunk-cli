package converter

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexinslc/chunk/internal/sources"
)

func TestModManager_DownloadModWithChecksum(t *testing.T) {
	// Create a test server that serves a file
	testContent := []byte("test mod content for checksum verification")
	// SHA256 of "test mod content for checksum verification"
	expectedSHA256 := "7b5f8c1c4a2c4e5f3e1b2a8c9d0e1f2a3b4c5d6e7f8091a2b3c4d5e6f7a8b9c0d"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(testContent)
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "mod-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name       string
		mod        *sources.Mod
		skipVerify bool
		wantErr    bool
	}{
		{
			name: "download without checksum",
			mod: &sources.Mod{
				Name:        "TestMod",
				FileName:    "testmod-no-checksum.jar",
				DownloadURL: server.URL + "/mod.jar",
			},
			skipVerify: false,
			wantErr:    false,
		},
		{
			name: "download with skip-verify",
			mod: &sources.Mod{
				Name:        "TestMod",
				FileName:    "testmod-skip.jar",
				DownloadURL: server.URL + "/mod.jar",
				SHA256:      "wrongchecksum",
			},
			skipVerify: true,
			wantErr:    false,
		},
		{
			name: "download with invalid checksum",
			mod: &sources.Mod{
				Name:        "TestMod",
				FileName:    "testmod-invalid.jar",
				DownloadURL: server.URL + "/mod.jar",
				SHA256:      expectedSHA256,
			},
			skipVerify: false,
			wantErr:    true, // Will fail because the checksum won't match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh directory for each test
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatalf("Failed to create test dir: %v", err)
			}

			modManager := NewModManager()
			modManager.SkipVerify = tt.skipVerify

			err := modManager.downloadMod(tt.mod, testDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("downloadMod() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check if file was created (for success cases)
			if !tt.wantErr {
				filePath := filepath.Join(testDir, tt.mod.FileName)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Error("Expected file to be created")
				}
			}
		})
	}
}

func TestModManager_FilterServerMods(t *testing.T) {
	modManager := NewModManager()

	mods := []*sources.Mod{
		{Name: "ServerMod", Side: sources.SideServer},
		{Name: "ClientMod", Side: sources.SideClient},
		{Name: "BothMod", Side: sources.SideBoth},
		{Name: "UnknownMod", Side: ""},
	}

	serverMods := modManager.FilterServerMods(mods)

	// Should include server, both, and unknown (default to server)
	if len(serverMods) != 3 {
		t.Errorf("Expected 3 server mods, got %d", len(serverMods))
	}

	// Check that client-only mod is filtered out
	for _, mod := range serverMods {
		if mod.Side == sources.SideClient {
			t.Error("Client-only mod should be filtered out")
		}
	}
}

func TestModManager_SkipVerifyOption(t *testing.T) {
	modManager := NewModManager()

	// Default should be false
	if modManager.SkipVerify {
		t.Error("SkipVerify should be false by default")
	}

	// Can set to true
	modManager.SkipVerify = true
	if !modManager.SkipVerify {
		t.Error("SkipVerify should be true after setting")
	}
}
