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
