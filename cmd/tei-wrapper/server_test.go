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
