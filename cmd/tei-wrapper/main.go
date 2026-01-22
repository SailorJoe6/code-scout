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

	"github.com/jlanders/code-scout/internal/config"
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
	config       *config.Config // Config from .code-scout.json
	multiModel   bool
	teiPort      int
	teiBinary    string
	initialModel string
	currentModel string        // Currently loaded model
	teiCmd       *exec.Cmd
	teiBaseURL   string
	client       *http.Client
	mu           sync.RWMutex  // Protects model switching
	switching    bool          // True during model switch

	// Idle-based preloading (Slice 3 - sequential, no RAM penalty)
	idlePreload     bool          // Enable idle preloading
	idleTimeout     time.Duration // Time before triggering preload
	lastRequestTime time.Time     // Last request timestamp
	preferredModel  string        // Model to preload when idle (code model)
	idleTimer       *time.Timer   // Timer for idle detection

	// Memory management
	maxBatchTokens int           // Max batch tokens for TEI (controls memory usage)

	// Reranker TEI management (dedicated second instance)
	rerankPort           int           // Port for reranker TEI (default: 8081)
	rerankModel          string        // Model ID for reranker (empty = disabled, from CLI flag)
	currentRerankModel   string        // Currently loaded reranker model
	rerankCmd            *exec.Cmd     // Reranker TEI process
	rerankBaseURL        string        // http://localhost:{rerankPort}
	rerankHealthy        bool          // Current health status
	rerankSwitching      bool          // True during reranker model switch

	// Multi-model TEI management (code + text)
	codeModel       string
	textModel       string
	codeTEIPort     int
	textTEIPort     int
	codeTEICmd      *exec.Cmd
	textTEICmd      *exec.Cmd
	codeTEIBaseURL  string
	textTEIBaseURL  string
}

