package telemetry

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alexinslc/chunk/internal/config"
)

func PromptForTelemetry() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if cfg.TelemetryAsked {
		return nil
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 Help improve Chunk!")
	fmt.Println()
	fmt.Println("We'd like to collect anonymous usage data to improve")
	fmt.Println("the tool. This includes:")
	fmt.Println("  • Commands used")
	fmt.Println("  • Installation success/failure rates")
	fmt.Println("  • Error types encountered")
	fmt.Println()
	fmt.Println("We do NOT collect:")
	fmt.Println("  • Personal information")
	fmt.Println("  • Server names or IP addresses")
	fmt.Println("  • World data or configs")
	fmt.Println()
	fmt.Print("Enable telemetry? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	response = strings.TrimSpace(strings.ToLower(response))

	enabled := response == "" || response == "y" || response == "yes"
	cfg.SetTelemetry(enabled)

	if err := cfg.Save(); err != nil {
		return err
	}

	if enabled {
		fmt.Println("✓ Telemetry enabled. Thank you!")
	} else {
		fmt.Println("✓ Telemetry disabled.")
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}

func TrackEvent(eventName string, properties map[string]interface{}) {
	cfg, err := config.Load()
	if err != nil || !cfg.IsTelemetryEnabled() {
		return
	}

	// TODO: Implement actual telemetry sending
	// For now, this is a no-op placeholder
}
