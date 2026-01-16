package embeddings

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewClient tests the default client constructor
func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.endpoint != DefaultEndpoint {
		t.Errorf("Expected endpoint %s, got %s", DefaultEndpoint, client.endpoint)
	}
	if client.model != DefaultCodeModel {
		t.Errorf("Expected model %s, got %s", DefaultCodeModel, client.model)
	}
	if client.apiKey != "" {
		t.Errorf("Expected empty API key, got %s", client.apiKey)
	}
}

// TestNewClientWithModel tests client with custom model
func TestNewClientWithModel(t *testing.T) {
	customModel := "test-model"
	client := NewClientWithModel(customModel)
	if client.model != customModel {
		t.Errorf("Expected model %s, got %s", customModel, client.model)
	}
	if client.endpoint != DefaultEndpoint {
		t.Errorf("Expected default endpoint %s, got %s", DefaultEndpoint, client.endpoint)
	}
}

// TestNewClientWithEndpoint tests client with custom endpoint and model
func TestNewClientWithEndpoint(t *testing.T) {
	customEndpoint := "http://custom:8080"
	customModel := "custom-model"
	client := NewClientWithEndpoint(customEndpoint, customModel)
	if client.endpoint != customEndpoint {
		t.Errorf("Expected endpoint %s, got %s", customEndpoint, client.endpoint)
	}
	if client.model != customModel {
		t.Errorf("Expected model %s, got %s", customModel, client.model)
	}
}

// TestNewClientWithConfig tests client with full config
func TestNewClientWithConfig(t *testing.T) {
	customEndpoint := "http://custom:8080"
	customKey := "sk-test-key"
	customModel := "custom-model"
	client := NewClientWithConfig(customEndpoint, customKey, customModel)
	if client.endpoint != customEndpoint {
		t.Errorf("Expected endpoint %s, got %s", customEndpoint, client.endpoint)
	}
	if client.apiKey != customKey {
		t.Errorf("Expected API key %s, got %s", customKey, client.apiKey)
	}
	if client.model != customModel {
		t.Errorf("Expected model %s, got %s", customModel, client.model)
	}
}

// TestDeprecatedConstructors tests backwards compatibility
func TestDeprecatedConstructors(t *testing.T) {
	t.Run("NewOllamaClient", func(t *testing.T) {
		client := NewOllamaClient()
		if client == nil {
			t.Fatal("NewOllamaClient returned nil")
		}
		if client.model != DefaultCodeModel {
			t.Errorf("Expected model %s, got %s", DefaultCodeModel, client.model)
		}
	})

	t.Run("NewOllamaClientWithModel", func(t *testing.T) {
		customModel := "test-model"
		client := NewOllamaClientWithModel(customModel)
		if client.model != customModel {
			t.Errorf("Expected model %s, got %s", customModel, client.model)
		}
	})

	t.Run("NewOllamaClientWithEndpoint", func(t *testing.T) {
		customEndpoint := "http://test:8080"
		customModel := "test-model"
		client := NewOllamaClientWithEndpoint(customEndpoint, customModel)
		if client.endpoint != customEndpoint {
			t.Errorf("Expected endpoint %s, got %s", customEndpoint, client.endpoint)
		}
	})
}

