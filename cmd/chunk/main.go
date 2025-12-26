package main

import (
	"fmt"
	"os"

	"github.com/alexinslc/chunk/cmd/chunk/commands"
	"github.com/alexinslc/chunk/internal/bench"
	"github.com/alexinslc/chunk/internal/telemetry"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
)

var rootCmd = &cobra.Command{
	Use:   "chunk",
	Short: "Chunk - Modpack Server Toolkit",
	Long: `Chunk is a universal CLI tool for deploying modded Minecraft servers.

Deploy modded Minecraft servers with a single command—no more hours 
of manual mod installation.

Examples:
  chunk install atm9
  chunk install alexinslc/my-cool-mod
  chunk search "all the mods"
  chunk upgrade atm9
  chunk diff atm9`,
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := telemetry.PromptForTelemetry(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not handle telemetry prompt: %v\n", err)
		}
		if err := bench.EnsureCoreBench(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not auto-add core bench: %v\n", err)
		}
	},
}

func init() {
	rootCmd.SetVersionTemplate(`{{printf "chunk version %s\n" .Version}}`)
	rootCmd.AddCommand(commands.InstallCmd)
	rootCmd.AddCommand(commands.UninstallCmd)
	rootCmd.AddCommand(commands.SearchCmd)
	rootCmd.AddCommand(commands.UpgradeCmd)
	rootCmd.AddCommand(commands.DiffCmd)
	rootCmd.AddCommand(commands.CheckCmd)
	rootCmd.AddCommand(commands.BenchCmd)
	rootCmd.AddCommand(commands.UpdateCmd)
	rootCmd.AddCommand(commands.InfoCmd)
	rootCmd.AddCommand(commands.ListCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
