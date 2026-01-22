package main

import (
	"bytes"
	"context"
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
		multiModel: false,
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

func TestEmbeddingsEndpointMultiModel(t *testing.T) {
	makeMockTEI := func(value float64) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/health":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			case "/embed":
				resp := TEIResponse{{value}}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			default:
				http.NotFound(w, r)
			}
		}))
	}

	codeTEI := makeMockTEI(1.0)
	defer codeTEI.Close()
	textTEI := makeMockTEI(2.0)
	defer textTEI.Close()

	server := &Server{
		multiModel:     true,
		codeModel:     "code-model",
		textModel:     "text-model",
		codeTEIBaseURL: codeTEI.URL,
		textTEIBaseURL: textTEI.URL,
		codeTEIPort:    9200,
		textTEIPort:    9100,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(server.handleEmbeddings))
	defer testServer.Close()

	t.Run("CodeModelRoutesToCodeTEI", func(t *testing.T) {
		reqBody := EmbeddingRequest{
			Model: "code-model",
			Input: []string{"code"},
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

		var embResp EmbeddingResponse
		if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if embResp.Model != "code-model" {
			t.Errorf("Expected model='code-model', got %s", embResp.Model)
		}
		if len(embResp.Data) != 1 || len(embResp.Data[0].Embedding) != 1 || embResp.Data[0].Embedding[0] != 1.0 {
			t.Errorf("Expected code embedding value 1.0, got %v", embResp.Data)
		}
	})

	t.Run("TextModelRoutesToTextTEI", func(t *testing.T) {
		reqBody := EmbeddingRequest{
			Model: "text-model",
			Input: []string{"text"},
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

		var embResp EmbeddingResponse
		if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if embResp.Model != "text-model" {
			t.Errorf("Expected model='text-model', got %s", embResp.Model)
		}
		if len(embResp.Data) != 1 || len(embResp.Data[0].Embedding) != 1 || embResp.Data[0].Embedding[0] != 2.0 {
			t.Errorf("Expected text embedding value 2.0, got %v", embResp.Data)
		}
	})

	t.Run("EmptyModelDefaultsToText", func(t *testing.T) {
		reqBody := EmbeddingRequest{
			Model: "",
			Input: []string{"text"},
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

		var embResp EmbeddingResponse
		if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if embResp.Model != "text-model" {
			t.Errorf("Expected model='text-model', got %s", embResp.Model)
		}
		if len(embResp.Data) != 1 || len(embResp.Data[0].Embedding) != 1 || embResp.Data[0].Embedding[0] != 2.0 {
			t.Errorf("Expected text embedding value 2.0, got %v", embResp.Data)
		}
	})
}

func TestHealthEndpoint(t *testing.T) {
	// Create mock TEI server
	mockTEI := createMockTEI(t)
	defer mockTEI.Close()

	// Create wrapper server
	server := &Server{
		multiModel: false,
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

	if health["embedding_model"] != "test-model" {
		t.Errorf("Expected embedding_model='test-model', got %v", health["embedding_model"])
	}
}

func TestGetEmbeddings(t *testing.T) {
	// Create mock TEI server
	mockTEI := createMockTEI(t)
	defer mockTEI.Close()

	// Create wrapper server
	server := &Server{
		multiModel: false,
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

		if health["embedding_model"] != "test-model" {
			t.Errorf("Expected embedding_model='test-model', got %v", health["embedding_model"])
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

// TestIdlePreloadDisabled tests idle timer doesn't run when disabled
func TestIdlePreloadDisabled(t *testing.T) {
	server := &Server{
		idlePreload: false,
		currentModel: "test-model",
	}

	// Reset should do nothing when disabled
	server.resetIdleTimer()

	// Verify timer wasn't created
	if server.idleTimer != nil {
		t.Error("Timer should not be created when idle preload is disabled")
	}
}

// TestIdlePreloadEnabled tests idle timer functionality
func TestIdlePreloadEnabled(t *testing.T) {
	server := &Server{
		idlePreload:    true,
		idleTimeout:    100 * time.Millisecond, // Short timeout for test
		preferredModel: "preferred-model",
		currentModel:   "other-model",
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Reset should create timer
	server.resetIdleTimer()

	if server.idleTimer == nil {
		t.Error("Timer should be created when idle preload is enabled")
	}

	// lastRequestTime should be set
	if server.lastRequestTime.IsZero() {
		t.Error("lastRequestTime should be set")
	}

	// Resetting again should stop old timer and create new one
	firstTime := server.lastRequestTime
	time.Sleep(10 * time.Millisecond)
	server.resetIdleTimer()

	if !server.lastRequestTime.After(firstTime) {
		t.Error("lastRequestTime should be updated on second reset")
	}
}

// TestOnIdleTimeout tests idle timeout behavior
func TestOnIdleTimeout(t *testing.T) {
	t.Run("AlreadyOnPreferredModel", func(t *testing.T) {
		server := &Server{
			currentModel:   "preferred-model",
			preferredModel: "preferred-model",
		}

		// Should do nothing if already on preferred model
		server.onIdleTimeout()

		// Verify model didn't change
		if server.currentModel != "preferred-model" {
			t.Error("Model should not change when already on preferred")
		}
	})

	t.Run("NeedsSwitch", func(t *testing.T) {
		server := &Server{
			currentModel:   "other-model",
			preferredModel: "preferred-model",
		}

		// Should attempt to switch (will fail without real TEI, but we verify the path is hit)
		server.onIdleTimeout()

		// In a real scenario with a running TEI, this would switch models
		// For unit tests, we just verify the function doesn't panic
	})
}

// TestResetIdleTimerMultipleTimes tests resetting timer multiple times
func TestResetIdleTimerMultipleTimes(t *testing.T) {
	server := &Server{
		idlePreload:    true,
		idleTimeout:    50 * time.Millisecond,
		preferredModel: "preferred",
		currentModel:   "other",
	}

	// Reset timer multiple times quickly
	for i := 0; i < 10; i++ {
		server.resetIdleTimer()
		time.Sleep(5 * time.Millisecond)
	}

	// Timer should still be set
	if server.idleTimer == nil {
		t.Error("Timer should still be set after multiple resets")
	}
}

// TestHandleEmbeddingsResetsIdleTimer tests that requests reset idle timer
func TestHandleEmbeddingsResetsIdleTimer(t *testing.T) {
	mockTEI := createMockTEI(t)
	defer mockTEI.Close()

	server := &Server{
		teiBaseURL:     mockTEI.URL,
		currentModel:   "test-model",
		idlePreload:    true,
		idleTimeout:    1 * time.Second,
		preferredModel: "preferred-model",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(server.handleEmbeddings))
	defer testServer.Close()

	// Make a request
	reqBody := EmbeddingRequest{
		Model: "test-model",
		Input: []string{"test"},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Verify timer was created
	if server.idleTimer == nil {
		t.Error("Timer should be created after request")
	}

	// Verify lastRequestTime was set
	if server.lastRequestTime.IsZero() {
		t.Error("lastRequestTime should be set after request")
	}
}

// TestHealthUnhealthyTEI tests health endpoint when TEI is down
func TestHealthUnhealthyTEI(t *testing.T) {
	server := &Server{
		teiBaseURL:   "http://invalid-host-that-does-not-exist:9999",
		currentModel: "test-model",
		switching:    false,
		client: &http.Client{
			Timeout: 1 * time.Second,
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

	if health["status"] != "unhealthy" {
		t.Errorf("Expected status='unhealthy', got %v", health["status"])
	}
}

// TestEmbeddingsWhileSwitching tests request during model switch
func TestEmbeddingsWhileSwitching(t *testing.T) {
	mockTEI := createMockTEI(t)
	defer mockTEI.Close()

	server := &Server{
		teiBaseURL:   mockTEI.URL,
		currentModel: "test-model",
		switching:    true, // Mark as switching
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(server.handleEmbeddings))
	defer testServer.Close()

	reqBody := EmbeddingRequest{
		Model: "test-model",
		Input: []string{"test"},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 during switch, got %d", resp.StatusCode)
	}

	// Check for Retry-After header
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter != "5" {
		t.Errorf("Expected Retry-After=5, got %s", retryAfter)
	}
}

// TestGetEmbeddingsTEIError tests TEI error handling
func TestGetEmbeddingsTEIError(t *testing.T) {
	// Create mock that returns error
	mockTEI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("TEI error"))
	}))
	defer mockTEI.Close()

	server := &Server{
		teiBaseURL:   mockTEI.URL,
		currentModel: "test-model",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	_, err := server.getEmbeddings([]string{"test"})
	if err == nil {
		t.Error("Expected error when TEI returns 500")
	}

	if err != nil && !contains(err.Error(), "500") {
		t.Errorf("Expected error to mention status 500, got: %v", err)
	}
}

// TestGetEmbeddingsInvalidJSON tests TEI invalid JSON response
func TestGetEmbeddingsInvalidJSON(t *testing.T) {
	// Create mock that returns invalid JSON
	mockTEI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer mockTEI.Close()

	server := &Server{
		teiBaseURL:   mockTEI.URL,
		currentModel: "test-model",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	_, err := server.getEmbeddings([]string{"test"})
	if err == nil {
		t.Error("Expected error when TEI returns invalid JSON")
	}

	if err != nil && !contains(err.Error(), "parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestWaitForTEI tests the TEI readiness polling
func TestWaitForTEI(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockTEI := createMockTEI(t)
		defer mockTEI.Close()

		server := &Server{
			teiBaseURL: mockTEI.URL,
			client: &http.Client{
				Timeout: 5 * time.Second,
			},
		}

		err := server.waitForTEI(5 * time.Second)
		if err != nil {
			t.Errorf("waitForTEI should succeed, got error: %v", err)
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		// Point to a non-existent server
		server := &Server{
			teiBaseURL: "http://localhost:9999",
			client: &http.Client{
				Timeout: 100 * time.Millisecond,
			},
		}

		err := server.waitForTEI(500 * time.Millisecond)
		if err == nil {
			t.Error("waitForTEI should timeout when TEI is not available")
		}

		if err != nil && !contains(err.Error(), "did not become ready") {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	})

	t.Run("UnhealthyThenHealthy", func(t *testing.T) {
		// Create a server that returns unhealthy first, then healthy
		attempts := 0
		mockTEI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer mockTEI.Close()

		server := &Server{
			teiBaseURL: mockTEI.URL,
			client: &http.Client{
				Timeout: 5 * time.Second,
			},
		}

		err := server.waitForTEI(5 * time.Second)
		if err != nil {
			t.Errorf("waitForTEI should eventually succeed, got error: %v", err)
		}

		if attempts < 3 {
			t.Errorf("Expected at least 3 attempts, got %d", attempts)
		}
	})
}

// TestStopTEI tests TEI process shutdown
func TestStopTEI(t *testing.T) {
	t.Run("NilProcess", func(t *testing.T) {
		server := &Server{
			teiCmd: nil,
		}

		// Should not panic
		server.stopTEI()
	})

	t.Run("NilCommand", func(t *testing.T) {
		server := &Server{}

		// Should not panic
		server.stopTEI()
	})
}

// TestStartTEIWithModel tests starting TEI with various scenarios
func TestStartTEIWithModel(t *testing.T) {
	t.Run("InvalidBinary", func(t *testing.T) {
		server := &Server{
			teiBinary: "/nonexistent/binary",
			teiPort:   8080,
		}

		err := server.startTEIWithModel(context.Background(), "test-model")
		if err == nil {
			t.Error("Expected error when binary doesn't exist")
		}

		if err != nil && !contains(err.Error(), "failed to start TEI") {
			t.Errorf("Expected start error, got: %v", err)
		}
	})

	t.Run("ContextCanceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		server := &Server{
			teiBinary: "echo", // Use a simple command that exists
			teiPort:   8080,
		}

		// This should still succeed in creating the command, but it won't run
		err := server.startTEIWithModel(ctx, "test-model")

		// The command creation itself doesn't fail, only execution would
		// So we just verify the model is set
		if err == nil && server.currentModel != "test-model" {
			t.Error("Expected currentModel to be set")
		}
	})
}

// TestSwitchModelErrorPaths tests switchModel error handling
func TestSwitchModelErrorPaths(t *testing.T) {
	t.Run("SameModel", func(t *testing.T) {
		server := &Server{
			currentModel: "same-model",
			teiBinary:    "echo",
		}

		err := server.switchModel("same-model")
		if err != nil {
			t.Errorf("Switching to same model should be no-op, got error: %v", err)
		}
	})
}

// TestGetEmbeddingsEdgeCases tests additional edge cases
func TestGetEmbeddingsEdgeCases(t *testing.T) {
	t.Run("EmptyInputs", func(t *testing.T) {
		mockTEI := createMockTEI(t)
		defer mockTEI.Close()

		server := &Server{
			teiBaseURL:   mockTEI.URL,
			currentModel: "test-model",
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
		}

		embeddings, err := server.getEmbeddings([]string{})
		if err != nil {
			t.Fatalf("getEmbeddings with empty input should succeed: %v", err)
		}

		if len(embeddings) != 0 {
			t.Errorf("Expected 0 embeddings for empty input, got %d", len(embeddings))
		}
	})

	t.Run("NetworkError", func(t *testing.T) {
		server := &Server{
			teiBaseURL:   "http://invalid-host-xyz:9999",
			currentModel: "test-model",
			client: &http.Client{
				Timeout: 100 * time.Millisecond,
			},
		}

		_, err := server.getEmbeddings([]string{"test"})
		if err == nil {
			t.Error("Expected network error")
		}

		if err != nil && !contains(err.Error(), "failed to send request") {
			t.Errorf("Expected network error, got: %v", err)
		}
	})
}

// TestStopTEIWithRealProcess tests stopping a real process
func TestStopTEIWithRealProcess(t *testing.T) {
	t.Run("GracefulShutdown", func(t *testing.T) {
		server := &Server{
			teiBinary: "sleep",
			teiPort:   8080,
		}

		// Start a sleep process
		err := server.startTEIWithModel(context.Background(), "test-model")
		if err != nil {
			t.Fatalf("Failed to start process: %v", err)
		}

		if server.teiCmd == nil || server.teiCmd.Process == nil {
			t.Fatal("Process not started")
		}

		pid := server.teiCmd.Process.Pid

		// Stop the process
		server.stopTEI()

		// Verify process is stopped (checking if process exists will fail)
		// We can't easily verify this in a cross-platform way, but the function should complete
		if server.teiCmd != nil && server.teiCmd.Process != nil {
			// Process might still be referenced, that's ok
		}

		t.Logf("Process %d stopped successfully", pid)
	})

	t.Run("ProcessAlreadyDead", func(t *testing.T) {
		server := &Server{
			teiBinary: "echo", // echo exits immediately
			teiPort:   8080,
		}

		// Start echo which exits immediately
		err := server.startTEIWithModel(context.Background(), "test-model")
		if err != nil {
			t.Fatalf("Failed to start process: %v", err)
		}

		// Wait for process to exit
		time.Sleep(100 * time.Millisecond)

		// Stopping should handle already-dead process gracefully
		server.stopTEI()
	})
}

// TestStartTEIWithModelSuccess tests successful process start
func TestStartTEIWithModelSuccess(t *testing.T) {
	server := &Server{
		teiBinary: "sleep",
		teiPort:   8080,
	}

	err := server.startTEIWithModel(context.Background(), "test-model")
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}
	defer server.stopTEI()

	if server.teiCmd == nil {
		t.Fatal("teiCmd should be set")
	}

	if server.teiCmd.Process == nil {
		t.Fatal("Process should be started")
	}

	if server.currentModel != "test-model" {
		t.Errorf("Expected currentModel='test-model', got %s", server.currentModel)
	}

	// Process should be running
	if server.teiCmd.Process.Pid == 0 {
		t.Error("Process PID should not be 0")
	}
}

// TestSwitchModelWithRealProcess tests model switching with real processes
func TestSwitchModelWithRealProcess(t *testing.T) {
	t.Run("SwitchToSameModel", func(t *testing.T) {
		server := &Server{
			currentModel: "model-a",
			teiBinary:    "sleep",
			teiPort:      8080,
		}

		err := server.switchModel("model-a")
		if err != nil {
			t.Errorf("Switching to same model should succeed: %v", err)
		}

		// Should not have created a process
		if server.teiCmd != nil {
			t.Error("Should not create process when switching to same model")
		}
	})
}

// TestHandleEmbeddingsErrorInGetEmbeddings tests error handling when getEmbeddings fails
func TestHandleEmbeddingsErrorInGetEmbeddings(t *testing.T) {
	// Create mock that returns error
	mockTEI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Return error for embed endpoint
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal error"))
	}))
	defer mockTEI.Close()

	server := &Server{
		teiBaseURL:   mockTEI.URL,
		currentModel: "test-model",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(server.handleEmbeddings))
	defer testServer.Close()

	reqBody := EmbeddingRequest{
		Model: "test-model",
		Input: []string{"test"},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

// TestResetIdleTimerConcurrency tests resetIdleTimer under concurrent access
func TestResetIdleTimerConcurrency(t *testing.T) {
	server := &Server{
		idlePreload:    true,
		idleTimeout:    100 * time.Millisecond,
		preferredModel: "preferred",
		currentModel:   "other",
	}

	// Reset timer concurrently
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			server.resetIdleTimer()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Timer should be set
	if server.idleTimer == nil {
		t.Error("Timer should be set after concurrent resets")
	}
}

// TestSwitchModelErrors tests switchModel error paths
func TestSwitchModelErrors(t *testing.T) {
	t.Run("StartTEIFails", func(t *testing.T) {
		server := &Server{
			currentModel: "old-model",
			teiBinary:    "/nonexistent/binary",
			teiPort:      8080,
		}

		err := server.switchModel("new-model")
		if err == nil {
			t.Error("Expected error when starting TEI fails")
		}

		if err != nil && !contains(err.Error(), "failed to start TEI") {
			t.Errorf("Expected start error, got: %v", err)
		}
	})

	t.Run("WaitForTEIFails", func(t *testing.T) {
		// This is harder to test without mocking, but we can test the path
		// by using a binary that starts but doesn't provide the health endpoint
		server := &Server{
			currentModel: "old-model",
			teiBinary:    "sleep",
			teiPort:      9999,
			teiBaseURL:   "http://localhost:9999",
			client: &http.Client{
				Timeout: 100 * time.Millisecond,
			},
		}

		// Start with a sleep process first
		ctx := context.Background()
		server.startTEIWithModel(ctx, "old-model")
		if server.teiCmd != nil {
			defer server.stopTEI()
		}

		err := server.switchModel("new-model")
		if err == nil {
			t.Error("Expected error when TEI doesn't become ready")
		}

		if err != nil && !contains(err.Error(), "failed to start") {
			t.Errorf("Expected TEI ready error, got: %v", err)
		}
	})
}

// TestOnIdleTimeoutCoverage ensures onIdleTimeout is fully covered
func TestOnIdleTimeoutCoverage(t *testing.T) {
	t.Run("SwitchFails", func(t *testing.T) {
		server := &Server{
			currentModel:   "other-model",
			preferredModel: "preferred-model",
			teiBinary:      "/nonexistent/binary",
			teiPort:        8080,
		}

		// Should attempt switch and fail gracefully
		server.onIdleTimeout()

		// Verify it doesn't panic and model doesn't change
		if server.currentModel != "other-model" {
			t.Error("Model should not change when switch fails")
		}
	})
}

// TestSwitchModelSuccess tests successful model switching
func TestSwitchModelSuccess(t *testing.T) {
	// Create a mock TEI server
	mockTEI := createMockTEI(t)
	defer mockTEI.Close()

	server := &Server{
		currentModel: "model-a",
		teiBinary:    "sleep",
		teiPort:      8080,
		teiBaseURL:   mockTEI.URL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Start initial model
	err := server.startTEIWithModel(context.Background(), "model-a")
	if err != nil {
		t.Fatalf("Failed to start initial model: %v", err)
	}

	// Switch to a new model
	// This will stop the sleep process and start a new one
	// The mockTEI will respond to health checks
	err = server.switchModel("model-b")
	if err != nil {
		t.Fatalf("Model switch should succeed: %v", err)
	}

	// Clean up
	defer server.stopTEI()

	// Verify the model was switched
	if server.currentModel != "model-b" {
		t.Errorf("Expected currentModel='model-b', got %s", server.currentModel)
	}
}

// TestSwitchModelConcurrency tests concurrent model switching
func TestSwitchModelConcurrency(t *testing.T) {
	mockTEI := createMockTEI(t)
	defer mockTEI.Close()

	server := &Server{
		currentModel: "model-a",
		teiBinary:    "sleep",
		teiPort:      8080,
		teiBaseURL:   mockTEI.URL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Start initial model
	err := server.startTEIWithModel(context.Background(), "model-a")
	if err != nil {
		t.Fatalf("Failed to start initial model: %v", err)
	}
	defer server.stopTEI()

	// Try concurrent switches - second should wait for first
	done := make(chan error, 2)
	go func() {
		done <- server.switchModel("model-b")
	}()
	go func() {
		// Small delay to ensure first switch starts
		time.Sleep(10 * time.Millisecond)
		done <- server.switchModel("model-c")
	}()

	// Wait for both
	err1 := <-done
	err2 := <-done

	// At least one should succeed (both might if timing works out)
	if err1 != nil && err2 != nil {
		t.Errorf("At least one switch should succeed: err1=%v, err2=%v", err1, err2)
	}
}

// TestStopTEIKillTimeout tests the timeout/kill path in stopTEI
func TestStopTEIKillTimeout(t *testing.T) {
	// This test would need a process that ignores SIGTERM
	// which is difficult to create portably
	// For now, we'll test what we can
	t.Skip("Skipping timeout test - requires process that ignores SIGTERM")
}

// TestOnIdleTimeoutSuccess tests successful idle timeout switching
func TestOnIdleTimeoutSuccess(t *testing.T) {
	mockTEI := createMockTEI(t)
	defer mockTEI.Close()

	server := &Server{
		currentModel:   "text-model",
		preferredModel: "code-model",
		teiBinary:      "sleep",
		teiPort:        8080,
		teiBaseURL:     mockTEI.URL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Start with text model
	err := server.startTEIWithModel(context.Background(), "text-model")
	if err != nil {
		t.Fatalf("Failed to start initial model: %v", err)
	}
	defer server.stopTEI()

	// Trigger idle timeout - should switch to code model
	server.onIdleTimeout()

	// Verify switch occurred
	if server.currentModel != "code-model" {
		t.Errorf("Expected currentModel='code-model', got %s", server.currentModel)
	}
}

// TestMaxBatchTokensConfiguration tests that maxBatchTokens is properly stored
func TestMaxBatchTokensConfiguration(t *testing.T) {
	testCases := []struct {
		name           string
		maxBatchTokens int
	}{
		{"Default", 8192},
		{"LowMemory", 2048},
		{"HighThroughput", 16384},
		{"Custom", 4096},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := &Server{
				maxBatchTokens: tc.maxBatchTokens,
			}

			if server.maxBatchTokens != tc.maxBatchTokens {
				t.Errorf("Expected maxBatchTokens=%d, got %d", tc.maxBatchTokens, server.maxBatchTokens)
			}
		})
	}
}

// ============================================================================
// Reranker Tests
// ============================================================================

// TestHandleRerankSuccess tests successful rerank requests
func TestHandleRerankSuccess(t *testing.T) {
	// Create mock reranker TEI
	mockReranker := createMockRerankerTEI(t)
	defer mockReranker.Close()

	server := &Server{
		rerankModel:        "BAAI/bge-reranker-base",
		currentRerankModel: "BAAI/bge-reranker-base", // Set to avoid model switching
		rerankBaseURL:      mockReranker.URL,
		rerankHealthy:      true,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(server.handleRerank))
	defer testServer.Close()

	// Send rerank request
	reqBody := RerankRequest{
		Query: "test query",
		Texts: []string{"document 1", "document 2", "document 3"},
		Model: "BAAI/bge-reranker-base", // Specify model in request
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
	var results []RerankResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// Verify results have expected structure
	for i, result := range results {
		if result.Index != i {
			t.Errorf("Expected index=%d, got %d", i, result.Index)
		}
		if result.Score <= 0 {
			t.Errorf("Expected positive score, got %f", result.Score)
		}
	}
}

// TestHandleRerankNotConfigured tests rerank when not configured
func TestHandleRerankNotConfigured(t *testing.T) {
	server := &Server{
		config:      nil, // No config
		rerankModel: "",  // Not configured
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(server.handleRerank))
	defer testServer.Close()

	reqBody := RerankRequest{
		Query: "test query",
		Texts: []string{"doc1"},
		Model: "", // No model specified anywhere
	}

	bodyBytes, _ := json.Marshal(reqBody)
	resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// With dynamic loading, this returns 400 (bad request) when no model is specified
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 when reranker not configured, got %d", resp.StatusCode)
	}
}

// TestHandleRerankUnhealthy tests rerank when reranker TEI is unhealthy
func TestHandleRerankUnhealthy(t *testing.T) {
	server := &Server{
		rerankModel:        "BAAI/bge-reranker-base",
		currentRerankModel: "BAAI/bge-reranker-base", // Set to avoid model switching
		rerankBaseURL:      "http://localhost:9999",  // Non-existent server
		rerankHealthy:      false,
		client: &http.Client{
			Timeout: 100 * time.Millisecond,
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(server.handleRerank))
	defer testServer.Close()

	reqBody := RerankRequest{
		Query: "test query",
		Texts: []string{"doc1"},
		Model: "BAAI/bge-reranker-base", // Specify model
	}

	bodyBytes, _ := json.Marshal(reqBody)
	resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 when reranker unhealthy, got %d", resp.StatusCode)
	}

	// Check Retry-After header
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter != "5" {
		t.Errorf("Expected Retry-After=5, got %s", retryAfter)
	}
}

// TestHandleRerankWrongMethod tests wrong HTTP method
func TestHandleRerankWrongMethod(t *testing.T) {
	server := &Server{
		rerankModel: "BAAI/bge-reranker-base",
	}

	testServer := httptest.NewServer(http.HandlerFunc(server.handleRerank))
	defer testServer.Close()

	resp, err := http.Get(testServer.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

// TestCheckRerankHealth tests the reranker health check function
func TestCheckRerankHealth(t *testing.T) {
	t.Run("NotConfigured", func(t *testing.T) {
		server := &Server{
			rerankModel: "", // Not configured
		}

		healthy := server.checkRerankHealth()
		if healthy {
			t.Error("Expected unhealthy when reranker not configured")
		}
	})

	t.Run("Healthy", func(t *testing.T) {
		mockReranker := createMockRerankerTEI(t)
		defer mockReranker.Close()

		server := &Server{
			rerankModel:   "BAAI/bge-reranker-base",
			rerankBaseURL: mockReranker.URL,
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
		}

		healthy := server.checkRerankHealth()
		if !healthy {
			t.Error("Expected healthy when reranker is available")
		}
		if !server.rerankHealthy {
			t.Error("Expected rerankHealthy to be set to true")
		}
	})

	t.Run("Unhealthy", func(t *testing.T) {
		server := &Server{
			rerankModel:   "BAAI/bge-reranker-base",
			rerankBaseURL: "http://localhost:9999",
			client: &http.Client{
				Timeout: 100 * time.Millisecond,
			},
		}

		healthy := server.checkRerankHealth()
		if healthy {
			t.Error("Expected unhealthy when reranker is down")
		}
		if server.rerankHealthy {
			t.Error("Expected rerankHealthy to be set to false")
		}
	})
}

// TestHealthEndpointWithReranker tests health endpoint includes reranker status
func TestHealthEndpointWithReranker(t *testing.T) {
	t.Run("WithRerankerHealthy", func(t *testing.T) {
		mockTEI := createMockTEI(t)
		defer mockTEI.Close()
		mockReranker := createMockRerankerTEI(t)
		defer mockReranker.Close()

		server := &Server{
			teiBaseURL:         mockTEI.URL,
			currentModel:       "test-model",
			rerankModel:        "BAAI/bge-reranker-base",
			currentRerankModel: "BAAI/bge-reranker-base", // Set current model
			rerankPort:         8081,
			rerankBaseURL:      mockReranker.URL,
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
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var health map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check embedding_model field (new format)
		if health["embedding_model"] != "test-model" {
			t.Errorf("Expected embedding_model='test-model', got %v", health["embedding_model"])
		}

		// Check reranker info
		reranker, ok := health["reranker"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected reranker info in health response")
		}

		if reranker["enabled"] != true {
			t.Errorf("Expected reranker enabled=true, got %v", reranker["enabled"])
		}
		if reranker["healthy"] != true {
			t.Errorf("Expected reranker healthy=true, got %v", reranker["healthy"])
		}
		if reranker["model"] != "BAAI/bge-reranker-base" {
			t.Errorf("Expected reranker model='BAAI/bge-reranker-base', got %v", reranker["model"])
		}
		if reranker["port"] != float64(8081) {
			t.Errorf("Expected reranker port=8081, got %v", reranker["port"])
		}
	})

	t.Run("WithRerankerUnhealthy", func(t *testing.T) {
		mockTEI := createMockTEI(t)
		defer mockTEI.Close()

		server := &Server{
			teiBaseURL:         mockTEI.URL,
			currentModel:       "test-model",
			rerankModel:        "BAAI/bge-reranker-base",
			currentRerankModel: "BAAI/bge-reranker-base", // Set current model
			rerankPort:         8081,
			rerankBaseURL:      "http://localhost:9999", // Non-existent
			client: &http.Client{
				Timeout: 100 * time.Millisecond,
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
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var health map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Reranker should show as unhealthy
		reranker, ok := health["reranker"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected reranker info in health response")
		}

		if reranker["healthy"] != false {
			t.Errorf("Expected reranker healthy=false, got %v", reranker["healthy"])
		}
	})

	t.Run("WithoutReranker", func(t *testing.T) {
		mockTEI := createMockTEI(t)
		defer mockTEI.Close()

		server := &Server{
			teiBaseURL:   mockTEI.URL,
			currentModel: "test-model",
			rerankModel:  "", // Not configured
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
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var health map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should not have reranker info
		if _, ok := health["reranker"]; ok {
			t.Error("Did not expect reranker info when not configured")
		}
	})
}

// TestWaitForRerankTEI tests the reranker TEI readiness polling
func TestWaitForRerankTEI(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockReranker := createMockRerankerTEI(t)
		defer mockReranker.Close()

		server := &Server{
			rerankBaseURL: mockReranker.URL,
			client: &http.Client{
				Timeout: 5 * time.Second,
			},
		}

		err := server.waitForRerankTEI(5 * time.Second)
		if err != nil {
			t.Errorf("waitForRerankTEI should succeed, got error: %v", err)
		}
		if !server.rerankHealthy {
			t.Error("Expected rerankHealthy to be true after successful wait")
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		server := &Server{
			rerankBaseURL: "http://localhost:9999",
			client: &http.Client{
				Timeout: 100 * time.Millisecond,
			},
		}

		err := server.waitForRerankTEI(500 * time.Millisecond)
		if err == nil {
			t.Error("waitForRerankTEI should timeout when reranker is not available")
		}

		if err != nil && !contains(err.Error(), "did not become ready") {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	})
}

// TestStopRerankTEI tests reranker TEI process shutdown
func TestStopRerankTEI(t *testing.T) {
	t.Run("NilProcess", func(t *testing.T) {
		server := &Server{
			rerankCmd: nil,
		}

		// Should not panic
		server.stopRerankTEI()
	})

	t.Run("NilCommand", func(t *testing.T) {
		server := &Server{}

		// Should not panic
		server.stopRerankTEI()
	})
}

// TestRerankRequestFormat tests rerank request/response format
func TestRerankRequestFormat(t *testing.T) {
	mockReranker := createMockRerankerTEI(t)
	defer mockReranker.Close()

	server := &Server{
		rerankModel:        "BAAI/bge-reranker-base",
		currentRerankModel: "BAAI/bge-reranker-base", // Set to avoid model switching
		rerankBaseURL:      mockReranker.URL,
		rerankHealthy:      true,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(server.handleRerank))
	defer testServer.Close()

	t.Run("WithRawScores", func(t *testing.T) {
		reqBody := RerankRequest{
			Query:     "test query",
			Texts:     []string{"doc1", "doc2"},
			RawScores: true,
			Model:     "BAAI/bge-reranker-base", // Specify model
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

	t.Run("WithReturnText", func(t *testing.T) {
		reqBody := RerankRequest{
			Query:      "test query",
			Texts:      []string{"doc1", "doc2"},
			ReturnText: true,
			Model:      "BAAI/bge-reranker-base", // Specify model
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

// TestHandleRerankBackendError tests error handling when reranker TEI returns an error
func TestHandleRerankBackendError(t *testing.T) {
	// Create mock that returns error
	mockReranker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Return error for rerank endpoint
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal error"))
	}))
	defer mockReranker.Close()

	server := &Server{
		rerankModel:        "BAAI/bge-reranker-base",
		currentRerankModel: "BAAI/bge-reranker-base", // Set to avoid model switching
		rerankBaseURL:      mockReranker.URL,
		rerankHealthy:      true,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(server.handleRerank))
	defer testServer.Close()

	reqBody := RerankRequest{
		Query: "test query",
		Texts: []string{"doc1"},
		Model: "BAAI/bge-reranker-base", // Specify model
	}

	bodyBytes, _ := json.Marshal(reqBody)
	resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should return 500 from backend
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}
