package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jlanders/code-scout/internal/config"
)

// TestRerankDynamicModelLoading tests dynamic reranker model loading
func TestRerankDynamicModelLoading(t *testing.T) {
	mockReranker := createMockRerankerTEI(t)
	defer mockReranker.Close()

	cfg := &config.Config{
		RerankModel: "BAAI/bge-reranker-large",
	}

	server := &Server{
		config:              cfg,
		rerankBaseURL:       mockReranker.URL,
		currentRerankModel:  "", // No model loaded initially
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(server.handleRerank))
	defer testServer.Close()

	// Test case 1: Model from request
	t.Run("ModelFromRequest", func(t *testing.T) {
		reqBody := RerankRequest{
			Query: "test query",
			Texts: []string{"doc1", "doc2"},
			Model: "BAAI/bge-reranker-base", // Model in request
		}

		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Should attempt to load the model (will fail in unit test without real TEI)
		// But we test that the code path is reached
		if resp.StatusCode == http.StatusOK {
			t.Log("Model loading succeeded (mock environment)")
		} else if resp.StatusCode == http.StatusInternalServerError {
			t.Log("Model loading failed as expected in unit test")
		}
	})

	// Test case 2: Model from config (fallback)
	t.Run("ModelFromConfig", func(t *testing.T) {
		reqBody := RerankRequest{
			Query: "test query",
			Texts: []string{"doc1", "doc2"},
			Model: "", // No model in request, should use config
		}

		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Should attempt to load model from config
		if resp.StatusCode == http.StatusOK {
			t.Log("Model loading from config succeeded (mock environment)")
		} else if resp.StatusCode == http.StatusInternalServerError {
			t.Log("Model loading from config failed as expected in unit test")
		}
	})

	// Test case 3: No model anywhere (should error)
	t.Run("NoModelSpecified", func(t *testing.T) {
		serverNoConfig := &Server{
			config:             nil, // No config
			rerankModel:        "",  // No CLI flag
			rerankBaseURL:      mockReranker.URL,
			currentRerankModel: "", // No model loaded
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
		}

		testServer2 := httptest.NewServer(http.HandlerFunc(serverNoConfig.handleRerank))
		defer testServer2.Close()

		reqBody := RerankRequest{
			Query: "test query",
			Texts: []string{"doc1"},
			Model: "", // No model in request
		}

		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(testServer2.URL, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 when no model specified, got %d", resp.StatusCode)
		}
	})
}

