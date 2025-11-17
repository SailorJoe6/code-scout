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
}

func TestLoadFromFile(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Write test config
	testConfig := `{
  "endpoint": "http://custom:8080",
  "code_model": "custom-code",
  "text_model": "custom-text"
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
		Endpoint: "http://custom:8080",
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
}

func TestSave(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "subdir", "config.json")

	cfg := &Config{
		Endpoint:  "http://test:9000",
		CodeModel: "test-code",
		TextModel: "test-text",
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
				Endpoint:  "https://api.example.com",
				CodeModel: "model1",
				TextModel: "model2",
			},
			expectErr: false,
		},
		{
			name: "endpoint with trailing slash",
			config: &Config{
				Endpoint:  "http://localhost:11434/",
				CodeModel: "model1",
				TextModel: "model2",
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
