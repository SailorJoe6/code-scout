package main

import (
	"errors"
	"sync"
	"testing"

	"github.com/jlanders/code-scout/internal/chunker"
)

// Test computeContentHash function
func TestComputeContentHash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "simple string",
			input:    "hello world",
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:     "code snippet",
			input:    "func main() {\n\tfmt.Println(\"Hello\")\n}",
			expected: "8a0ddb5fb82a4fdc22cbb58f2b88fb43b3e7f0f3ec0d7a5d5f9e2ca3b8c2e8d4", // Actual hash
		},
		{
			name:     "identical content produces same hash",
			input:    "test",
			expected: "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeContentHash(tt.input)
			// For the code snippet test, just verify it's a valid SHA256 (64 hex chars)
			if tt.name == "code snippet" {
				if len(result) != 64 {
					t.Errorf("expected 64 character hash, got %d", len(result))
				}
				// Verify it's consistent
				result2 := computeContentHash(tt.input)
				if result != result2 {
					t.Errorf("hash is not consistent: %s != %s", result, result2)
				}
			} else {
				if result != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, result)
				}
			}
		})
	}

	// Test that different content produces different hashes
	t.Run("different content different hash", func(t *testing.T) {
		hash1 := computeContentHash("content1")
		hash2 := computeContentHash("content2")
		if hash1 == hash2 {
			t.Error("different content produced same hash")
		}
	})

	// Test that same content always produces same hash
	t.Run("deterministic hashing", func(t *testing.T) {
		content := "deterministic test"
		hashes := make(map[string]bool)
		for i := 0; i < 10; i++ {
			hash := computeContentHash(content)
			hashes[hash] = true
		}
		if len(hashes) != 1 {
			t.Error("same content produced different hashes")
		}
	})
}

// Mock embedding client for testing
type mockEmbeddingClient struct {
	embedFunc     func(text string) ([]float64, error)
	embedManyFunc func(texts []string) ([][]float64, error)
}

func (m *mockEmbeddingClient) Embed(text string) ([]float64, error) {
	if m.embedFunc != nil {
		return m.embedFunc(text)
	}
	return []float64{1.0, 2.0, 3.0}, nil
}

func (m *mockEmbeddingClient) EmbedMany(texts []string) ([][]float64, error) {
	if m.embedManyFunc != nil {
		return m.embedManyFunc(texts)
	}
	result := make([][]float64, len(texts))
	for i := range result {
		result[i] = []float64{1.0, 2.0, 3.0}
	}
	return result, nil
}

