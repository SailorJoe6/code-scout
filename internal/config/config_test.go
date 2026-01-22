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
	if cfg.CodeModel != "nomic-ai/CodeRankEmbed" {
		t.Errorf("expected default code model nomic-ai/CodeRankEmbed, got %s", cfg.CodeModel)
	}
	if cfg.TextModel != "nomic-ai/nomic-embed-text-v1.5" {
		t.Errorf("expected default text model nomic-ai/nomic-embed-text-v1.5, got %s", cfg.TextModel)
	}
	if cfg.RerankModel != "" {
		t.Errorf("expected default rerank model to be empty, got %s", cfg.RerankModel)
	}
	if cfg.RerankTopK != 0 {
		t.Errorf("expected default rerank_top_k to be 0, got %d", cfg.RerankTopK)
	}
	if cfg.SingleModelMode == nil {
		t.Errorf("expected default single_model_mode to be set, got nil")
	} else if *cfg.SingleModelMode != false {
		t.Errorf("expected default single_model_mode to be false, got %v", *cfg.SingleModelMode)
	}
	if cfg.TextTEIPort != 8080 {
		t.Errorf("expected default text_tei_port to be 8080, got %d", cfg.TextTEIPort)
	}
	if cfg.CodeTEIPort != 8082 {
		t.Errorf("expected default code_tei_port to be 8082, got %d", cfg.CodeTEIPort)
	}
	if cfg.RerankPort != 8081 {
		t.Errorf("expected default rerank_port to be 8081, got %d", cfg.RerankPort)
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
  "rerank_top_k": 25,
  "text_tei_port": 9100,
  "code_tei_port": 9200,
  "rerank_port": 9300
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
	if cfg.TextTEIPort != 9100 {
		t.Errorf("expected text_tei_port 9100, got %d", cfg.TextTEIPort)
	}
	if cfg.CodeTEIPort != 9200 {
		t.Errorf("expected code_tei_port 9200, got %d", cfg.CodeTEIPort)
	}
	if cfg.RerankPort != 9300 {
		t.Errorf("expected rerank_port 9300, got %d", cfg.RerankPort)
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
		CodeTEIPort: 9201,
		// CodeModel and TextModel left empty
	}

	mergeConfig(dst, src)

	if dst.Endpoint != "http://custom:8080" {
		t.Errorf("expected merged endpoint http://custom:8080, got %s", dst.Endpoint)
	}
	// Should keep defaults for empty fields
	if dst.CodeModel != "nomic-ai/CodeRankEmbed" {
		t.Errorf("expected default code model, got %s", dst.CodeModel)
	}
	if dst.TextModel != "nomic-ai/nomic-embed-text-v1.5" {
		t.Errorf("expected default text model, got %s", dst.TextModel)
	}
	if dst.RerankModel != "custom-rerank" {
		t.Errorf("expected rerank model custom-rerank, got %s", dst.RerankModel)
	}
	if dst.RerankTopK != 20 {
		t.Errorf("expected rerank top k 20, got %d", dst.RerankTopK)
	}
	if dst.CodeTEIPort != 9201 {
		t.Errorf("expected code_tei_port 9201, got %d", dst.CodeTEIPort)
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
		TextTEIPort: 9101,
		CodeTEIPort: 9201,
		RerankPort:  9301,
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
	if loaded.TextTEIPort != cfg.TextTEIPort {
		t.Errorf("expected text_tei_port %d, got %d", cfg.TextTEIPort, loaded.TextTEIPort)
	}
	if loaded.CodeTEIPort != cfg.CodeTEIPort {
		t.Errorf("expected code_tei_port %d, got %d", cfg.CodeTEIPort, loaded.CodeTEIPort)
	}
	if loaded.RerankPort != cfg.RerankPort {
		t.Errorf("expected rerank_port %d, got %d", cfg.RerankPort, loaded.RerankPort)
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
	if cfg.CodeModel != "nomic-ai/CodeRankEmbed" {
		t.Errorf("expected default code model, got %s", cfg.CodeModel)
	}
	if cfg.TextModel != "nomic-ai/nomic-embed-text-v1.5" {
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
		CodeModel: "nomic-ai/CodeRankEmbed",
		TextModel: "nomic-ai/nomic-embed-text-v1.5",
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
		CodeModel:   "nomic-ai/CodeRankEmbed",
		TextModel:   "nomic-ai/nomic-embed-text-v1.5",
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

func TestSingleModelMode_LoadFromFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		configJSON     string
		expectedValue  *bool
		expectedNil    bool
	}{
		{
			name: "explicit true",
			configJSON: `{
  "endpoint": "http://localhost:11434",
  "code_model": "model1",
  "text_model": "model2",
  "single_model_mode": true
}`,
			expectedValue: func() *bool { v := true; return &v }(),
			expectedNil:   false,
		},
		{
			name: "explicit false",
			configJSON: `{
  "endpoint": "http://localhost:11434",
  "code_model": "model1",
  "text_model": "model2",
  "single_model_mode": false
}`,
			expectedValue: func() *bool { v := false; return &v }(),
			expectedNil:   false,
		},
		{
			name: "not specified",
			configJSON: `{
  "endpoint": "http://localhost:11434",
  "code_model": "model1",
  "text_model": "model2"
}`,
			expectedValue: nil,
			expectedNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tempDir, tt.name+".json")
			if err := os.WriteFile(configPath, []byte(tt.configJSON), 0644); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			cfg, err := loadFromFile(configPath)
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			if tt.expectedNil {
				if cfg.SingleModelMode != nil {
					t.Errorf("expected SingleModelMode to be nil, got %v", *cfg.SingleModelMode)
				}
			} else {
				if cfg.SingleModelMode == nil {
					t.Errorf("expected SingleModelMode to be non-nil")
				} else if *cfg.SingleModelMode != *tt.expectedValue {
					t.Errorf("expected SingleModelMode %v, got %v", *tt.expectedValue, *cfg.SingleModelMode)
				}
			}
		})
	}
}

