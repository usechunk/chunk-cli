package sources

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alexinslc/chunk/internal/config"
)

func AuthenticateChunkHub() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	if cfg.ChunkHubAPIKey != "" {
		fmt.Println("✓ Already authenticated with ChunkHub")
		fmt.Println("  To re-authenticate, run: chunk auth chunkhub")
		return nil
	}
	
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔑 ChunkHub Authentication")
	fmt.Println()
	fmt.Println("To upload and manage modpacks on ChunkHub, you need")
	fmt.Println("to authenticate with an API key.")
	fmt.Println()
	fmt.Println("Get your API key from: https://chunkhub.io/account/keys")
	fmt.Println()
	fmt.Print("Enter your ChunkHub API key (or press Enter to skip): ")
	
	reader := bufio.NewReader(os.Stdin)
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	
	apiKey = strings.TrimSpace(apiKey)
	
	if apiKey == "" {
		fmt.Println("⚠ Skipped authentication. Some features may be limited.")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		return nil
	}
	
	client := NewChunkHubClient("")
	client.SetAPIKey(apiKey)
	
	if err := validateAPIKey(client); err != nil {
		fmt.Printf("✗ Invalid API key: %v\n", err)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		return fmt.Errorf("authentication failed")
	}
	
	cfg.SetChunkHubAPIKey(apiKey)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	
	fmt.Println("✓ Successfully authenticated with ChunkHub!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	
	return nil
}

func validateAPIKey(client *ChunkHubClient) error {
	_, err := client.Search("test")
	return err
}

func GetAuthenticatedChunkHubClient() *ChunkHubClient {
	client := NewChunkHubClient("")
	
	cfg, err := config.Load()
	if err == nil && cfg.ChunkHubAPIKey != "" {
		client.SetAPIKey(cfg.ChunkHubAPIKey)
	}
	
	return client
}
