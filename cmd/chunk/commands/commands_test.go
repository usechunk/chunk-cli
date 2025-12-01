package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err = root.Execute()
	return buf.String(), err
}

func TestInstallCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "install without args",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{Use: "chunk"}
			rootCmd.AddCommand(InstallCmd)

			_, err := executeCommand(rootCmd, append([]string{"install"}, tt.args...)...)
			if (err != nil) != tt.wantErr {
				t.Errorf("InstallCmd error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInstallCommandFlags(t *testing.T) {
	// Test that the install command has the expected flags
	rootCmd := &cobra.Command{Use: "chunk"}
	rootCmd.AddCommand(InstallCmd)

	// Check that --dir flag exists
	dirFlag := InstallCmd.Flags().Lookup("dir")
	if dirFlag == nil {
		t.Error("Expected --dir flag to exist")
	}
	if dirFlag.Shorthand != "d" {
		t.Errorf("Expected --dir shorthand to be 'd', got '%s'", dirFlag.Shorthand)
	}
}

func TestSearchCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "search with query",
			args:    []string{"all the mods"},
			wantErr: false,
		},
		{
			name:    "search with single word",
			args:    []string{"atm"},
			wantErr: false,
		},
		{
			name:    "search without args",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{Use: "chunk"}
			rootCmd.AddCommand(SearchCmd)

			_, err := executeCommand(rootCmd, append([]string{"search"}, tt.args...)...)
			if (err != nil) != tt.wantErr {
				t.Errorf("SearchCmd error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpgradeCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "upgrade with modpack name",
			args:    []string{"atm9"},
			wantErr: false,
		},
		{
			name:    "upgrade with directory",
			args:    []string{"atm9", "--dir", "/tmp/test"},
			wantErr: false,
		},
		{
			name:    "upgrade without args",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{Use: "chunk"}
			rootCmd.AddCommand(UpgradeCmd)

			_, err := executeCommand(rootCmd, append([]string{"upgrade"}, tt.args...)...)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpgradeCmd error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDiffCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "diff with modpack name",
			args:    []string{"atm9"},
			wantErr: false,
		},
		{
			name:    "diff with github repo",
			args:    []string{"alexinslc/my-modpack"},
			wantErr: false,
		},
		{
			name:    "diff without args",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{Use: "chunk"}
			rootCmd.AddCommand(DiffCmd)

			_, err := executeCommand(rootCmd, append([]string{"diff"}, tt.args...)...)
			if (err != nil) != tt.wantErr {
				t.Errorf("DiffCmd error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "check without args",
			args:    []string{},
			wantErr: false, // Should work on current directory
		},
		{
			name:    "check with modpack",
			args:    []string{"atm9"},
			wantErr: false,
		},
		{
			name:    "check with directory",
			args:    []string{"--dir", "/tmp"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{Use: "chunk"}
			rootCmd.AddCommand(CheckCmd)

			_, err := executeCommand(rootCmd, append([]string{"check"}, tt.args...)...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckCmd error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckCommandFlags(t *testing.T) {
	// Test that the check command has the expected flags
	rootCmd := &cobra.Command{Use: "chunk"}
	rootCmd.AddCommand(CheckCmd)

	// Check that --dir flag exists
	dirFlag := CheckCmd.Flags().Lookup("dir")
	if dirFlag == nil {
		t.Error("Expected --dir flag to exist")
	}
	if dirFlag.Shorthand != "d" {
		t.Errorf("Expected --dir shorthand to be 'd', got '%s'", dirFlag.Shorthand)
	}

	// Check that --format flag exists
	formatFlag := CheckCmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Error("Expected --format flag to exist")
	}
	if formatFlag.Shorthand != "f" {
		t.Errorf("Expected --format shorthand to be 'f', got '%s'", formatFlag.Shorthand)
	}
}