func TestSingleModelMode_MergeConfig(t *testing.T) {
	tests := []struct {
		name           string
		dstValue       *bool
		srcValue       *bool
		expectedValue  *bool
	}{
		{
			name:          "src true overrides dst nil",
			dstValue:      nil,
			srcValue:      func() *bool { v := true; return &v }(),
			expectedValue: func() *bool { v := true; return &v }(),
		},
		{
			name:          "src false overrides dst nil",
			dstValue:      nil,
			srcValue:      func() *bool { v := false; return &v }(),
			expectedValue: func() *bool { v := false; return &v }(),
		},
		{
			name:          "src false overrides dst true",
			dstValue:      func() *bool { v := true; return &v }(),
			srcValue:      func() *bool { v := false; return &v }(),
			expectedValue: func() *bool { v := false; return &v }(),
		},
		{
			name:          "src true overrides dst false",
			dstValue:      func() *bool { v := false; return &v }(),
			srcValue:      func() *bool { v := true; return &v }(),
			expectedValue: func() *bool { v := true; return &v }(),
		},
		{
			name:          "src nil keeps dst true",
			dstValue:      func() *bool { v := true; return &v }(),
			srcValue:      nil,
			expectedValue: func() *bool { v := true; return &v }(),
		},
		{
			name:          "src nil keeps dst false",
			dstValue:      func() *bool { v := false; return &v }(),
			srcValue:      nil,
			expectedValue: func() *bool { v := false; return &v }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := &Config{
				Endpoint:        "http://localhost:11434",
				CodeModel:       "model1",
				TextModel:       "model2",
				SingleModelMode: tt.dstValue,
			}

			src := &Config{
				SingleModelMode: tt.srcValue,
			}

			mergeConfig(dst, src)

			if tt.expectedValue == nil {
				if dst.SingleModelMode != nil {
					t.Errorf("expected SingleModelMode to be nil, got %v", *dst.SingleModelMode)
				}
			} else {
				if dst.SingleModelMode == nil {
					t.Errorf("expected SingleModelMode to be non-nil")
				} else if *dst.SingleModelMode != *tt.expectedValue {
					t.Errorf("expected SingleModelMode %v, got %v", *tt.expectedValue, *dst.SingleModelMode)
				}
			}
		})
	}
}

func TestSingleModelMode_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	singleModelMode := false
	cfg := &Config{
		Endpoint:        "http://test:9000",
		CodeModel:       "test-code",
		TextModel:       "test-text",
		SingleModelMode: &singleModelMode,
	}

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := loadFromFile(configPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if loaded.SingleModelMode == nil {
		t.Errorf("expected SingleModelMode to be set, got nil")
	} else if *loaded.SingleModelMode != false {
		t.Errorf("expected SingleModelMode to be false, got %v", *loaded.SingleModelMode)
	}
}

