package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock TEI server for testing
func createMockTEI(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))

		case "/embed":
			// Parse request
			var req TEIRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Failed to parse request: %v", err)
			}

			// Return mock embeddings (768-dimensional vectors)
			embeddings := make([][]float64, len(req.Inputs))
			for i := range req.Inputs {
				embeddings[i] = make([]float64, 768)
				// Fill with simple mock values
				for j := range embeddings[i] {
					embeddings[i][j] = float64(i) + float64(j)*0.001
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(embeddings)

		default:
			http.NotFound(w, r)
		}
	}))
}

// Mock reranker TEI server for testing
func createMockRerankerTEI(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))

		case "/rerank":
			// Parse request
			var req RerankRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Failed to parse rerank request: %v", err)
			}

			// Return mock rerank results (sorted by score, highest first)
			results := make([]RerankResult, len(req.Texts))
			for i := range req.Texts {
				results[i] = RerankResult{
					Index: i,
					Score: 1.0 - float64(i)*0.1, // Decreasing scores
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(results)

		default:
			http.NotFound(w, r)
		}
	}))
}
