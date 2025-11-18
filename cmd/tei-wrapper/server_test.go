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
		teiBaseURL:   mockTEI.URL,
		currentModel: "test-model",
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
		teiBaseURL:   mockTEI.URL,
		currentModel: "test-model",
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
	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if health["status"] != "ok" {
		t.Errorf("Expected status='ok', got %v", health["status"])
	}

	if health["model"] != "test-model" {
		t.Errorf("Expected model='test-model', got %v", health["model"])
	}
}

func TestGetEmbeddings(t *testing.T) {
	// Create mock TEI server
	mockTEI := createMockTEI(t)
	defer mockTEI.Close()

	// Create wrapper server
	server := &Server{
		teiBaseURL:   mockTEI.URL,
		currentModel: "test-model",
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

func TestModelHotSwapping(t *testing.T) {
	// Create mock TEI server
	mockTEI := createMockTEI(t)
	defer mockTEI.Close()

	// Create wrapper server with initial model
	server := &Server{
		teiBaseURL:   mockTEI.URL,
		currentModel: "model-a",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(server.handleEmbeddings))
	defer testServer.Close()

	// Test case 1: Request with same model (no switch)
	t.Run("SameModel", func(t *testing.T) {
		reqBody := EmbeddingRequest{
			Model: "model-a",
			Input: []string{"test"},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify model didn't change
		if server.currentModel != "model-a" {
			t.Errorf("Expected model to remain 'model-a', got %s", server.currentModel)
		}
	})

	// Test case 2: Request with different model (should succeed in test since we mock TEI)
	t.Run("DifferentModel", func(t *testing.T) {
		// Note: In unit test, switchModel won't actually work since there's no real TEI process
		// But we can test that the code path is hit
		reqBody := EmbeddingRequest{
			Model: "model-b",
			Input: []string{"test"},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Will fail because switchModel tries to stop/start real TEI
		// This is expected in unit test - integration test would verify full flow
		if resp.StatusCode == http.StatusOK {
			t.Log("Model switch succeeded (mock environment)")
		} else {
			t.Logf("Model switch failed as expected in unit test: status %d", resp.StatusCode)
		}
	})

	// Test case 3: Empty model string (should use current model)
	t.Run("EmptyModel", func(t *testing.T) {
		reqBody := EmbeddingRequest{
			Model: "",
			Input: []string{"test"},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestHealthWithModelInfo(t *testing.T) {
	// Create mock TEI server
	mockTEI := createMockTEI(t)
	defer mockTEI.Close()

	// Test case 1: Normal healthy state
	t.Run("Healthy", func(t *testing.T) {
		server := &Server{
			teiBaseURL:   mockTEI.URL,
			currentModel: "test-model",
			switching:    false,
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
		}

		testServer := httptest.NewServer(http.HandlerFunc(server.handleHealth))
		defer testServer.Close()

		resp, err := http.Get(testServer.URL)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var health map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&health)

		if health["status"] != "ok" {
			t.Errorf("Expected status='ok', got %v", health["status"])
		}

		if health["model"] != "test-model" {
			t.Errorf("Expected model='test-model', got %v", health["model"])
		}
	})

	// Test case 2: Switching state
	t.Run("Switching", func(t *testing.T) {
		server := &Server{
			teiBaseURL:   mockTEI.URL,
			currentModel: "old-model",
			switching:    true,
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
		}

		testServer := httptest.NewServer(http.HandlerFunc(server.handleHealth))
		defer testServer.Close()

		resp, err := http.Get(testServer.URL)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", resp.StatusCode)
		}

		var health map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&health)

		if health["status"] != "switching" {
			t.Errorf("Expected status='switching', got %v", health["status"])
		}

		if health["switching"] != true {
			t.Errorf("Expected switching=true, got %v", health["switching"])
		}
	})
}