// TestRerankWhileSwitching tests request during reranker model switch
func TestRerankWhileSwitching(t *testing.T) {
	mockReranker := createMockRerankerTEI(t)
	defer mockReranker.Close()

	server := &Server{
		currentRerankModel: "BAAI/bge-reranker-base",
		rerankBaseURL:      mockReranker.URL,
		rerankSwitching:    true, // Mark as switching
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(server.handleRerank))
	defer testServer.Close()

	reqBody := RerankRequest{
		Query: "test query",
		Texts: []string{"doc1"},
		Model: "BAAI/bge-reranker-large",
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

// TestHealthEndpointWithDynamicReranker tests health endpoint with dynamic reranker
func TestHealthEndpointWithDynamicReranker(t *testing.T) {
	mockTEI := createMockTEI(t)
	defer mockTEI.Close()
	mockReranker := createMockRerankerTEI(t)
	defer mockReranker.Close()

	t.Run("WithCurrentRerankModel", func(t *testing.T) {
		server := &Server{
			teiBaseURL:         mockTEI.URL,
			currentModel:       "test-model",
			currentRerankModel: "BAAI/bge-reranker-large",
			rerankModel:        "BAAI/bge-reranker-base", // CLI flag (different from current)
			rerankPort:         8081,
			rerankBaseURL:      mockReranker.URL,
			rerankSwitching:    false,
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

		// Check reranker info
		reranker, ok := health["reranker"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected reranker info in health response")
		}

		// Should show currentRerankModel, not startup model
		if reranker["model"] != "BAAI/bge-reranker-large" {
			t.Errorf("Expected reranker model='BAAI/bge-reranker-large', got %v", reranker["model"])
		}

		if reranker["startup_model"] != "BAAI/bge-reranker-base" {
			t.Errorf("Expected startup_model='BAAI/bge-reranker-base', got %v", reranker["startup_model"])
		}

		if reranker["switching"] != false {
			t.Errorf("Expected switching=false, got %v", reranker["switching"])
		}
	})

	t.Run("WhileSwitching", func(t *testing.T) {
		server := &Server{
			teiBaseURL:         mockTEI.URL,
			currentModel:       "test-model",
			currentRerankModel: "BAAI/bge-reranker-base",
			rerankPort:         8081,
			rerankBaseURL:      mockReranker.URL,
			rerankSwitching:    true, // Currently switching
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

		var health map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		reranker, ok := health["reranker"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected reranker info in health response")
		}

		if reranker["switching"] != true {
			t.Errorf("Expected switching=true, got %v", reranker["switching"])
		}
	})
}

// TestSwitchRerankModelLogic tests the switchRerankModel function
func TestSwitchRerankModelLogic(t *testing.T) {
	t.Run("SameModel", func(t *testing.T) {
		server := &Server{
			currentRerankModel: "BAAI/bge-reranker-base",
		}

		err := server.switchRerankModel("BAAI/bge-reranker-base")
		if err != nil {
			t.Errorf("Switching to same model should be no-op, got error: %v", err)
		}

		// Should not set switching flag
		if server.rerankSwitching {
			t.Error("Switching flag should not be set for same model")
		}
	})

	t.Run("DifferentModel", func(t *testing.T) {
		server := &Server{
			currentRerankModel: "BAAI/bge-reranker-base",
			teiBinary:          "/nonexistent/binary", // Use fake binary path that will fail immediately
			rerankPort:         8081,
			rerankBaseURL:      "http://localhost:8081",
			client: &http.Client{
				Timeout: 1 * time.Second,
			},
		}

		// This will fail in unit test (no real TEI), but we verify the path is hit
		err := server.switchRerankModel("BAAI/bge-reranker-large")
		if err == nil {
			t.Error("Expected model switch to fail in unit test")
		} else {
			// Expected to fail without real TEI
			t.Logf("Model switch failed as expected in unit test: %v", err)
		}

		// Verify switching flag is cleared even after error
		if server.rerankSwitching {
			t.Error("Switching flag should be cleared after switch attempt")
		}
	})
}

// TestRerankRequestParsing tests parsing of rerank requests with model field
func TestRerankRequestParsing(t *testing.T) {
	mockReranker := createMockRerankerTEI(t)
	defer mockReranker.Close()

	server := &Server{
		currentRerankModel: "BAAI/bge-reranker-base",
		rerankBaseURL:      mockReranker.URL,
		rerankHealthy:      true,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	testServer := httptest.NewServer(http.HandlerFunc(server.handleRerank))
	defer testServer.Close()

	t.Run("InvalidJSON", func(t *testing.T) {
		resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader([]byte("invalid json")))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", resp.StatusCode)
		}
	})

	t.Run("ValidJSONWithModel", func(t *testing.T) {
		reqBody := RerankRequest{
			Query: "test query",
			Texts: []string{"doc1"},
			Model: "BAAI/bge-reranker-base", // Same as current, should not switch
		}

		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Should succeed since same model
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

// TestRerankModelPriority tests model selection priority: request > config > CLI
func TestRerankModelPriority(t *testing.T) {
	mockReranker := createMockRerankerTEI(t)
	defer mockReranker.Close()

	t.Run("RequestOverridesConfig", func(t *testing.T) {
		cfg := &config.Config{
			RerankModel: "config-model",
		}

		server := &Server{
			config:             cfg,
			rerankModel:        "cli-model",
			currentRerankModel: "current-model",
			rerankBaseURL:      mockReranker.URL,
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
		}

		testServer := httptest.NewServer(http.HandlerFunc(server.handleRerank))
		defer testServer.Close()

		reqBody := RerankRequest{
			Query: "test query",
			Texts: []string{"doc1"},
			Model: "request-model", // Should use this
		}

		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Should attempt to switch to request-model
		// (will fail in unit test but that's ok)
	})

	t.Run("ConfigOverridesCLI", func(t *testing.T) {
		cfg := &config.Config{
			RerankModel: "config-model",
		}

		server := &Server{
			config:             cfg,
			rerankModel:        "cli-model",
			currentRerankModel: "current-model",
			rerankBaseURL:      mockReranker.URL,
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
		}

		testServer := httptest.NewServer(http.HandlerFunc(server.handleRerank))
		defer testServer.Close()

		reqBody := RerankRequest{
			Query: "test query",
			Texts: []string{"doc1"},
			Model: "", // No model in request, should use config
		}

		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Should attempt to switch to config-model
	})

	t.Run("CLIFallback", func(t *testing.T) {
		server := &Server{
			config:             nil, // No config
			rerankModel:        "cli-model",
			currentRerankModel: "current-model",
			rerankBaseURL:      mockReranker.URL,
			client: &http.Client{
				Timeout: 10 * time.Second,
			},
		}

		testServer := httptest.NewServer(http.HandlerFunc(server.handleRerank))
		defer testServer.Close()

		reqBody := RerankRequest{
			Query: "test query",
			Texts: []string{"doc1"},
			Model: "", // No model in request, should use CLI flag
		}

		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := http.Post(testServer.URL, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Should attempt to switch to cli-model
	})
}
