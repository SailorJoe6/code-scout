package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// OpenAI API request format
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// OpenAI API response format
type EmbeddingResponse struct {
	Object string            `json:"object"`
	Data   []EmbeddingData   `json:"data"`
	Model  string            `json:"model"`
	Usage  EmbeddingUsage    `json:"usage"`
}

type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// TEI request format (simpler)
type TEIRequest struct {
	Inputs []string `json:"inputs"`
}

// TEI response format
type TEIResponse [][]float64

// Server manages the TEI wrapper
type Server struct {
	teiPort      int
	teiBinary    string
	initialModel string
	currentModel string        // Currently loaded model
	teiCmd       *exec.Cmd
	teiBaseURL   string
	client       *http.Client
	mu           sync.RWMutex  // Protects model switching
	switching    bool          // True during model switch
}

func main() {
	// Command line flags
	port := flag.Int("port", 11434, "Port to listen on (Ollama-compatible default)")
	teiPort := flag.Int("tei-port", 8080, "TEI internal port")
	teiBinary := flag.String("tei-binary", "text-embeddings-router", "Path to TEI binary")
	model := flag.String("model", "nomic-ai/nomic-embed-text-v1.5", "Initial model to load")
	flag.Parse()

	// Create server
	server := &Server{
		teiPort:      *teiPort,
		teiBinary:    *teiBinary,
		initialModel: *model,
		currentModel: *model,
		teiBaseURL:   fmt.Sprintf("http://localhost:%d", *teiPort),
		client: &http.Client{
			Timeout: 120 * time.Second, // Long timeout for large batches
		},
	}

	// Start TEI process
	log.Printf("Starting TEI with model: %s", server.initialModel)
	if err := server.startTEIWithModel(context.Background(), server.initialModel); err != nil {
		log.Fatalf("Failed to start TEI: %v", err)
	}
	defer server.stopTEI()

	// Wait for TEI to be ready
	log.Printf("Waiting for TEI to be ready...")
	if err := server.waitForTEI(30 * time.Second); err != nil {
		log.Fatalf("TEI failed to start: %v", err)
	}
	log.Printf("TEI is ready!")

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/embeddings", server.handleEmbeddings)
	mux.HandleFunc("/health", server.handleHealth)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: mux,
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(ctx)
	}()

	// Start server
	log.Printf("TEI wrapper listening on :%d", *port)
	log.Printf("OpenAI-compatible endpoint: http://localhost:%d/v1/embeddings", *port)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

// startTEIWithModel starts the TEI process with the specified model
func (s *Server) startTEIWithModel(ctx context.Context, model string) error {
	// TEI command: text-embeddings-router --model-id <model> --port <port>
	s.teiCmd = exec.CommandContext(ctx, s.teiBinary,
		"--model-id", model,
		"--port", fmt.Sprintf("%d", s.teiPort),
		"--max-batch-tokens", "16384", // Reasonable default
	)

	// Capture output for debugging
	s.teiCmd.Stdout = os.Stdout
	s.teiCmd.Stderr = os.Stderr

	if err := s.teiCmd.Start(); err != nil {
		return fmt.Errorf("failed to start TEI: %w", err)
	}

	log.Printf("TEI process started with model %s (PID: %d)", model, s.teiCmd.Process.Pid)
	s.currentModel = model
	return nil
}

// stopTEI gracefully stops the TEI process
func (s *Server) stopTEI() {
	if s.teiCmd == nil || s.teiCmd.Process == nil {
		return
	}

	log.Printf("Stopping TEI process (PID: %d)", s.teiCmd.Process.Pid)

	// Send SIGTERM for graceful shutdown
	if err := s.teiCmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.Printf("Failed to send SIGTERM: %v", err)
		s.teiCmd.Process.Kill()
		return
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- s.teiCmd.Wait()
	}()

	select {
	case <-done:
		log.Printf("TEI stopped gracefully")
	case <-time.After(5 * time.Second):
		log.Printf("TEI didn't stop in time, killing...")
		s.teiCmd.Process.Kill()
	}
}

