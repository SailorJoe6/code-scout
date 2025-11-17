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
type Client interface {
	Embed(text string) ([]float64, error)
	EmbedMany(texts []string) ([][]float64, error)
}

// OllamaClient handles communication with Ollama API using OpenAI-compatible format
type OllamaClient struct {
	endpoint string
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
func (c *OllamaClient) EmbedMany(texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	return c.embedWithRetry(texts, len(texts))
}

// EmbedBatch generates embeddings for multiple texts (alias for EmbedMany)
func (c *OllamaClient) EmbedBatch(texts []string) ([][]float64, error) {
	return c.EmbedMany(texts)
}

func (c *OllamaClient) embedWithRetry(texts []string, expected int) ([][]float64, error) {
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
func (c *OllamaClient) embedOnce(texts []string) ([][]float64, error) {
	reqBody := openAIEmbedRequest{
		Model: c.model,
		Input: texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

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
