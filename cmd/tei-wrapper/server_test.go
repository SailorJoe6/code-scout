package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEmbeddingsEndpoint(t *testing.T) {
	// Create mock TEI server
	mockTEI := createMockTEI(t)
	defer mockTEI.Close()

	// Create wrapper server pointing to mock TEI
	server := &Server{
		teiBaseURL: mockTEI.URL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		initialModel: "test-model",
	}

	// Create test HTTP server with the wrapper handler
	testServer := httptest.NewServer(http.HandlerFunc(server.handleEmbeddings))
	defer testServer.Close()

	// Test case 1: Valid request with multiple inputs
	t.Run("ValidRequest", func(t *testing.T) {
		reqBody := EmbeddingRequest{
			Model: "test-model",
			Input: []string{"Hello world", "Testing embeddings"},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		// Parse response
		var embResp EmbeddingResponse
		if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify response structure
		if embResp.Object != "list" {
			t.Errorf("Expected object='list', got %s", embResp.Object)
		}

		if embResp.Model != "test-model" {
			t.Errorf("Expected model='test-model', got %s", embResp.Model)
		}

		if len(embResp.Data) != 2 {
			t.Fatalf("Expected 2 embeddings, got %d", len(embResp.Data))
		}

		// Verify first embedding
		if embResp.Data[0].Index != 0 {
			t.Errorf("Expected index=0, got %d", embResp.Data[0].Index)
		}

		if len(embResp.Data[0].Embedding) != 768 {
			t.Errorf("Expected 768-dim embedding, got %d", len(embResp.Data[0].Embedding))
		}

		// Verify second embedding
		if embResp.Data[1].Index != 1 {
			t.Errorf("Expected index=1, got %d", embResp.Data[1].Index)
		}
	})

	// Test case 2: Empty input
	t.Run("EmptyInput", func(t *testing.T) {
		reqBody := EmbeddingRequest{
			Model: "test-model",
			Input: []string{},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	// Test case 3: Invalid JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader([]byte("invalid json")))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	// Test case 4: Wrong HTTP method
	t.Run("WrongMethod", func(t *testing.T) {
		resp, err := http.Get(testServer.URL)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", resp.StatusCode)
		}
	})
}

func TestHealthEndpoint(t *testing.T) {
	// Create mock TEI server
	mockTEI := createMockTEI(t)
	defer mockTEI.Close()

	// Create wrapper server
	server := &Server{
		teiBaseURL: mockTEI.URL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		initialModel: "test-model",
	}

	// Create test HTTP server with health handler
	testServer := httptest.NewServer(http.HandlerFunc(server.handleHealth))
	defer testServer.Close()

	// Test health check
	resp, err := http.Get(testServer.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse response
	var health map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if health["status"] != "ok" {
		t.Errorf("Expected status='ok', got %s", health["status"])
	}

	if health["model"] != "test-model" {
		t.Errorf("Expected model='test-model', got %s", health["model"])
	}
}

func TestGetEmbeddings(t *testing.T) {
	// Create mock TEI server
	mockTEI := createMockTEI(t)
	defer mockTEI.Close()

	// Create wrapper server
	server := &Server{
		teiBaseURL: mockTEI.URL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Test getting embeddings
	inputs := []string{"test 1", "test 2", "test 3"}
	embeddings, err := server.getEmbeddings(inputs)
	if err != nil {
		t.Fatalf("getEmbeddings failed: %v", err)
	}

	if len(embeddings) != 3 {
		t.Fatalf("Expected 3 embeddings, got %d", len(embeddings))
	}

	for i, emb := range embeddings {
		if len(emb) != 768 {
			t.Errorf("Embedding %d: expected 768 dimensions, got %d", i, len(emb))
		}
	}
}