func TestLoad_ProjectConfigSearchesUpDirectoryTree(t *testing.T) {
	// Save original working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Save original HOME env var
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Create temp directory structure:
	// tempDir/
	//   .code-scout.json (root config)
	//   subdir1/
	//     subdir2/
	//       subdir3/ (we'll cd here)
	tempDir := t.TempDir()
	subdir1 := filepath.Join(tempDir, "subdir1")
	subdir2 := filepath.Join(subdir1, "subdir2")
	subdir3 := filepath.Join(subdir2, "subdir3")

	if err := os.MkdirAll(subdir3, 0755); err != nil {
		t.Fatalf("failed to create subdirs: %v", err)
	}

	// Set HOME to location with no user config
	tempHome := filepath.Join(tempDir, "home")
	os.Setenv("HOME", tempHome)

	// Create project config at root
	rootConfig := `{
  "endpoint": "http://root-config:7000",
  "code_model": "root-code",
  "text_model": "root-text",
  "rerank_model": "root-rerank"
}`
	rootConfigPath := filepath.Join(tempDir, ".code-scout.json")
	if err := os.WriteFile(rootConfigPath, []byte(rootConfig), 0644); err != nil {
		t.Fatalf("failed to write root config: %v", err)
	}

	// Change to deep subdirectory
	if err := os.Chdir(subdir3); err != nil {
		t.Fatalf("failed to change to subdir3: %v", err)
	}

	// Load config (should find the root config by searching up)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have root config values
	if cfg.Endpoint != "http://root-config:7000" {
		t.Errorf("expected root endpoint, got %s", cfg.Endpoint)
	}
	if cfg.CodeModel != "root-code" {
		t.Errorf("expected root code model, got %s", cfg.CodeModel)
	}
	if cfg.TextModel != "root-text" {
		t.Errorf("expected root text model, got %s", cfg.TextModel)
	}
	if cfg.RerankModel != "root-rerank" {
		t.Errorf("expected root rerank model, got %s", cfg.RerankModel)
	}
}

func TestLoad_ProjectConfigFindsNearestConfig(t *testing.T) {
	// Save original working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Save original HOME env var
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Create temp directory structure with configs at multiple levels:
	// tempDir/
	//   .code-scout.json (root config)
	//   subdir1/
	//     .code-scout.json (middle config)
	//     subdir2/ (we'll cd here)
	tempDir := t.TempDir()
	subdir1 := filepath.Join(tempDir, "subdir1")
	subdir2 := filepath.Join(subdir1, "subdir2")

	if err := os.MkdirAll(subdir2, 0755); err != nil {
		t.Fatalf("failed to create subdirs: %v", err)
	}

	// Set HOME to location with no user config
	tempHome := filepath.Join(tempDir, "home")
	os.Setenv("HOME", tempHome)

	// Create root config
	rootConfig := `{
  "endpoint": "http://root:7000",
  "code_model": "root-code"
}`
	rootConfigPath := filepath.Join(tempDir, ".code-scout.json")
	if err := os.WriteFile(rootConfigPath, []byte(rootConfig), 0644); err != nil {
		t.Fatalf("failed to write root config: %v", err)
	}

	// Create middle config (closer to where we'll cd)
	middleConfig := `{
  "endpoint": "http://middle:8000",
  "code_model": "middle-code"
}`
	middleConfigPath := filepath.Join(subdir1, ".code-scout.json")
	if err := os.WriteFile(middleConfigPath, []byte(middleConfig), 0644); err != nil {
		t.Fatalf("failed to write middle config: %v", err)
	}

	// Change to subdir2 (middle config is closest)
	if err := os.Chdir(subdir2); err != nil {
		t.Fatalf("failed to change to subdir2: %v", err)
	}

	// Load config (should find the middle config, not the root)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have middle config values
	if cfg.Endpoint != "http://middle:8000" {
		t.Errorf("expected middle endpoint, got %s", cfg.Endpoint)
	}
	if cfg.CodeModel != "middle-code" {
		t.Errorf("expected middle code model, got %s", cfg.CodeModel)
	}
}
