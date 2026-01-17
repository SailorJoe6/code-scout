package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RerankClient handles reranking using cross-encoder models via TEI /rerank endpoint
type RerankClient struct {
	endpoint string
	apiKey   string // Optional API key for authentication
	model    string
	client   *http.Client
}

// RerankRequest represents the TEI /rerank request
type RerankRequest struct {
	Query      string   `json:"query"`
	Texts      []string `json:"texts"`
	RawScores  bool     `json:"raw_scores,omitempty"`
	ReturnText bool     `json:"return_text,omitempty"`
}

// RerankResult represents a single reranking result
type RerankResult struct {
	Index int     `json:"index"`
	Score float64 `json:"score"`
	Text  string  `json:"text,omitempty"` // Only present if return_text=true
}

// NewRerankClient creates a new reranking client
func NewRerankClient(endpoint, apiKey, model string) *RerankClient {
	return &RerankClient{
		endpoint: endpoint,
		apiKey:   apiKey,
		model:    model,
		client:   &http.Client{Timeout: 120 * time.Second},
	}
}

// Rerank reranks texts based on their relevance to the query using a cross-encoder model
// Returns results sorted by relevance score (highest first)
func (c *RerankClient) Rerank(query string, texts []string) ([]RerankResult, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	const maxRetries = 3
	const initialBackoff = 1 * time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := initialBackoff * time.Duration(1<<uint(attempt-1))
			time.Sleep(backoff)
		}

		results, err := c.rerankOnce(query, texts)
		if err == nil {
			return results, nil
		}

		lastErr = err
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// rerankOnce makes a single reranking request without retries
func (c *RerankClient) rerankOnce(query string, texts []string) ([]RerankResult, error) {
	reqBody := RerankRequest{
		Query:      query,
		Texts:      texts,
		RawScores:  false, // Use normalized scores
		ReturnText: false, // Don't need text in response (we already have it)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.endpoint + "/rerank"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Add Authorization header if API key is provided
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to rerank API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("rerank API returned status %d: %s", resp.StatusCode, string(body))
	}

	var results []RerankResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return results, nil
}
