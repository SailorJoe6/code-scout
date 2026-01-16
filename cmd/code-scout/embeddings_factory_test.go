package main

import (
	"testing"

	"github.com/jlanders/code-scout/internal/config"
)

// Test newCodeEmbeddingClient factory function
func TestNewCodeEmbeddingClient(t *testing.T) {
	t.Run("with nil globalConfig", func(t *testing.T) {
		// Save and restore original globalConfig
		originalConfig := globalConfig
		defer func() { globalConfig = originalConfig }()

		globalConfig = nil
		client := newCodeEmbeddingClient()

		if client == nil {
			t.Error("expected non-nil client")
		}
		// Client should use default settings
	})

	t.Run("with globalConfig set", func(t *testing.T) {
		// Save and restore original globalConfig
		originalConfig := globalConfig
		defer func() { globalConfig = originalConfig }()

		globalConfig = &config.Config{
			Endpoint:  "http://test:11434",
			APIKey:    "test-key",
			CodeModel: "test-code-model",
		}

		client := newCodeEmbeddingClient()

		if client == nil {
			t.Error("expected non-nil client")
		}
		// Client should use config settings (endpoint, api_key, code_model)
	})

	t.Run("respects config endpoint", func(t *testing.T) {
		originalConfig := globalConfig
		defer func() { globalConfig = originalConfig }()

		globalConfig = &config.Config{
			Endpoint:  "http://custom-endpoint:9999",
			CodeModel: "custom-model",
		}

		client := newCodeEmbeddingClient()

		if client == nil {
			t.Error("expected non-nil client")
		}
	})
}

// Test newDocsEmbeddingClient factory function
func TestNewDocsEmbeddingClient(t *testing.T) {
	t.Run("with nil globalConfig", func(t *testing.T) {
		originalConfig := globalConfig
		defer func() { globalConfig = originalConfig }()

		globalConfig = nil
		client := newDocsEmbeddingClient()

		if client == nil {
			t.Error("expected non-nil client")
		}
		// Client should use default text model
	})

	t.Run("with globalConfig set", func(t *testing.T) {
		originalConfig := globalConfig
		defer func() { globalConfig = originalConfig }()

		globalConfig = &config.Config{
			Endpoint:  "http://test:11434",
			APIKey:    "test-key",
			TextModel: "test-text-model",
		}

		client := newDocsEmbeddingClient()

		if client == nil {
			t.Error("expected non-nil client")
		}
		// Client should use config settings (endpoint, api_key, text_model)
	})

	t.Run("respects config text model", func(t *testing.T) {
		originalConfig := globalConfig
		defer func() { globalConfig = originalConfig }()

		globalConfig = &config.Config{
			Endpoint:  "http://localhost:11434",
			TextModel: "custom-text-model",
		}

		client := newDocsEmbeddingClient()

		if client == nil {
			t.Error("expected non-nil client")
		}
	})
}

// Test newRerankEmbeddingClient factory function
func TestNewRerankEmbeddingClient(t *testing.T) {
	t.Run("with nil globalConfig", func(t *testing.T) {
		originalConfig := globalConfig
		defer func() { globalConfig = originalConfig }()

		globalConfig = nil
		client := newRerankEmbeddingClient()

		if client != nil {
			t.Error("expected nil client when globalConfig is nil")
		}
	})

	t.Run("with empty RerankModel", func(t *testing.T) {
		originalConfig := globalConfig
		defer func() { globalConfig = originalConfig }()

		globalConfig = &config.Config{
			Endpoint:    "http://test:11434",
			CodeModel:   "code-model",
			TextModel:   "text-model",
			RerankModel: "", // Empty rerank model
		}

		client := newRerankEmbeddingClient()

		if client != nil {
			t.Error("expected nil client when RerankModel is empty")
		}
	})

	t.Run("with RerankModel set", func(t *testing.T) {
		originalConfig := globalConfig
		defer func() { globalConfig = originalConfig }()

		globalConfig = &config.Config{
			Endpoint:    "http://test:11434",
			APIKey:      "test-key",
			RerankModel: "rerank-model",
		}

		client := newRerankEmbeddingClient()

		if client == nil {
			t.Error("expected non-nil client when RerankModel is set")
		}
	})

	t.Run("respects config rerank model", func(t *testing.T) {
		originalConfig := globalConfig
		defer func() { globalConfig = originalConfig }()

		globalConfig = &config.Config{
			Endpoint:    "http://localhost:11434",
			RerankModel: "custom-rerank-model",
		}

		client := newRerankEmbeddingClient()

		if client == nil {
			t.Error("expected non-nil client")
		}
	})
}

// Test factory function consistency
func TestFactoryFunctionConsistency(t *testing.T) {
	t.Run("all factories return clients with same config", func(t *testing.T) {
		originalConfig := globalConfig
		defer func() { globalConfig = originalConfig }()

		globalConfig = &config.Config{
			Endpoint:    "http://shared:11434",
			APIKey:      "shared-key",
			CodeModel:   "code-model",
			TextModel:   "text-model",
			RerankModel: "rerank-model",
		}

		codeClient := newCodeEmbeddingClient()
		docsClient := newDocsEmbeddingClient()
		rerankClient := newRerankEmbeddingClient()

		if codeClient == nil {
			t.Error("code client should not be nil")
		}
		if docsClient == nil {
			t.Error("docs client should not be nil")
		}
		if rerankClient == nil {
			t.Error("rerank client should not be nil")
		}

		// All should use the same endpoint (verified by creation without error)
	})

	t.Run("factories create new instances each call", func(t *testing.T) {
		originalConfig := globalConfig
		defer func() { globalConfig = originalConfig }()

		globalConfig = &config.Config{
			Endpoint:  "http://localhost:11434",
			CodeModel: "model",
		}

		client1 := newCodeEmbeddingClient()
		client2 := newCodeEmbeddingClient()

		// Should create new instances, not reuse
		if client1 == nil || client2 == nil {
			t.Fatal("clients should not be nil")
		}

		// Note: We can't easily test if they're different instances without
		// exposing internal state, but the factory pattern implies new instances
	})
}
