package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultEndpoint is the default embedding API endpoint (Ollama local)
	DefaultEndpoint = "http://localhost:11434"
	// DefaultCodeModel is the default model for code embeddings
	DefaultCodeModel = "code-scout-code"
	// DefaultTextModel is the default model for text/documentation embeddings
	DefaultTextModel = "code-scout-text"
)

// Client is the interface for embedding clients
type Client interface {
	Embed(text string) ([]float64, error)
	EmbedMany(texts []string) ([][]float64, error)
}

// OpenAIClient handles communication with OpenAI-compatible embedding APIs
// (supports Ollama, OpenRouter, and other compatible services)
type OpenAIClient struct {
	endpoint string
	apiKey   string        // Optional API key for authentication
	model    string
	client   *http.Client
}

// openAIEmbedRequest represents the OpenAI-compatible embedding request
type openAIEmbedRequest struct {
	Model string      `json:"model"`
	Input interface{} `json:"input"`
}

// openAIEmbedResponse represents the OpenAI-compatible embedding response
type openAIEmbedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// NewClient creates a new embedding client with default endpoint and code model
func NewClient() *OpenAIClient {
	return &OpenAIClient{
		endpoint: DefaultEndpoint,
		model:    DefaultCodeModel,
		client:   &http.Client{},
	}
}

// NewClientWithModel creates a new embedding client with default endpoint and custom model
func NewClientWithModel(model string) *OpenAIClient {
	return &OpenAIClient{
		endpoint: DefaultEndpoint,
		model:    model,
		client:   &http.Client{},
	}
}

// NewClientWithEndpoint creates a new embedding client with custom endpoint and model
func NewClientWithEndpoint(endpoint, model string) *OpenAIClient {
	return &OpenAIClient{
		endpoint: endpoint,
		model:    model,
		client:   &http.Client{},
	}
}

// NewClientWithConfig creates a new embedding client with custom endpoint, API key, and model
func NewClientWithConfig(endpoint, apiKey, model string) *OpenAIClient {
	return &OpenAIClient{
		endpoint: endpoint,
		apiKey:   apiKey,
		model:    model,
		client:   &http.Client{},
	}
}

// Deprecated: Use NewClient instead
func NewOllamaClient() *OpenAIClient {
	return NewClient()
}

// Deprecated: Use NewClientWithModel instead
func NewOllamaClientWithModel(model string) *OpenAIClient {
	return NewClientWithModel(model)
}

// Deprecated: Use NewClientWithEndpoint instead
func NewOllamaClientWithEndpoint(endpoint, model string) *OpenAIClient {
	return NewClientWithEndpoint(endpoint, model)
}

// Embed generates an embedding for the given text using OpenAI-compatible API with retry logic
func (c *OpenAIClient) Embed(text string) ([]float64, error) {
	embeddings, err := c.EmbedMany([]string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return embeddings[0], nil
}

// EmbedMany generates embeddings for multiple texts in a single API request when possible
func (c *OpenAIClient) EmbedMany(texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	return c.embedWithRetry(texts, len(texts))
}

// EmbedBatch generates embeddings for multiple texts (alias for EmbedMany)
func (c *OpenAIClient) EmbedBatch(texts []string) ([][]float64, error) {
	return c.EmbedMany(texts)
}

func (c *OpenAIClient) embedWithRetry(texts []string, expected int) ([][]float64, error) {
	const maxRetries = 3
	const initialBackoff = 1 * time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := initialBackoff * time.Duration(1<<uint(attempt-1))
			time.Sleep(backoff)
		}

		embeddings, err := c.embedOnce(texts)
		if err == nil {
			if len(embeddings) != expected {
				return nil, fmt.Errorf("expected %d embeddings, got %d", expected, len(embeddings))
			}
			return embeddings, nil
		}

		lastErr = err
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// embedOnce makes a single embedding request without retries
func (c *OpenAIClient) embedOnce(texts []string) ([][]float64, error) {
	reqBody := openAIEmbedRequest{
		Model: c.model,
		Input: texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.endpoint + "/v1/embeddings"
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
		return nil, fmt.Errorf("failed to make request to embedding API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding API returned status %d: %s", resp.StatusCode, string(body))
	}

	var embedResp openAIEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embedResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	embeddings := make([][]float64, len(embedResp.Data))
	for i, data := range embedResp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}
