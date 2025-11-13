package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	// DefaultOllamaEndpoint is the default Ollama API endpoint
	DefaultOllamaEndpoint = "http://localhost:11434"
	// DefaultCodeModel is the default model for code embeddings
	DefaultCodeModel = "code-scout-code"
)

// OllamaClient handles communication with Ollama API
type OllamaClient struct {
	endpoint string
	model    string
	client   *http.Client
}

// ollamaEmbedRequest represents the request to Ollama's embed API
type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaEmbedResponse represents the response from Ollama's embed API
type ollamaEmbedResponse struct {
	Embedding []float64 `json:"embedding"`
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient() *OllamaClient {
	return &OllamaClient{
		endpoint: DefaultOllamaEndpoint,
		model:    DefaultCodeModel,
		client:   &http.Client{},
	}
}

// Embed generates an embedding for the given text
func (c *OllamaClient) Embed(text string) ([]float64, error) {
	// Prepare request
	reqBody := ollamaEmbedRequest{
		Model:  c.model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	url := c.endpoint + "/api/embeddings"
	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to make request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var embedResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embedResp.Embedding, nil
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
