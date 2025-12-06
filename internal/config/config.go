package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Bench struct {
	Name        string     `json:"name"`
	URL         string     `json:"url"`
	Path        string     `json:"path"`
	Added       time.Time  `json:"added"`
	LastUpdated *time.Time `json:"last_updated,omitempty"`
}

type Config struct {
	TelemetryEnabled *bool   `json:"telemetry_enabled,omitempty"`
	TelemetryAsked   bool    `json:"telemetry_asked"`
	ConfigVersion    string  `json:"config_version"`
	ChunkHubAPIKey   string  `json:"chunkhub_api_key,omitempty"`
	Benches          []Bench `json:"benches,omitempty"`
}

func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".config", "chunk")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{
			ConfigVersion:  "1.0",
			TelemetryAsked: false,
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func (c *Config) IsTelemetryEnabled() bool {
	if c.TelemetryEnabled == nil {
		return false
	}
	return *c.TelemetryEnabled
}

func (c *Config) SetTelemetry(enabled bool) {
	c.TelemetryEnabled = &enabled
	c.TelemetryAsked = true
}

func (c *Config) GetChunkHubAPIKey() string {
	return c.ChunkHubAPIKey
}

func (c *Config) SetChunkHubAPIKey(apiKey string) {
	c.ChunkHubAPIKey = apiKey
}
