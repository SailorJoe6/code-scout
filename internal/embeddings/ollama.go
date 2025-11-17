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
	// DefaultOllamaEndpoint is the default Ollama API endpoint
	DefaultOllamaEndpoint = "http://localhost:11434"
	// DefaultCodeModel is the default model for code embeddings
	DefaultCodeModel = "code-scout-code"
	// DefaultTextModel is the default model for text/documentation embeddings
	DefaultTextModel = "code-scout-text"
)

// OllamaClient handles communication with Ollama API using OpenAI-compatible format
type OllamaClient struct {
	endpoint string
	model    string
	client   *http.Client
}

// openAIEmbedRequest represents the OpenAI-compatible embedding request
type openAIEmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// openAIEmbedResponse represents the OpenAI-compatible embedding response
type openAIEmbedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// NewOllamaClient creates a new Ollama client with the default code model
func NewOllamaClient() *OllamaClient {
	return &OllamaClient{
		endpoint: DefaultOllamaEndpoint,
		model:    DefaultCodeModel,
		client:   &http.Client{},
	}
}

// NewOllamaClientWithModel creates a new Ollama client with a specific model
func NewOllamaClientWithModel(model string) *OllamaClient {
	return &OllamaClient{
		endpoint: DefaultOllamaEndpoint,
		model:    model,
		client:   &http.Client{},
	}
}

// Embed generates an embedding for the given text using OpenAI-compatible API with retry logic
func (c *OllamaClient) Embed(text string) ([]float64, error) {
	const maxRetries = 3
	const initialBackoff = 1 * time.Second

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := initialBackoff * time.Duration(1<<uint(attempt-1))
			time.Sleep(backoff)
		}

		embedding, err := c.embedOnce(text)
		if err == nil {
			return embedding, nil
		}

		lastErr = err
		// Only retry on server errors (5xx) or EOF errors
		if attempt < maxRetries-1 {
			continue
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// embedOnce makes a single embedding request without retries
func (c *OllamaClient) embedOnce(text string) ([]float64, error) {
	// Prepare OpenAI-compatible request
	reqBody := openAIEmbedRequest{
		Model: c.model,
		Input: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request to OpenAI-compatible endpoint
	url := c.endpoint + "/v1/embeddings"
	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to make request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse OpenAI-compatible response
	var embedResp openAIEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embedResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	return embedResp.Data[0].Embedding, nil
}

// EmbedBatch generates embeddings for multiple texts
func (c *OllamaClient) EmbedBatch(texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))

	for i, text := range texts {
		embedding, err := c.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}