func main() {
	// Load config from .code-scout.json (or use defaults)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	singleModelMode := true
	if cfg.SingleModelMode != nil {
		singleModelMode = *cfg.SingleModelMode
	}

	textTEIPortDefault := cfg.TextTEIPort
	if textTEIPortDefault == 0 {
		textTEIPortDefault = 8080
	}
	codeTEIPortDefault := cfg.CodeTEIPort
	if codeTEIPortDefault == 0 {
		codeTEIPortDefault = 8082
	}
	rerankPortDefault := cfg.RerankPort
	if rerankPortDefault == 0 {
		rerankPortDefault = 8081
	}

	// Command line flags (with config defaults)
	port := flag.Int("port", 11435, "Port to listen on")
	teiPort := flag.Int("tei-port", textTEIPortDefault, "TEI internal port (text model in multi-model mode)")
	codeTEIPort := flag.Int("code-tei-port", codeTEIPortDefault, "TEI port for code model (multi-model mode)")
	teiBinary := flag.String("tei-binary", "text-embeddings-router", "Path to TEI binary")

	// Model flags with config file as defaults (CLI overrides config)
	model := flag.String("model", cfg.TextModel, "Initial model (text model in multi-model mode)")
	idlePreload := flag.Bool("idle-preload", false, "Enable idle-based preloading of text model")
	idleTimeout := flag.Duration("idle-timeout", 30*time.Second, "Idle time before preloading text model")
	maxBatchTokens := flag.Int("max-batch-tokens", 8192, "Maximum batch tokens for TEI (controls memory usage, lower = less RAM)")

	// Reranker flags (optional - enables dedicated reranker TEI instance)
	rerankPort := flag.Int("rerank-port", rerankPortDefault, "Port for reranker TEI instance")
	rerankModel := flag.String("rerank-model", "", "Model ID for reranker (deprecated; empty = load on-demand from config)")
	flag.Parse()

	rerankModelSet := false
	flag.CommandLine.Visit(func(f *flag.Flag) {
		if f.Name == "rerank-model" {
			rerankModelSet = true
		}
	})

	// Priority: CLI flag > config file > hardcoded default
	initialModel := *model
	if initialModel == "" {
		initialModel = "nomic-ai/nomic-embed-text-v1.5" // Hardcoded fallback
	}

	textModel := initialModel
	codeModel := cfg.CodeModel

	// Create server
	server := &Server{
		config:       cfg,
		multiModel:   !singleModelMode,
		teiPort:      *teiPort,
		teiBinary:    *teiBinary,
		initialModel: initialModel,
		currentModel: initialModel,
		teiBaseURL:   fmt.Sprintf("http://localhost:%d", *teiPort),
		client: &http.Client{
			Timeout: 120 * time.Second, // Long timeout for large batches
		},
		idlePreload:    *idlePreload,
		idleTimeout:    *idleTimeout,
		preferredModel: cfg.TextModel, // Prefer text model from config when idle
		maxBatchTokens: *maxBatchTokens,
		// Reranker config
		rerankPort:    *rerankPort,
		rerankModel:   *rerankModel,
		rerankBaseURL: fmt.Sprintf("http://localhost:%d", *rerankPort),
		// Multi-model config
		codeModel:      codeModel,
		textModel:      textModel,
		codeTEIPort:    *codeTEIPort,
		textTEIPort:    *teiPort,
		codeTEIBaseURL: fmt.Sprintf("http://localhost:%d", *codeTEIPort),
		textTEIBaseURL: fmt.Sprintf("http://localhost:%d", *teiPort),
	}

	if !server.multiModel {
		log.Printf("Starting TEI with model: %s", server.initialModel)
		if err := server.startTEIWithModel(context.Background(), server.initialModel); err != nil {
			log.Fatalf("Failed to start TEI: %v", err)
		}
		defer server.stopTEI()

		log.Printf("Waiting for TEI to be ready...")
		if err := server.waitForTEI(90 * time.Second); err != nil {
			log.Fatalf("TEI failed to start: %v", err)
		}
		log.Printf("TEI is ready!")
	} else {
		log.Printf("Starting multi-model TEI (code: %s, text: %s)", server.codeModel, server.textModel)
		if err := server.startMultiTEI(context.Background()); err != nil {
			log.Fatalf("Failed to start multi-model TEI: %v", err)
		}
		defer server.stopMultiTEI()

		log.Printf("Waiting for TEI processes to be ready...")
		if err := server.waitForMultiTEI(90 * time.Second); err != nil {
			log.Fatalf("TEI processes failed to start: %v", err)
		}
		log.Printf("TEI processes are ready!")
	}

	// Start reranker TEI only when explicitly set via CLI flag
	if rerankModelSet && server.rerankModel != "" {
		log.Printf("Starting reranker TEI with model: %s", server.rerankModel)
		if err := server.startRerankTEI(context.Background()); err != nil {
			log.Fatalf("Failed to start reranker TEI: %v", err)
		}
		defer server.stopRerankTEI()

		log.Printf("Waiting for reranker TEI to be ready...")
		if err := server.waitForRerankTEI(90 * time.Second); err != nil {
			log.Fatalf("Reranker TEI failed to start: %v", err)
		}
		server.currentRerankModel = server.rerankModel
		log.Printf("Reranker TEI is ready!")
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/embeddings", server.handleEmbeddings)
	mux.HandleFunc("/rerank", server.handleRerank)
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
	if cfg.RerankModel != "" {
		log.Printf("Rerank endpoint: http://localhost:%d/rerank (model: %s, loads on-demand)", *port, cfg.RerankModel)
	} else {
		log.Printf("Rerank endpoint: http://localhost:%d/rerank (set rerank_model in config to enable)", *port)
	}
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

// startTEIProcess starts a TEI process for a model/port pair.
func (s *Server) startTEIProcess(ctx context.Context, model string, port int) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, s.teiBinary,
		"--model-id", model,
		"--port", fmt.Sprintf("%d", port),
		"--max-batch-tokens", fmt.Sprintf("%d", s.maxBatchTokens),
	)

	// Capture output for debugging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start TEI: %w", err)
	}

	log.Printf("TEI process started with model %s (PID: %d, port: %d)", model, cmd.Process.Pid, port)
	return cmd, nil
}

// startTEIWithModel starts the TEI process with the specified model.
func (s *Server) startTEIWithModel(ctx context.Context, model string) error {
	cmd, err := s.startTEIProcess(ctx, model, s.teiPort)
	if err != nil {
		return err
	}
	s.teiCmd = cmd
	s.currentModel = model
	return nil
}

// stopTEIProcess gracefully stops a TEI process.
func (s *Server) stopTEIProcess(cmd *exec.Cmd, label string) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	log.Printf("Stopping %s process (PID: %d)", label, cmd.Process.Pid)

	// Send SIGTERM for graceful shutdown
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.Printf("Failed to send SIGTERM to %s: %v", label, err)
		cmd.Process.Kill()
		return
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		log.Printf("%s stopped gracefully", label)
	case <-time.After(5 * time.Second):
		log.Printf("%s didn't stop in time, killing...", label)
		cmd.Process.Kill()
	}
}

// stopTEI gracefully stops the TEI process.
func (s *Server) stopTEI() {
	s.stopTEIProcess(s.teiCmd, "TEI")
}