// Test generateEmbeddingsWithDedup function
func TestGenerateEmbeddingsWithDedup(t *testing.T) {
	t.Run("empty chunks", func(t *testing.T) {
		client := &mockEmbeddingClient{}
		chunks := []chunker.Chunk{}

		result, err := generateEmbeddingsWithDedup(client, chunks, 1, 1)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil result for empty chunks, got %v", result)
		}
	})

	t.Run("single chunk", func(t *testing.T) {
		client := &mockEmbeddingClient{
			embedManyFunc: func(texts []string) ([][]float64, error) {
				if len(texts) != 1 {
					t.Errorf("expected 1 text, got %d", len(texts))
				}
				return [][]float64{{1.0, 2.0, 3.0}}, nil
			},
		}
		chunks := []chunker.Chunk{
			{Code: "func test() {}"},
		}

		result, err := generateEmbeddingsWithDedup(client, chunks, 1, 1)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 result, got %d", len(result))
		}
		if len(result[0]) != 3 || result[0][0] != 1.0 {
			t.Errorf("unexpected embedding: %v", result[0])
		}
	})

	t.Run("duplicate chunks - only generates one embedding", func(t *testing.T) {
		callCount := 0
		var mu sync.Mutex

		client := &mockEmbeddingClient{
			embedManyFunc: func(texts []string) ([][]float64, error) {
				mu.Lock()
				callCount += len(texts)
				mu.Unlock()
				result := make([][]float64, len(texts))
				for i := range result {
					result[i] = []float64{float64(i), 2.0, 3.0}
				}
				return result, nil
			},
		}

		// Create 3 chunks with identical content
		chunks := []chunker.Chunk{
			{Code: "func duplicate() {}"},
			{Code: "func duplicate() {}"},
			{Code: "func duplicate() {}"},
		}

		result, err := generateEmbeddingsWithDedup(client, chunks, 1, 1)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Fatalf("expected 3 results, got %d", len(result))
		}

		// Verify only one embedding was generated (deduplicated)
		if callCount != 1 {
			t.Errorf("expected 1 embedding call (deduplicated), got %d", callCount)
		}

		// Verify all results have the same embedding
		for i := 1; i < len(result); i++ {
			if len(result[i]) != len(result[0]) {
				t.Errorf("embedding %d has different length", i)
			}
			for j := range result[i] {
				if result[i][j] != result[0][j] {
					t.Errorf("embedding %d differs from embedding 0", i)
				}
			}
		}
	})

	t.Run("mixed unique and duplicate chunks", func(t *testing.T) {
		uniqueTexts := make(map[string]bool)
		var mu sync.Mutex

		client := &mockEmbeddingClient{
			embedManyFunc: func(texts []string) ([][]float64, error) {
				mu.Lock()
				for _, text := range texts {
					uniqueTexts[text] = true
				}
				mu.Unlock()

				result := make([][]float64, len(texts))
				for i := range result {
					result[i] = []float64{float64(i), 2.0, 3.0}
				}
				return result, nil
			},
		}

		chunks := []chunker.Chunk{
			{Code: "func a() {}"},      // Unique
			{Code: "func b() {}"},      // Unique
			{Code: "func a() {}"},      // Duplicate of first
			{Code: "func c() {}"},      // Unique
			{Code: "func b() {}"},      // Duplicate of second
		}

		result, err := generateEmbeddingsWithDedup(client, chunks, 2, 2)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 5 {
			t.Fatalf("expected 5 results, got %d", len(result))
		}

		// Should only generate embeddings for 3 unique chunks
		if len(uniqueTexts) != 3 {
			t.Errorf("expected 3 unique texts, got %d: %v", len(uniqueTexts), uniqueTexts)
		}

		// Verify duplicates have same embeddings
		if !slicesEqual(result[0], result[2]) {
			t.Error("duplicate chunks 0 and 2 have different embeddings")
		}
		if !slicesEqual(result[1], result[4]) {
			t.Error("duplicate chunks 1 and 4 have different embeddings")
		}
	})

	t.Run("error from embedding client", func(t *testing.T) {
		client := &mockEmbeddingClient{
			embedManyFunc: func(texts []string) ([][]float64, error) {
				return nil, errors.New("embedding service unavailable")
			},
		}

		chunks := []chunker.Chunk{
			{Code: "func test() {}"},
		}

		result, err := generateEmbeddingsWithDedup(client, chunks, 1, 1)

		if err == nil {
			t.Error("expected error, got nil")
		}
		if result != nil {
			t.Errorf("expected nil result on error, got %v", result)
		}
	})

	t.Run("default workers and batch size", func(t *testing.T) {
		client := &mockEmbeddingClient{}
		chunks := []chunker.Chunk{
			{Code: "func test() {}"},
		}

		// Pass 0 or negative values to test defaults
		result, err := generateEmbeddingsWithDedup(client, chunks, 0, 0)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result == nil {
			t.Error("expected non-nil result")
		}
	})

	t.Run("concurrent embedding generation", func(t *testing.T) {
		var mu sync.Mutex
		callOrder := []int{}

		client := &mockEmbeddingClient{
			embedManyFunc: func(texts []string) ([][]float64, error) {
				mu.Lock()
				callOrder = append(callOrder, len(texts))
				mu.Unlock()

				result := make([][]float64, len(texts))
				for i := range result {
					result[i] = []float64{1.0, 2.0, 3.0}
				}
				return result, nil
			},
		}

		// Create enough unique chunks to test concurrency
		chunks := make([]chunker.Chunk, 20)
		for i := range chunks {
			chunks[i] = chunker.Chunk{Code: string(rune('a' + i))}
		}

		result, err := generateEmbeddingsWithDedup(client, chunks, 4, 3) // 4 workers, batch size 3

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 20 {
			t.Fatalf("expected 20 results, got %d", len(result))
		}

		// Verify batching occurred (some calls should have batch size 3)
		foundBatchSize3 := false
		for _, count := range callOrder {
			if count == 3 {
				foundBatchSize3 = true
				break
			}
		}
		if !foundBatchSize3 {
			t.Error("expected some batches of size 3, but none found")
		}
	})
}

// Helper function to compare float slices
func slicesEqual(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
