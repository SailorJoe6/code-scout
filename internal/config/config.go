package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the application configuration
type Config struct {
	Endpoint        string `json:"endpoint"`
	APIKey          string `json:"api_key,omitempty"`           // Optional API key for authentication
	CodeModel       string `json:"code_model"`
	TextModel       string `json:"text_model"`
	RerankModel     string `json:"rerank_model,omitempty"`      // Model name for reranking (enables reranking if set)
	RerankTopK      int    `json:"rerank_top_k,omitempty"`
	SingleModelMode *bool  `json:"single_model_mode,omitempty"` // Use single TEI process with model switching (default: true)
}

// Default returns the default configuration
func Default() *Config {
	singleModelMode := true
	cfg := &Config{
		Endpoint:        "http://localhost:11434",
		CodeModel:       "nomic-ai/CodeRankEmbed",
		TextModel:       "nomic-ai/nomic-embed-text-v1.5",
		SingleModelMode: &singleModelMode, // Default to single-model mode (TEI wrapper with model switching)
	}
	return cfg
}

// Load loads configuration from file paths in order of precedence:
// 1. Project-level: .code-scout.json in current directory
// 2. User-level: ~/.code-scout/config.json
// If no config file exists, returns default config
func Load() (*Config, error) {
	cfg := Default()

	// Try user-level config first
	if userConfig, err := loadUserConfig(); err == nil && userConfig != nil {
		mergeConfig(cfg, userConfig)
	}

	// Try project-level config (overrides user-level)
	if projectConfig, err := loadProjectConfig(); err == nil && projectConfig != nil {
		mergeConfig(cfg, projectConfig)
	}

	return cfg, nil
}

// loadUserConfig loads ~/.code-scout/config.json
func loadUserConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".code-scout", "config.json")
	return loadFromFile(configPath)
}

// loadProjectConfig loads .code-scout.json from current directory
func loadProjectConfig() (*Config, error) {
	return loadFromFile(".code-scout.json")
}

// loadFromFile loads configuration from a JSON file
func loadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist, not an error
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// mergeConfig merges non-empty values from src into dst
func mergeConfig(dst, src *Config) {
	if src.Endpoint != "" {
		dst.Endpoint = src.Endpoint
	}
	if src.APIKey != "" {
		dst.APIKey = src.APIKey
	}
	if src.CodeModel != "" {
		dst.CodeModel = src.CodeModel
	}
	if src.TextModel != "" {
		dst.TextModel = src.TextModel
	}
	if src.RerankModel != "" {
		dst.RerankModel = src.RerankModel
	}
	if src.RerankTopK != 0 {
		dst.RerankTopK = src.RerankTopK
	}
	// Merge SingleModelMode if explicitly set (non-nil)
	if src.SingleModelMode != nil {
		dst.SingleModelMode = src.SingleModelMode
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate endpoint is a valid URL
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}

	parsedURL, err := url.Parse(c.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("endpoint must use http or https scheme, got: %s", parsedURL.Scheme)
	}

	// Remove trailing slash if present
	c.Endpoint = strings.TrimSuffix(c.Endpoint, "/")

	// Validate models are not empty
	if c.CodeModel == "" {
		return fmt.Errorf("code_model cannot be empty")
	}
	if c.TextModel == "" {
		return fmt.Errorf("text_model cannot be empty")
	}
	if c.RerankTopK < 0 {
		return fmt.Errorf("rerank_top_k cannot be negative")
	}

	return nil
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
