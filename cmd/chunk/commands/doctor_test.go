package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestDoctorCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "doctor without args",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "doctor with verbose flag",
			args:    []string{"--verbose"},
			wantErr: false,
		},
		{
			name:    "doctor with short verbose flag",
			args:    []string{"-v"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{Use: "chunk"}
			rootCmd.AddCommand(DoctorCmd)

			_, err := executeCommand(rootCmd, append([]string{"doctor"}, tt.args...)...)
			if (err != nil) != tt.wantErr {
				t.Errorf("DoctorCmd error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDoctorCommandFlags(t *testing.T) {
	// Test that the doctor command has the expected flags
	rootCmd := &cobra.Command{Use: "chunk"}
	rootCmd.AddCommand(DoctorCmd)

	// Check that --verbose flag exists
	verboseFlag := DoctorCmd.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Error("Expected --verbose flag to exist")
	}
	if verboseFlag.Shorthand != "v" {
		t.Errorf("Expected --verbose shorthand to be 'v', got '%s'", verboseFlag.Shorthand)
	}
	if verboseFlag.DefValue != "false" {
		t.Errorf("Expected --verbose default to be 'false', got '%s'", verboseFlag.DefValue)
	}
}

func TestCheckGit(t *testing.T) {
	result := checkGit()
	
	// We can't guarantee git is installed in all test environments,
	// but we can verify the function returns a valid result
	if result.message == "" {
		t.Error("Expected checkGit to return a non-empty message")
	}
	
	// If git is not installed, expect a fix message
	if !result.success && result.fix == "" {
		t.Error("Expected checkGit to provide a fix when unsuccessful")
	}
}

func TestCheckJava(t *testing.T) {
	results := checkJava()
	
	// Should return at least one result (either success or failure)
	if len(results) == 0 {
		t.Error("Expected checkJava to return at least one result")
	}
	
	// Verify all results have messages
	for i, result := range results {
		if result.message == "" {
			t.Errorf("Expected result %d to have a non-empty message", i)
		}
	}
}

func TestCheckDiskSpace(t *testing.T) {
	result := checkDiskSpace()
	
	// Disk space check should always return a result
	if result.message == "" {
		t.Error("Expected checkDiskSpace to return a non-empty message")
	}
}

func TestCheckBenches(t *testing.T) {
	results := checkBenches()
	
	// Should return at least one result
	if len(results) == 0 {
		t.Error("Expected checkBenches to return at least one result")
	}
	
	// Verify all results have messages
	for i, result := range results {
		if result.message == "" {
			t.Errorf("Expected result %d to have a non-empty message", i)
		}
	}
}

func TestCheckRecipes(t *testing.T) {
	results := checkRecipes()
	
	// Should return at least one result
	if len(results) == 0 {
		t.Error("Expected checkRecipes to return at least one result")
	}
	
	// Verify all results have messages
	for i, result := range results {
		if result.message == "" {
			t.Errorf("Expected result %d to have a non-empty message", i)
		}
	}
}

func TestCheckInstallations(t *testing.T) {
	results := checkInstallations()
	
	// Should return at least one result
	if len(results) == 0 {
		t.Error("Expected checkInstallations to return at least one result")
	}
	
	// Verify all results have messages
	for i, result := range results {
		if result.message == "" {
			t.Errorf("Expected result %d to have a non-empty message", i)
		}
	}
}

func TestCheckNetwork(t *testing.T) {
	results := checkNetwork()
	
	// Should return at least one result (for each source checked)
	if len(results) < 1 {
		t.Error("Expected checkNetwork to return at least one result")
	}
	
	// Verify all results have messages
	for i, result := range results {
		if result.message == "" {
			t.Errorf("Expected result %d to have a non-empty message", i)
		}
	}
}