// waitForTEI waits for TEI to be ready by polling the health endpoint
func (s *Server) waitForTEI(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := s.client.Get(s.teiBaseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("TEI did not become ready within %v", timeout)
}

// switchModel switches to a new model by stopping and restarting TEI
func (s *Server) switchModel(newModel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already on the requested model
	if s.currentModel == newModel {
		return nil
	}

	log.Printf("Switching model from %s to %s", s.currentModel, newModel)
	s.switching = true
	defer func() { s.switching = false }()

	// Stop current TEI process
	s.stopTEI()

	// Start new TEI process with new model
	ctx := context.Background()
	if err := s.startTEIWithModel(ctx, newModel); err != nil {
		return fmt.Errorf("failed to start TEI with new model: %w", err)
	}

	// Wait for new TEI to be ready
	if err := s.waitForTEI(30 * time.Second); err != nil {
		return fmt.Errorf("new TEI failed to start: %w", err)
	}

	log.Printf("Model switched successfully to %s", newModel)
	return nil
}

// handleEmbeddings handles POST /v1/embeddings requests
func (s *Server) handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse OpenAI request
	var req EmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.Input) == 0 {
		http.Error(w, "No input provided", http.StatusBadRequest)
		return
	}

	// Check if we need to switch models
	s.mu.RLock()
	needsSwitch := req.Model != "" && req.Model != s.currentModel
	isSwitching := s.switching
	s.mu.RUnlock()

	if isSwitching {
		// Return 503 with Retry-After header during switch
		w.Header().Set("Retry-After", "5")
		http.Error(w, "Model switch in progress, please retry", http.StatusServiceUnavailable)
		return
	}

	if needsSwitch {
		// Switch to the requested model
		if err := s.switchModel(req.Model); err != nil {
			log.Printf("Model switch failed: %v", err)
			http.Error(w, fmt.Sprintf("Model switch failed: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Forward to TEI
	embeddings, err := s.getEmbeddings(req.Input)
	if err != nil {
		log.Printf("TEI request failed: %v", err)
		http.Error(w, fmt.Sprintf("Embedding failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Build OpenAI-compatible response
	resp := EmbeddingResponse{
		Object: "list",
		Model:  req.Model,
		Data:   make([]EmbeddingData, len(embeddings)),
		Usage: EmbeddingUsage{
			PromptTokens: len(req.Input),
			TotalTokens:  len(req.Input),
		},
	}

	for i, emb := range embeddings {
		resp.Data[i] = EmbeddingData{
			Object:    "embedding",
			Embedding: emb,
			Index:     i,
		}
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// getEmbeddings sends a request to TEI and returns the embeddings
func (s *Server) getEmbeddings(inputs []string) ([][]float64, error) {
	// Build TEI request
	teiReq := TEIRequest{
		Inputs: inputs,
	}

	reqBody, err := json.Marshal(teiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request to TEI
	resp, err := s.client.Post(
		s.teiBaseURL+"/embed",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to TEI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TEI returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse TEI response
	var teiResp TEIResponse
	if err := json.NewDecoder(resp.Body).Decode(&teiResp); err != nil {
		return nil, fmt.Errorf("failed to parse TEI response: %w", err)
	}

	return teiResp, nil
}

// handleHealth returns the health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	currentModel := s.currentModel
	isSwitching := s.switching
	s.mu.RUnlock()

	// Check if currently switching models
	if isSwitching {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "switching",
			"model":     currentModel,
			"switching": true,
		})
		return
	}

	// Check if TEI is healthy
	resp, err := s.client.Get(s.teiBaseURL + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "unhealthy",
			"model":  currentModel,
			"error":  "TEI is not responding",
		})
		return
	}
	resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"model":  currentModel,
	})
}