// createMockEmbeddingServer creates a test HTTP server that simulates the embedding API
func createMockEmbeddingServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// TestEmbedSuccess tests successful single embedding
func TestEmbedSuccess(t *testing.T) {
	server := createMockEmbeddingServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("Expected path /v1/embeddings, got %s", r.URL.Path)
		}

		// Check request body
		var req openAIEmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Model != "test-model" {
			t.Errorf("Expected model test-model, got %s", req.Model)
		}

		// Return mock embedding
		resp := openAIEmbedResponse{
			Data: []struct {
				Embedding []float64 `json:"embedding"`
			}{
				{Embedding: []float64{0.1, 0.2, 0.3}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	client := NewClientWithEndpoint(server.URL, "test-model")
	embedding, err := client.Embed("test text")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(embedding) != 3 {
		t.Errorf("Expected 3 dimensions, got %d", len(embedding))
	}
	if embedding[0] != 0.1 || embedding[1] != 0.2 || embedding[2] != 0.3 {
		t.Errorf("Unexpected embedding values: %v", embedding)
	}
}

// TestEmbedManySuccess tests successful batch embeddings
func TestEmbedManySuccess(t *testing.T) {
	server := createMockEmbeddingServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req openAIEmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Check that we received multiple inputs
		inputs, ok := req.Input.([]interface{})
		if !ok {
			t.Fatalf("Expected Input to be array, got %T", req.Input)
		}
		if len(inputs) != 3 {
			t.Errorf("Expected 3 inputs, got %d", len(inputs))
		}

		// Return mock embeddings
		resp := openAIEmbedResponse{
			Data: []struct {
				Embedding []float64 `json:"embedding"`
			}{
				{Embedding: []float64{0.1, 0.2}},
				{Embedding: []float64{0.3, 0.4}},
				{Embedding: []float64{0.5, 0.6}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	client := NewClientWithEndpoint(server.URL, "test-model")
	embeddings, err := client.EmbedMany([]string{"text1", "text2", "text3"})
	if err != nil {
		t.Fatalf("EmbedMany failed: %v", err)
	}

	if len(embeddings) != 3 {
		t.Errorf("Expected 3 embeddings, got %d", len(embeddings))
	}
	if len(embeddings[0]) != 2 {
		t.Errorf("Expected 2 dimensions, got %d", len(embeddings[0]))
	}
}

// TestEmbedBatchAlias tests that EmbedBatch is an alias for EmbedMany
func TestEmbedBatchAlias(t *testing.T) {
	server := createMockEmbeddingServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := openAIEmbedResponse{
			Data: []struct {
				Embedding []float64 `json:"embedding"`
			}{
				{Embedding: []float64{0.1, 0.2}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	client := NewClientWithEndpoint(server.URL, "test-model")
	embeddings, err := client.EmbedBatch([]string{"text1"})
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}
	if len(embeddings) != 1 {
		t.Errorf("Expected 1 embedding, got %d", len(embeddings))
	}
}

// TestEmbedManyEmpty tests empty input handling
func TestEmbedManyEmpty(t *testing.T) {
	client := NewClient()
	embeddings, err := client.EmbedMany([]string{})
	if err != nil {
		t.Errorf("EmbedMany with empty input should not error, got: %v", err)
	}
	if embeddings != nil {
		t.Errorf("Expected nil embeddings for empty input, got %v", embeddings)
	}
}

// TestEmbedWithAPIKey tests that API key is sent in Authorization header
func TestEmbedWithAPIKey(t *testing.T) {
	expectedKey := "sk-test-key-123"
	keyReceived := false

	server := createMockEmbeddingServer(t, func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "Bearer "+expectedKey {
			keyReceived = true
		}

		resp := openAIEmbedResponse{
			Data: []struct {
				Embedding []float64 `json:"embedding"`
			}{
				{Embedding: []float64{0.1, 0.2}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	client := NewClientWithConfig(server.URL, expectedKey, "test-model")
	_, err := client.Embed("test")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if !keyReceived {
		t.Error("API key was not sent in Authorization header")
	}
}

// TestEmbedHTTPError tests handling of non-200 status codes
func TestEmbedHTTPError(t *testing.T) {
	server := createMockEmbeddingServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	})

	client := NewClientWithEndpoint(server.URL, "test-model")
	_, err := client.Embed("test")
	if err == nil {
		t.Fatal("Expected error for 500 status code")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Expected error to mention status code 500, got: %v", err)
	}
}

// TestEmbedInvalidJSON tests handling of invalid JSON response
func TestEmbedInvalidJSON(t *testing.T) {
	server := createMockEmbeddingServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	})

	client := NewClientWithEndpoint(server.URL, "test-model")
	_, err := client.Embed("test")
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("Expected decode error, got: %v", err)
	}
}

// TestEmbedEmptyResponse tests handling of empty response data
func TestEmbedEmptyResponse(t *testing.T) {
	server := createMockEmbeddingServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := openAIEmbedResponse{Data: []struct {
			Embedding []float64 `json:"embedding"`
		}{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	client := NewClientWithEndpoint(server.URL, "test-model")
	_, err := client.Embed("test")
	if err == nil {
		t.Fatal("Expected error for empty response")
	}
	if !strings.Contains(err.Error(), "no embedding") {
		t.Errorf("Expected 'no embedding' error, got: %v", err)
	}
}

// TestEmbedMismatchedCount tests handling of mismatched embedding count
func TestEmbedMismatchedCount(t *testing.T) {
	server := createMockEmbeddingServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Return 2 embeddings when 3 were requested
		resp := openAIEmbedResponse{
			Data: []struct {
				Embedding []float64 `json:"embedding"`
			}{
				{Embedding: []float64{0.1, 0.2}},
				{Embedding: []float64{0.3, 0.4}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	client := NewClientWithEndpoint(server.URL, "test-model")
	_, err := client.EmbedMany([]string{"text1", "text2", "text3"})
	if err == nil {
		t.Fatal("Expected error for mismatched embedding count")
	}
	if !strings.Contains(err.Error(), "expected 3") {
		t.Errorf("Expected count mismatch error, got: %v", err)
	}
}

// TestEmbedRetrySuccess tests retry logic when first attempt fails
func TestEmbedRetrySuccess(t *testing.T) {
	attemptCount := atomic.Int32{}

	server := createMockEmbeddingServer(t, func(w http.ResponseWriter, r *http.Request) {
		count := attemptCount.Add(1)
		if count < 2 {
			// Fail first attempt
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Succeed on second attempt
		resp := openAIEmbedResponse{
			Data: []struct {
				Embedding []float64 `json:"embedding"`
			}{
				{Embedding: []float64{0.1, 0.2}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	client := NewClientWithEndpoint(server.URL, "test-model")
	// Reduce backoff time for faster test
	client.client.Timeout = 5 * time.Second

	embedding, err := client.Embed("test")
	if err != nil {
		t.Fatalf("Embed should succeed after retry, got error: %v", err)
	}
	if len(embedding) != 2 {
		t.Errorf("Expected 2 dimensions, got %d", len(embedding))
	}
	if attemptCount.Load() != 2 {
		t.Errorf("Expected 2 attempts, got %d", attemptCount.Load())
	}
}

// TestEmbedRetryExhaustion tests retry logic when all attempts fail
func TestEmbedRetryExhaustion(t *testing.T) {
	attemptCount := atomic.Int32{}

	server := createMockEmbeddingServer(t, func(w http.ResponseWriter, r *http.Request) {
		attemptCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("persistent error"))
	})

	client := NewClientWithEndpoint(server.URL, "test-model")
	client.client.Timeout = 5 * time.Second

	_, err := client.Embed("test")
	if err == nil {
		t.Fatal("Expected error after all retries exhausted")
	}
	if !strings.Contains(err.Error(), "failed after 3 attempts") {
		t.Errorf("Expected retry exhaustion message, got: %v", err)
	}
	if attemptCount.Load() != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount.Load())
	}
}

// TestEmbedNetworkError tests handling of network errors
func TestEmbedNetworkError(t *testing.T) {
	// Use invalid endpoint that will cause connection error
	client := NewClientWithEndpoint("http://invalid-host-that-does-not-exist:9999", "test-model")
	client.client.Timeout = 1 * time.Second

	_, err := client.Embed("test")
	if err == nil {
		t.Fatal("Expected network error")
	}
	if !strings.Contains(err.Error(), "failed after 3 attempts") {
		t.Errorf("Expected retry failure, got: %v", err)
	}
}

// TestEmbedContentType tests that Content-Type header is set
func TestEmbedContentType(t *testing.T) {
	contentTypeReceived := false

	server := createMockEmbeddingServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") == "application/json" {
			contentTypeReceived = true
		}

		resp := openAIEmbedResponse{
			Data: []struct {
				Embedding []float64 `json:"embedding"`
			}{
				{Embedding: []float64{0.1, 0.2}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	client := NewClientWithEndpoint(server.URL, "test-model")
	_, err := client.Embed("test")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if !contentTypeReceived {
		t.Error("Content-Type header was not set to application/json")
	}
}

// BenchmarkEmbed benchmarks single embedding
func BenchmarkEmbed(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIEmbedResponse{
			Data: []struct {
				Embedding []float64 `json:"embedding"`
			}{
				{Embedding: make([]float64, 768)}, // Realistic dimension count
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithEndpoint(server.URL, "test-model")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Embed(fmt.Sprintf("test text %d", i))
		if err != nil {
			b.Fatalf("Embed failed: %v", err)
		}
	}
}

// BenchmarkEmbedMany benchmarks batch embeddings
func BenchmarkEmbedMany(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIEmbedRequest
		json.NewDecoder(r.Body).Decode(&req)

		inputs := req.Input.([]interface{})
		data := make([]struct {
			Embedding []float64 `json:"embedding"`
		}, len(inputs))
		for i := range data {
			data[i].Embedding = make([]float64, 768)
		}

		resp := openAIEmbedResponse{Data: data}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithEndpoint(server.URL, "test-model")
	texts := []string{"text1", "text2", "text3", "text4", "text5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.EmbedMany(texts)
		if err != nil {
			b.Fatalf("EmbedMany failed: %v", err)
		}
	}
}
