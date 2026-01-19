package embeddings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewRerankClient tests the rerank client constructor
func TestNewRerankClient(t *testing.T) {
	endpoint := "http://localhost:8080"
	apiKey := "sk-test-key"
	model := "BAAI/bge-reranker-large"

	client := NewRerankClient(endpoint, apiKey, model)
	if client == nil {
		t.Fatal("NewRerankClient returned nil")
	}
	if client.endpoint != endpoint {
		t.Errorf("Expected endpoint %s, got %s", endpoint, client.endpoint)
	}
	if client.apiKey != apiKey {
		t.Errorf("Expected API key %s, got %s", apiKey, client.apiKey)
	}
	if client.model != model {
		t.Errorf("Expected model %s, got %s", model, client.model)
	}
}

// TestRerankSuccess tests successful reranking
func TestRerankSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/rerank" {
			t.Errorf("Expected path /rerank, got %s", r.URL.Path)
		}

		// Check request body
		var req RerankRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Query != "test query" {
			t.Errorf("Expected query 'test query', got %s", req.Query)
		}
		if len(req.Texts) != 3 {
			t.Errorf("Expected 3 texts, got %d", len(req.Texts))
		}

		// Return mock rerank results (sorted by score)
		resp := []RerankResult{
			{Index: 1, Score: 0.95},
			{Index: 0, Score: 0.80},
			{Index: 2, Score: 0.65},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRerankClient(server.URL, "", "test-model")
	results, err := client.Rerank("test query", []string{"text1", "text2", "text3"})
	if err != nil {
		t.Fatalf("Rerank failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
	// Results should be sorted by score (highest first)
	if results[0].Index != 1 || results[0].Score != 0.95 {
		t.Errorf("Expected result[0] to be index 1 with score 0.95, got index %d with score %f",
			results[0].Index, results[0].Score)
	}
	if results[1].Index != 0 || results[1].Score != 0.80 {
		t.Errorf("Expected result[1] to be index 0 with score 0.80, got index %d with score %f",
			results[1].Index, results[1].Score)
	}
	if results[2].Index != 2 || results[2].Score != 0.65 {
		t.Errorf("Expected result[2] to be index 2 with score 0.65, got index %d with score %f",
			results[2].Index, results[2].Score)
	}
}

// TestRerankEmpty tests reranking with empty texts
func TestRerankEmpty(t *testing.T) {
	client := NewRerankClient("http://localhost:8080", "", "test-model")
	results, err := client.Rerank("test query", []string{})
	if err != nil {
		t.Errorf("Rerank with empty texts should not error, got: %v", err)
	}
	if results != nil {
		t.Errorf("Expected nil results for empty texts, got %v", results)
	}
}

// TestRerankWithAPIKey tests that API key is sent in Authorization header
func TestRerankWithAPIKey(t *testing.T) {
	expectedKey := "sk-test-key-123"
	keyReceived := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "Bearer "+expectedKey {
			keyReceived = true
		}

		resp := []RerankResult{
			{Index: 0, Score: 0.95},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRerankClient(server.URL, expectedKey, "test-model")
	_, err := client.Rerank("test", []string{"text1"})
	if err != nil {
		t.Fatalf("Rerank failed: %v", err)
	}

	if !keyReceived {
		t.Error("API key was not sent in Authorization header")
	}
}

// TestRerankHTTPError tests handling of non-200 status codes
func TestRerankHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewRerankClient(server.URL, "", "test-model")
	_, err := client.Rerank("test", []string{"text1"})
	if err == nil {
		t.Fatal("Expected error for 500 status code")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Expected error to mention status code 500, got: %v", err)
	}
}

// TestRerankInvalidJSON tests handling of invalid JSON response
func TestRerankInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewRerankClient(server.URL, "", "test-model")
	_, err := client.Rerank("test", []string{"text1"})
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("Expected decode error, got: %v", err)
	}
}