func (s *Server) stopMultiTEI() {
	s.stopTEIProcess(s.codeTEICmd, "code TEI")
	s.stopTEIProcess(s.textTEICmd, "text TEI")
}

// waitForTEIAt waits for a TEI instance to be ready by polling the health endpoint.
func (s *Server) waitForTEIAt(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := s.client.Get(baseURL + "/health")
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

// waitForTEI waits for TEI to be ready by polling the health endpoint.
func (s *Server) waitForTEI(timeout time.Duration) error {
	return s.waitForTEIAt(s.teiBaseURL, timeout)
}

func (s *Server) startMultiTEI(ctx context.Context) error {
	if s.codeModel == "" || s.textModel == "" {
		return fmt.Errorf("code_model and text_model must be set for multi-model mode")
	}

	codeCmd, err := s.startTEIProcess(ctx, s.codeModel, s.codeTEIPort)
	if err != nil {
		return fmt.Errorf("code TEI: %w", err)
	}
	s.codeTEICmd = codeCmd

	textCmd, err := s.startTEIProcess(ctx, s.textModel, s.textTEIPort)
	if err != nil {
		s.stopTEIProcess(s.codeTEICmd, "code TEI")
		return fmt.Errorf("text TEI: %w", err)
	}
	s.textTEICmd = textCmd

	return nil
}

func (s *Server) waitForMultiTEI(timeout time.Duration) error {
	if err := s.waitForTEIAt(s.textTEIBaseURL, timeout); err != nil {
		return fmt.Errorf("text TEI: %w", err)
	}
	if err := s.waitForTEIAt(s.codeTEIBaseURL, timeout); err != nil {
		return fmt.Errorf("code TEI: %w", err)
	}
	return nil
}

// switchModel switches to a new model by stopping and restarting TEI
func (s *Server) switchModel(newModel string) error {
	if s.multiModel {
		return fmt.Errorf("model switching is disabled in multi-model mode")
	}

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
	if err := s.waitForTEI(90 * time.Second); err != nil {
		return fmt.Errorf("new TEI failed to start: %w", err)
	}

	log.Printf("Model switched successfully to %s", newModel)
	return nil
}

// switchRerankModel switches to a new reranker model by stopping and restarting reranker TEI
func (s *Server) switchRerankModel(newModel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already on the requested model
	if s.currentRerankModel == newModel {
		return nil
	}

	log.Printf("Switching reranker model from %s to %s", s.currentRerankModel, newModel)
	s.rerankSwitching = true
	defer func() { s.rerankSwitching = false }()

	// Stop current reranker TEI if running
	if s.rerankCmd != nil {
		s.stopRerankTEI()
	}

	// Start new reranker TEI with new model
	s.rerankModel = newModel
	ctx := context.Background()
	if err := s.startRerankTEI(ctx); err != nil {
		return fmt.Errorf("failed to start reranker TEI with new model: %w", err)
	}

	// Wait for new reranker TEI to be ready
	if err := s.waitForRerankTEI(90 * time.Second); err != nil {
		return fmt.Errorf("new reranker TEI failed to start: %w", err)
	}

	s.currentRerankModel = newModel
	log.Printf("Reranker model switched successfully to %s", newModel)
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

	var embeddings [][]float64
	selectedModel := req.Model

	if !s.multiModel {
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
		var err error
		embeddings, err = s.getEmbeddings(req.Input)
		if err != nil {
			log.Printf("TEI request failed: %v", err)
			http.Error(w, fmt.Sprintf("Embedding failed: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		targetURL, resolvedModel, err := s.resolveEmbeddingTarget(req.Model)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		selectedModel = resolvedModel

		embeddings, err = s.getEmbeddingsFrom(targetURL, req.Input)
		if err != nil {
			log.Printf("TEI request failed: %v", err)
			http.Error(w, fmt.Sprintf("Embedding failed: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Build OpenAI-compatible response
	resp := EmbeddingResponse{
		Object: "list",
		Model:  selectedModel,
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

	// Reset idle timer for preloading
	s.resetIdleTimer()
}

func (s *Server) resolveEmbeddingTarget(requestedModel string) (string, string, error) {
	if requestedModel == "" {
		return s.textTEIBaseURL, s.textModel, nil
	}
	if requestedModel == s.textModel {
		return s.textTEIBaseURL, s.textModel, nil
	}
	if requestedModel == s.codeModel {
		return s.codeTEIBaseURL, s.codeModel, nil
	}
	return "", "", fmt.Errorf("unknown embedding model: %s", requestedModel)
}

// getEmbeddings sends a request to TEI and returns the embeddings.
func (s *Server) getEmbeddings(inputs []string) ([][]float64, error) {
	return s.getEmbeddingsFrom(s.teiBaseURL, inputs)
}

func (s *Server) getEmbeddingsFrom(baseURL string, inputs []string) ([][]float64, error) {
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
		baseURL+"/embed",
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
	if s.multiModel {
		codeHealthy := s.checkTEIHealth(s.codeTEIBaseURL)
		textHealthy := s.checkTEIHealth(s.textTEIBaseURL)

		if !codeHealthy || !textHealthy {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "unhealthy",
				"embedding_models": map[string]interface{}{
					"code": map[string]interface{}{
						"model":   s.codeModel,
						"healthy": codeHealthy,
						"port":    s.codeTEIPort,
					},
					"text": map[string]interface{}{
						"model":   s.textModel,
						"healthy": textHealthy,
						"port":    s.textTEIPort,
					},
				},
			})
			return
		}

		response := map[string]interface{}{
			"status": "ok",
			"embedding_models": map[string]interface{}{
				"code": map[string]interface{}{
					"model":   s.codeModel,
					"healthy": true,
					"port":    s.codeTEIPort,
				},
				"text": map[string]interface{}{
					"model":   s.textModel,
					"healthy": true,
					"port":    s.textTEIPort,
				},
			},
		}

		if s.rerankerConfigured() {
			rerankHealthy := s.checkRerankHealth()
			response["reranker"] = map[string]interface{}{
				"enabled":       true,
				"healthy":       rerankHealthy,
				"model":         s.currentRerankModel,
				"port":          s.rerankPort,
				"switching":     s.rerankSwitching,
				"startup_model": s.rerankModel,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	s.mu.RLock()
	currentModel := s.currentModel
	isSwitching := s.switching
	s.mu.RUnlock()

	// Check if currently switching models
	if isSwitching {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":          "switching",
			"embedding_model": currentModel,
			"switching":       true,
		})
		return
	}

	// Check if embeddings TEI is healthy
	embeddingsHealthy := s.checkTEIHealth(s.teiBaseURL)

	if !embeddingsHealthy {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":          "unhealthy",
			"embedding_model": currentModel,
			"error":           "TEI is not responding",
		})
		return
	}

	// Build response with reranker status
	response := map[string]interface{}{
		"status":          "ok",
		"embedding_model": currentModel,
	}

	// Include reranker status if configured or currently loaded
	if s.rerankerConfigured() {
		rerankHealthy := s.checkRerankHealth()
		response["reranker"] = map[string]interface{}{
			"enabled":       true,
			"healthy":       rerankHealthy,
			"model":         s.currentRerankModel,
			"port":          s.rerankPort,
			"switching":     s.rerankSwitching,
			"startup_model": s.rerankModel, // Model from CLI flag (if any)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// resetIdleTimer resets the idle timer after each request
func (s *Server) resetIdleTimer() {
	if !s.idlePreload || s.multiModel {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastRequestTime = time.Now()

	// Stop existing timer if any
	if s.idleTimer != nil {
		s.idleTimer.Stop()
	}

	// Start new timer
	s.idleTimer = time.AfterFunc(s.idleTimeout, func() {
		s.onIdleTimeout()
	})
}

// onIdleTimeout is called when the system has been idle
func (s *Server) onIdleTimeout() {
	s.mu.RLock()
	currentModel := s.currentModel
	preferredModel := s.preferredModel
	s.mu.RUnlock()

	// If not on preferred model, switch to it
	if currentModel != preferredModel {
		log.Printf("System idle, preloading preferred model: %s", preferredModel)
		if err := s.switchModel(preferredModel); err != nil {
			log.Printf("Idle preload failed: %v", err)
		} else {
			log.Printf("Idle preload complete: %s ready for next run", preferredModel)
		}
	}
}

func (s *Server) checkTEIHealth(baseURL string) bool {
	resp, err := s.client.Get(baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ============================================================================
// Reranker TEI Management
// ============================================================================

// startRerankTEI starts the reranker TEI process
func (s *Server) startRerankTEI(ctx context.Context) error {
	// Reuse teiBinary - same TEI program, different model
	s.rerankCmd = exec.CommandContext(ctx, s.teiBinary,
		"--model-id", s.rerankModel,
		"--port", fmt.Sprintf("%d", s.rerankPort),
		"--max-batch-tokens", fmt.Sprintf("%d", s.maxBatchTokens),
	)
	s.rerankCmd.Stdout = os.Stdout
	s.rerankCmd.Stderr = os.Stderr

	if err := s.rerankCmd.Start(); err != nil {
		return fmt.Errorf("failed to start reranker TEI: %w", err)
	}

	log.Printf("Reranker TEI started with model %s (PID: %d, port: %d)",
		s.rerankModel, s.rerankCmd.Process.Pid, s.rerankPort)
	return nil
}

// stopRerankTEI gracefully stops the reranker TEI process
func (s *Server) stopRerankTEI() {
	if s.rerankCmd == nil || s.rerankCmd.Process == nil {
		return
	}

	log.Printf("Stopping reranker TEI process (PID: %d)", s.rerankCmd.Process.Pid)

	// Send SIGTERM for graceful shutdown
	if err := s.rerankCmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.Printf("Failed to send SIGTERM to reranker: %v", err)
		s.rerankCmd.Process.Kill()
		return
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- s.rerankCmd.Wait()
	}()

	select {
	case <-done:
		log.Printf("Reranker TEI stopped gracefully")
	case <-time.After(5 * time.Second):
		log.Printf("Reranker TEI didn't stop in time, killing...")
		s.rerankCmd.Process.Kill()
	}
}

// waitForRerankTEI waits for reranker TEI to be ready
func (s *Server) waitForRerankTEI(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := s.client.Get(s.rerankBaseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				s.rerankHealthy = true
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("reranker TEI did not become ready within %v", timeout)
}

func (s *Server) rerankerConfigured() bool {
	if s.currentRerankModel != "" || s.rerankModel != "" {
		return true
	}
	return s.config != nil && s.config.RerankModel != ""
}

// checkRerankHealth checks if reranker TEI is healthy
func (s *Server) checkRerankHealth() bool {
	if !s.rerankerConfigured() {
		return false // Reranker not configured
	}
	if s.currentRerankModel == "" && s.rerankModel == "" {
		s.rerankHealthy = false
		return false // Configured but not started yet
	}
	resp, err := s.client.Get(s.rerankBaseURL + "/health")
	if err != nil {
		s.rerankHealthy = false
		return false
	}
	defer resp.Body.Close()
	s.rerankHealthy = resp.StatusCode == http.StatusOK
	return s.rerankHealthy
}

// RerankRequest matches TEI /rerank request format
type RerankRequest struct {
	Query      string   `json:"query"`
	Texts      []string `json:"texts"`
	RawScores  bool     `json:"raw_scores,omitempty"`
	ReturnText bool     `json:"return_text,omitempty"`
	Model      string   `json:"model"` // Model ID for dynamic loading
}

// RerankResult matches TEI /rerank response format
type RerankResult struct {
	Index int     `json:"index"`
	Score float64 `json:"score"`
	Text  string  `json:"text,omitempty"`
}

// handleRerank handles POST /rerank requests by proxying to reranker TEI
func (s *Server) handleRerank(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request to get model
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request: %v", err), http.StatusBadRequest)
		return
	}

	var req RerankRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Determine which model to use (priority: request > config > CLI > error)
	requestedModel := req.Model
	if requestedModel == "" {
		// Fall back to config file
		if s.config != nil && s.config.RerankModel != "" {
			requestedModel = s.config.RerankModel
		} else if s.rerankModel != "" {
			// Fall back to CLI flag (deprecated)
			requestedModel = s.rerankModel
		} else {
			http.Error(w, "No reranker model specified. Configure in .code-scout.json or provide in request", http.StatusBadRequest)
			return
		}
	}

	// Check if we're currently switching models
	s.mu.RLock()
	isSwitching := s.rerankSwitching
	s.mu.RUnlock()

	if isSwitching {
		w.Header().Set("Retry-After", "5")
		http.Error(w, "Reranker model switch in progress, please retry", http.StatusServiceUnavailable)
		return
	}

	// Check if we need to switch models
	s.mu.RLock()
	needsSwitch := requestedModel != s.currentRerankModel
	s.mu.RUnlock()

	if needsSwitch {
		if err := s.switchRerankModel(requestedModel); err != nil {
			log.Printf("Reranker model switch failed: %v", err)
			http.Error(w, fmt.Sprintf("Model switch failed: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Check if reranker is healthy
	if !s.rerankHealthy && !s.checkRerankHealth() {
		w.Header().Set("Retry-After", "5")
		http.Error(w, "Reranker TEI is not available", http.StatusServiceUnavailable)
		return
	}

	// Proxy to reranker TEI
	resp, err := s.client.Post(
		s.rerankBaseURL+"/rerank",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		log.Printf("Reranker request failed: %v", err)
		s.rerankHealthy = false
		http.Error(w, fmt.Sprintf("Reranker request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Forward response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
