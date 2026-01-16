package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Endpoint != "http://localhost:11434" {
		t.Errorf("expected default endpoint http://localhost:11434, got %s", cfg.Endpoint)
	}
	if cfg.CodeModel != "code-scout-code" {
		t.Errorf("expected default code model code-scout-code, got %s", cfg.CodeModel)
	}
	if cfg.TextModel != "code-scout-text" {
		t.Errorf("expected default text model code-scout-text, got %s", cfg.TextModel)
	}
	if cfg.RerankModel != "" {
		t.Errorf("expected default rerank model to be empty, got %s", cfg.RerankModel)
	}
	if cfg.RerankTopK != 0 {
		t.Errorf("expected default rerank_top_k to be 0, got %d", cfg.RerankTopK)
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Write test config
	testConfig := `{
  "endpoint": "http://custom:8080",
  "code_model": "custom-code",
  "text_model": "custom-text",
  "rerank_model": "custom-rerank",
  "rerank_top_k": 25
}`
	if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load config
	cfg, err := loadFromFile(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Endpoint != "http://custom:8080" {
		t.Errorf("expected endpoint http://custom:8080, got %s", cfg.Endpoint)
	}
	if cfg.CodeModel != "custom-code" {
		t.Errorf("expected code model custom-code, got %s", cfg.CodeModel)
	}
	if cfg.TextModel != "custom-text" {
		t.Errorf("expected text model custom-text, got %s", cfg.TextModel)
	}
	if cfg.RerankModel != "custom-rerank" {
		t.Errorf("expected rerank model custom-rerank, got %s", cfg.RerankModel)
	}
	if cfg.RerankTopK != 25 {
		t.Errorf("expected rerank top k 25, got %d", cfg.RerankTopK)
	}
}

func TestLoadFromFile_NotExists(t *testing.T) {
	cfg, err := loadFromFile("/nonexistent/path/config.json")
	if err != nil {
		t.Errorf("expected no error for non-existent file, got: %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config for non-existent file, got: %v", cfg)
	}
}

func TestMergeConfig(t *testing.T) {
	dst := Default()
	src := &Config{
		Endpoint:    "http://custom:8080",
		RerankModel: "custom-rerank",
		RerankTopK:  20,
		// CodeModel and TextModel left empty
	}

	mergeConfig(dst, src)

	if dst.Endpoint != "http://custom:8080" {
		t.Errorf("expected merged endpoint http://custom:8080, got %s", dst.Endpoint)
	}
	// Should keep defaults for empty fields
	if dst.CodeModel != "code-scout-code" {
		t.Errorf("expected default code model, got %s", dst.CodeModel)
	}
	if dst.TextModel != "code-scout-text" {
		t.Errorf("expected default text model, got %s", dst.TextModel)
	}
	if dst.RerankModel != "custom-rerank" {
		t.Errorf("expected rerank model custom-rerank, got %s", dst.RerankModel)
	}
	if dst.RerankTopK != 20 {
		t.Errorf("expected rerank top k 20, got %d", dst.RerankTopK)
	}
}

func TestSave(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "subdir", "config.json")

	cfg := &Config{
		Endpoint:    "http://test:9000",
		CodeModel:   "test-code",
		TextModel:   "test-text",
		RerankModel: "test-rerank",
		RerankTopK:  30,
	}

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("config file was not created")
	}

	// Load and verify
	loaded, err := loadFromFile(configPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if loaded.Endpoint != cfg.Endpoint {
		t.Errorf("expected endpoint %s, got %s", cfg.Endpoint, loaded.Endpoint)
	}
	if loaded.CodeModel != cfg.CodeModel {
		t.Errorf("expected code model %s, got %s", cfg.CodeModel, loaded.CodeModel)
	}
	if loaded.TextModel != cfg.TextModel {
		t.Errorf("expected text model %s, got %s", cfg.TextModel, loaded.TextModel)
	}
	if loaded.RerankModel != cfg.RerankModel {
		t.Errorf("expected rerank model %s, got %s", cfg.RerankModel, loaded.RerankModel)
	}
	if loaded.RerankTopK != cfg.RerankTopK {
		t.Errorf("expected rerank top k %d, got %d", cfg.RerankTopK, loaded.RerankTopK)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		expectErr bool
	}{
		{
			name:      "valid config",
			config:    Default(),
			expectErr: false,
		},
		{
			name: "valid https endpoint",
			config: &Config{
				Endpoint:    "https://api.example.com",
				CodeModel:   "model1",
				TextModel:   "model2",
				RerankModel: "model3",
				RerankTopK:  15,
			},
			expectErr: false,
		},
		{
			name: "endpoint with trailing slash",
			config: &Config{
				Endpoint:    "http://localhost:11434/",
				CodeModel:   "model1",
				TextModel:   "model2",
				RerankModel: "model3",
				RerankTopK:  15,
			},
			expectErr: false,
		},
		{
			name: "empty endpoint",
			config: &Config{
				Endpoint:  "",
				CodeModel: "model1",
				TextModel: "model2",
			},
			expectErr: true,
		},
		{
			name: "invalid endpoint scheme",
			config: &Config{
				Endpoint:  "ftp://localhost:11434",
				CodeModel: "model1",
				TextModel: "model2",
			},
			expectErr: true,
		},
		{
			name: "empty code model",
			config: &Config{
				Endpoint:  "http://localhost:11434",
				CodeModel: "",
				TextModel: "model2",
			},
			expectErr: true,
		},
		{
			name: "empty text model",
			config: &Config{
				Endpoint:  "http://localhost:11434",
				CodeModel: "model1",
				TextModel: "",
			},
			expectErr: true,
		},
		{
			name: "negative rerank top k",
			config: &Config{
				Endpoint:    "http://localhost:11434",
				CodeModel:   "model1",
				TextModel:   "model2",
				RerankModel: "model3",
				RerankTopK:  -1,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidate_TrimsTrailingSlash(t *testing.T) {
	cfg := &Config{
		Endpoint:  "http://localhost:11434/",
		CodeModel: "model1",
		TextModel: "model2",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Endpoint != "http://localhost:11434" {
		t.Errorf("expected endpoint without trailing slash, got: %s", cfg.Endpoint)
	}
}

func TestLoad_NoConfigFiles(t *testing.T) {
	// Save original working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Save original HOME env var
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Create temp directory and change to it
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Set HOME to a temp location where no config exists
	tempHome := filepath.Join(tempDir, "home")
	os.Setenv("HOME", tempHome)

	// Load should return default config when no files exist
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be default config
	if cfg.Endpoint != "http://localhost:11434" {
		t.Errorf("expected default endpoint, got %s", cfg.Endpoint)
	}
	if cfg.CodeModel != "code-scout-code" {
		t.Errorf("expected default code model, got %s", cfg.CodeModel)
	}
	if cfg.TextModel != "code-scout-text" {
		t.Errorf("expected default text model, got %s", cfg.TextModel)
	}
}

func TestLoad_UserConfigOnly(t *testing.T) {
	// Save original working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Save original HOME env var
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Create temp directory
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Create user config
	tempHome := filepath.Join(tempDir, "home")
	os.Setenv("HOME", tempHome)
	userConfigDir := filepath.Join(tempHome, ".code-scout")
	if err := os.MkdirAll(userConfigDir, 0755); err != nil {
		t.Fatalf("failed to create user config dir: %v", err)
	}

	userConfig := `{
  "endpoint": "http://user-config:9000",
  "code_model": "user-code",
  "text_model": "user-text"
}`
	userConfigPath := filepath.Join(userConfigDir, "config.json")
	if err := os.WriteFile(userConfigPath, []byte(userConfig), 0644); err != nil {
		t.Fatalf("failed to write user config: %v", err)
	}

	// Load config (no project config exists)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have user config values
	if cfg.Endpoint != "http://user-config:9000" {
		t.Errorf("expected user endpoint, got %s", cfg.Endpoint)
	}
	if cfg.CodeModel != "user-code" {
		t.Errorf("expected user code model, got %s", cfg.CodeModel)
	}
	if cfg.TextModel != "user-text" {
		t.Errorf("expected user text model, got %s", cfg.TextModel)
	}
}

func TestLoad_ProjectConfigOnly(t *testing.T) {
	// Save original working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Save original HOME env var
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Create temp directory
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Set HOME to location with no user config
	tempHome := filepath.Join(tempDir, "home")
	os.Setenv("HOME", tempHome)

	// Create project config
	projectConfig := `{
  "endpoint": "http://project-config:8000",
  "code_model": "project-code",
  "text_model": "project-text",
  "rerank_model": "project-rerank",
  "rerank_top_k": 50
}`
	if err := os.WriteFile(".code-scout.json", []byte(projectConfig), 0644); err != nil {
		t.Fatalf("failed to write project config: %v", err)
	}

	// Load config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have project config values
	if cfg.Endpoint != "http://project-config:8000" {
		t.Errorf("expected project endpoint, got %s", cfg.Endpoint)
	}
	if cfg.CodeModel != "project-code" {
		t.Errorf("expected project code model, got %s", cfg.CodeModel)
	}
	if cfg.TextModel != "project-text" {
		t.Errorf("expected project text model, got %s", cfg.TextModel)
	}
	if cfg.RerankModel != "project-rerank" {
		t.Errorf("expected project rerank model, got %s", cfg.RerankModel)
	}
	if cfg.RerankTopK != 50 {
		t.Errorf("expected project rerank top k 50, got %d", cfg.RerankTopK)
	}
}

func TestLoad_BothConfigs_ProjectOverridesUser(t *testing.T) {
	// Save original working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Save original HOME env var
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Create temp directory
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Create user config
	tempHome := filepath.Join(tempDir, "home")
	os.Setenv("HOME", tempHome)
	userConfigDir := filepath.Join(tempHome, ".code-scout")
	if err := os.MkdirAll(userConfigDir, 0755); err != nil {
		t.Fatalf("failed to create user config dir: %v", err)
	}

	userConfig := `{
  "endpoint": "http://user:9000",
  "code_model": "user-code",
  "text_model": "user-text",
  "rerank_model": "user-rerank",
  "rerank_top_k": 25
}`
	userConfigPath := filepath.Join(userConfigDir, "config.json")
	if err := os.WriteFile(userConfigPath, []byte(userConfig), 0644); err != nil {
		t.Fatalf("failed to write user config: %v", err)
	}

	// Create project config (partial - only overrides some fields)
	projectConfig := `{
  "endpoint": "http://project:8000",
  "code_model": "project-code"
}`
	if err := os.WriteFile(".code-scout.json", []byte(projectConfig), 0644); err != nil {
		t.Fatalf("failed to write project config: %v", err)
	}

	// Load config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Project config should override user config for specified fields
	if cfg.Endpoint != "http://project:8000" {
		t.Errorf("expected project endpoint, got %s", cfg.Endpoint)
	}
	if cfg.CodeModel != "project-code" {
		t.Errorf("expected project code model, got %s", cfg.CodeModel)
	}
	// But keep user config for unspecified fields
	if cfg.TextModel != "user-text" {
		t.Errorf("expected user text model, got %s", cfg.TextModel)
	}
	if cfg.RerankModel != "user-rerank" {
		t.Errorf("expected user rerank model, got %s", cfg.RerankModel)
	}
	if cfg.RerankTopK != 25 {
		t.Errorf("expected user rerank top k 25, got %d", cfg.RerankTopK)
	}
}

func TestLoadFromFile_MalformedJSON(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Write malformed JSON
	malformedJSON := `{
  "endpoint": "http://localhost:11434",
  "code_model": "test"
  missing comma here
}`
	if err := os.WriteFile(configPath, []byte(malformedJSON), 0644); err != nil {
		t.Fatalf("failed to write malformed config: %v", err)
	}

	// Load should return error
	cfg, err := loadFromFile(configPath)
	if err == nil {
		t.Errorf("expected error for malformed JSON, got nil")
	}
	if cfg != nil {
		t.Errorf("expected nil config for malformed JSON, got: %v", cfg)
	}
}

func TestLoadFromFile_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Write completely invalid JSON
	if err := os.WriteFile(configPath, []byte("not json at all"), 0644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	// Load should return error
	cfg, err := loadFromFile(configPath)
	if err == nil {
		t.Errorf("expected error for invalid JSON, got nil")
	}
	if cfg != nil {
		t.Errorf("expected nil config for invalid JSON, got: %v", cfg)
	}
}

func TestMergeConfig_EmptyAPIKey(t *testing.T) {
	// Test that empty API key in source doesn't override existing API key in destination
	dst := &Config{
		Endpoint:  "http://localhost:11434",
		APIKey:    "existing-key",
		CodeModel: "code-scout-code",
		TextModel: "code-scout-text",
	}

	src := &Config{
		Endpoint: "http://newhost:8080",
		APIKey:   "", // Empty - should not override
	}

	mergeConfig(dst, src)

	// API key should remain unchanged
	if dst.APIKey != "existing-key" {
		t.Errorf("expected API key to remain 'existing-key', got %s", dst.APIKey)
	}
	// But endpoint should be updated
	if dst.Endpoint != "http://newhost:8080" {
		t.Errorf("expected endpoint to update, got %s", dst.Endpoint)
	}
}

func TestMergeConfig_ZeroRerankTopK(t *testing.T) {
	// Test that zero RerankTopK in source doesn't override non-zero value in destination
	dst := &Config{
		Endpoint:    "http://localhost:11434",
		CodeModel:   "code-scout-code",
		TextModel:   "code-scout-text",
		RerankModel: "rerank",
		RerankTopK:  25,
	}

	src := &Config{
		RerankModel: "new-rerank",
		RerankTopK:  0, // Zero - should not override
	}

	mergeConfig(dst, src)

	// RerankTopK should remain unchanged
	if dst.RerankTopK != 25 {
		t.Errorf("expected RerankTopK to remain 25, got %d", dst.RerankTopK)
	}
	// But RerankModel should be updated
	if dst.RerankModel != "new-rerank" {
		t.Errorf("expected RerankModel to update, got %s", dst.RerankModel)
	}
}

func TestMergeConfig_AllFieldsOverride(t *testing.T) {
	// Test that all non-empty fields in source override destination
	dst := &Config{
		Endpoint:    "http://old:1111",
		APIKey:      "old-key",
		CodeModel:   "old-code",
		TextModel:   "old-text",
		RerankModel: "old-rerank",
		RerankTopK:  10,
	}

	src := &Config{
		Endpoint:    "http://new:2222",
		APIKey:      "new-key",
		CodeModel:   "new-code",
		TextModel:   "new-text",
		RerankModel: "new-rerank",
		RerankTopK:  20,
	}

	mergeConfig(dst, src)

	if dst.Endpoint != "http://new:2222" {
		t.Errorf("expected Endpoint to update, got %s", dst.Endpoint)
	}
	if dst.APIKey != "new-key" {
		t.Errorf("expected APIKey to update, got %s", dst.APIKey)
	}
	if dst.CodeModel != "new-code" {
		t.Errorf("expected CodeModel to update, got %s", dst.CodeModel)
	}
	if dst.TextModel != "new-text" {
		t.Errorf("expected TextModel to update, got %s", dst.TextModel)
	}
	if dst.RerankModel != "new-rerank" {
		t.Errorf("expected RerankModel to update, got %s", dst.RerankModel)
	}
	if dst.RerankTopK != 20 {
		t.Errorf("expected RerankTopK to update, got %d", dst.RerankTopK)
	}
}

func TestValidate_InvalidURL(t *testing.T) {
	cfg := &Config{
		Endpoint:  "://invalid-url",
		CodeModel: "model1",
		TextModel: "model2",
	}

	err := cfg.Validate()
	if err == nil {
		t.Errorf("expected error for invalid URL, got nil")
	}
}