// TestRerankRetrySuccess tests retry logic when first attempt fails
func TestRerankRetrySuccess(t *testing.T) {
	attemptCount := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attemptCount.Add(1)
		if count < 2 {
			// Fail first attempt
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Succeed on second attempt
		resp := []RerankResult{
			{Index: 0, Score: 0.95},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRerankClient(server.URL, "", "test-model")
	client.client.Timeout = 5 * time.Second

	results, err := client.Rerank("test", []string{"text1"})
	if err != nil {
		t.Fatalf("Rerank should succeed after retry, got error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if attemptCount.Load() != 2 {
		t.Errorf("Expected 2 attempts, got %d", attemptCount.Load())
	}
}

// TestRerankRetryExhaustion tests retry logic when all attempts fail
func TestRerankRetryExhaustion(t *testing.T) {
	attemptCount := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("persistent error"))
	}))
	defer server.Close()

	client := NewRerankClient(server.URL, "", "test-model")
	client.client.Timeout = 5 * time.Second

	_, err := client.Rerank("test", []string{"text1"})
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

// TestRerankContentType tests that Content-Type header is set
func TestRerankContentType(t *testing.T) {
	contentTypeReceived := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") == "application/json" {
			contentTypeReceived = true
		}

		resp := []RerankResult{
			{Index: 0, Score: 0.95},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRerankClient(server.URL, "", "test-model")
	_, err := client.Rerank("test", []string{"text1"})
	if err != nil {
		t.Fatalf("Rerank failed: %v", err)
	}

	if !contentTypeReceived {
		t.Error("Content-Type header was not set to application/json")
	}
}

// TestRerankRawScoresParameter tests that raw_scores parameter is set correctly
func TestRerankRawScoresParameter(t *testing.T) {
	rawScoresValue := true // Will be set by handler

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req RerankRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// We expect raw_scores to be false (normalized scores)
		rawScoresValue = req.RawScores

		resp := []RerankResult{
			{Index: 0, Score: 0.95},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRerankClient(server.URL, "", "test-model")
	_, err := client.Rerank("test", []string{"text1"})
	if err != nil {
		t.Fatalf("Rerank failed: %v", err)
	}

	if rawScoresValue != false {
		t.Errorf("Expected raw_scores to be false, got %v", rawScoresValue)
	}
}

// TestRerankModelField tests that the Model field is included in the request
func TestRerankModelField(t *testing.T) {
	expectedModel := "BAAI/bge-reranker-large"
	modelReceived := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req RerankRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		modelReceived = req.Model

		resp := []RerankResult{
			{Index: 0, Score: 0.95},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewRerankClient(server.URL, "", expectedModel)
	_, err := client.Rerank("test", []string{"text1"})
	if err != nil {
		t.Fatalf("Rerank failed: %v", err)
	}

	if modelReceived != expectedModel {
		t.Errorf("Expected model %s, got %s", expectedModel, modelReceived)
	}
}

// TestRerankRequestMarshaling tests that RerankRequest marshals correctly with Model field
func TestRerankRequestMarshaling(t *testing.T) {
	req := RerankRequest{
		Query:      "test query",
		Texts:      []string{"text1", "text2"},
		RawScores:  false,
		ReturnText: false,
		Model:      "BAAI/bge-reranker-large",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal RerankRequest: %v", err)
	}

	// Unmarshal to verify all fields
	var decoded RerankRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal RerankRequest: %v", err)
	}

	if decoded.Query != req.Query {
		t.Errorf("Expected query %s, got %s", req.Query, decoded.Query)
	}
	if len(decoded.Texts) != len(req.Texts) {
		t.Errorf("Expected %d texts, got %d", len(req.Texts), len(decoded.Texts))
	}
	if decoded.Model != req.Model {
		t.Errorf("Expected model %s, got %s", req.Model, decoded.Model)
	}
}

// BenchmarkRerank benchmarks reranking
func BenchmarkRerank(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req RerankRequest
		json.NewDecoder(r.Body).Decode(&req)

		results := make([]RerankResult, len(req.Texts))
		for i := range results {
			results[i] = RerankResult{
				Index: i,
				Score: 0.5 + float64(i)*0.1,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}))
	defer server.Close()

	client := NewRerankClient(server.URL, "", "test-model")
	texts := []string{"text1", "text2", "text3", "text4", "text5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Rerank("test query", texts)
		if err != nil {
			b.Fatalf("Rerank failed: %v", err)
		}
	}
}
