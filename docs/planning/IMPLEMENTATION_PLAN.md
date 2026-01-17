# Cross-Encoder Reranking Implementation Plan

## Overview

This plan implements proper cross-encoder reranking support per the [specification](SPECIFICATION.md). The tei-wrapper will be extended to manage a second TEI instance dedicated to reranking.

## Architecture Understanding

### Current State
- tei-wrapper manages **one** TEI instance (port 8080)
- Hot-swaps between code and text embedding models on that single instance
- This is the "single-model mode" in code-scout config

### Near-Term Goal (This Spec)
- tei-wrapper manages **two** TEI instances:
  1. **Embeddings TEI** (port 8080) - hot-swaps between code/text models
  2. **Reranker TEI** (port 8081) - dedicated reranker model, no swapping

### Future Vision (YAGNI - Not This Spec)
- tei-wrapper manages **three** TEI instances (one per model type)
- Eliminates hot-swapping entirely
- Enables memory pressure optimization (shut down idle instances)
- Foundation: state management being built now

---

## Already Complete (code_scout-9d1)

- `RerankClient` in [internal/embeddings/reranker.go](../../internal/embeddings/reranker.go)
- Config fields in [internal/config/config.go](../../internal/config/config.go)
- Search integration in [cmd/code-scout/search.go](../../cmd/code-scout/search.go)
- Factory in [cmd/code-scout/embeddings_factory.go](../../cmd/code-scout/embeddings_factory.go)

---

## Phase 1: tei-wrapper Reranker Management (code_scout-dpf)

### Goal
Extend tei-wrapper to spawn and manage a dedicated TEI instance for reranking, with proper state tracking.

### 1.1 New Command-Line Flags

**File:** `cmd/tei-wrapper/main.go`

Add flags:
```go
rerankPort := flag.Int("rerank-port", 8081, "Port for reranker TEI instance")
rerankModel := flag.String("rerank-model", "BAAI/bge-reranker-base", "Model ID for reranker")
```

**Note:** We reuse the existing `--tei-binary` flag for spawning the reranker TEI process. There's no need for a separate `--rerank-binary` flag since it's the same TEI program - only the model differs.

### 1.2 Extend Server Struct for Reranker State

**File:** `cmd/tei-wrapper/main.go`

```go
type Server struct {
    // Existing embeddings TEI management
    teiPort      int
    teiBinary    string
    initialModel string
    currentModel string
    teiCmd       *exec.Cmd
    teiBaseURL   string
    client       *http.Client
    mu           sync.RWMutex
    switching    bool

    // NEW: Reranker TEI management
    rerankPort    int           // Port for reranker TEI (default: 8081)
    rerankModel   string        // Model ID for reranker
    rerankCmd     *exec.Cmd     // Reranker TEI process
    rerankBaseURL string        // http://localhost:{rerankPort}
    rerankHealthy bool          // Current health status

    // Idle preloading (existing)
    idlePreload     bool
    idleTimeout     time.Duration
    lastRequestTime time.Time
    preferredModel  string
    idleTimer       *time.Timer

    // Memory management (existing)
    maxBatchTokens int
}
```

### 1.3 Reranker Process Management Methods

**File:** `cmd/tei-wrapper/main.go`

Add new methods:

```go
// startRerankTEI starts the reranker TEI process
func (s *Server) startRerankTEI(ctx context.Context) error {
    // Reuse teiBinary - same TEI program, different model
    s.rerankCmd = exec.CommandContext(ctx, s.teiBinary,
        "--model-id", s.rerankModel,
        "--port", fmt.Sprintf("%d", s.rerankPort),
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

    log.Printf("Stopping reranker TEI (PID: %d)", s.rerankCmd.Process.Pid)

    // Send SIGTERM, wait with timeout, then SIGKILL if needed
    // (same pattern as stopTEI)
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

// checkRerankHealth checks if reranker TEI is healthy
func (s *Server) checkRerankHealth() bool {
    resp, err := s.client.Get(s.rerankBaseURL + "/health")
    if err != nil {
        s.rerankHealthy = false
        return false
    }
    defer resp.Body.Close()
    s.rerankHealthy = resp.StatusCode == http.StatusOK
    return s.rerankHealthy
}
```

### 1.4 Update main() for Dual Process Startup

**File:** `cmd/tei-wrapper/main.go`

Update startup sequence:
```go
func main() {
    // ... flag parsing ...

    server := &Server{
        // Existing fields...

        // NEW: Reranker config
        rerankPort:    *rerankPort,
        rerankModel:   *rerankModel,
        rerankBaseURL: fmt.Sprintf("http://localhost:%d", *rerankPort),
    }

    // Start embeddings TEI (existing)
    log.Printf("Starting embeddings TEI with model: %s", server.initialModel)
    if err := server.startTEIWithModel(context.Background(), server.initialModel); err != nil {
        log.Fatalf("Failed to start embeddings TEI: %v", err)
    }
    defer server.stopTEI()

    // NEW: Start reranker TEI
    log.Printf("Starting reranker TEI with model: %s", server.rerankModel)
    if err := server.startRerankTEI(context.Background()); err != nil {
        log.Fatalf("Failed to start reranker TEI: %v", err)
    }
    defer server.stopRerankTEI()

    // Wait for both to be ready
    log.Printf("Waiting for embeddings TEI...")
    if err := server.waitForTEI(90 * time.Second); err != nil {
        log.Fatalf("Embeddings TEI failed to start: %v", err)
    }

    log.Printf("Waiting for reranker TEI...")
    if err := server.waitForRerankTEI(90 * time.Second); err != nil {
        log.Fatalf("Reranker TEI failed to start: %v", err)
    }

    log.Printf("Both TEI instances ready!")

    // Setup HTTP routes
    mux := http.NewServeMux()
    mux.HandleFunc("/v1/embeddings", server.handleEmbeddings)
    mux.HandleFunc("/rerank", server.handleRerank)  // NEW
    mux.HandleFunc("/health", server.handleHealth)

    // ... rest of main() ...
}
```

### 1.5 Implement /rerank Endpoint Handler

**File:** `cmd/tei-wrapper/main.go`

Add types and handler:
```go
// RerankRequest matches TEI /rerank request format
type RerankRequest struct {
    Query      string   `json:"query"`
    Texts      []string `json:"texts"`
    RawScores  bool     `json:"raw_scores,omitempty"`
    ReturnText bool     `json:"return_text,omitempty"`
}

// RerankResult matches TEI /rerank response format
type RerankResult struct {
    Index int     `json:"index"`
    Score float64 `json:"score"`
    Text  string  `json:"text,omitempty"`
}

func (s *Server) handleRerank(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Check if reranker is healthy
    if !s.rerankHealthy && !s.checkRerankHealth() {
        w.Header().Set("Retry-After", "5")
        http.Error(w, "Reranker TEI is not available", http.StatusServiceUnavailable)
        return
    }

    // Read request body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to read request: %v", err), http.StatusBadRequest)
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
```

### 1.6 Update Health Check Response Format

**File:** `cmd/tei-wrapper/main.go`

Update `/health` to include reranker status:
```go
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    s.mu.RLock()
    currentModel := s.currentModel
    isSwitching := s.switching
    s.mu.RUnlock()

    // Check embeddings TEI health
    embeddingsHealthy := false
    if !isSwitching {
        resp, err := s.client.Get(s.teiBaseURL + "/health")
        if err == nil {
            resp.Body.Close()
            embeddingsHealthy = resp.StatusCode == http.StatusOK
        }
    }

    // Check reranker TEI health
    rerankHealthy := s.checkRerankHealth()

    // Determine overall status
    status := "ok"
    httpStatus := http.StatusOK
    if isSwitching {
        status = "switching"
        httpStatus = http.StatusServiceUnavailable
    } else if !embeddingsHealthy {
        status = "unhealthy"
        httpStatus = http.StatusServiceUnavailable
    }

    response := map[string]interface{}{
        "status":          status,
        "embedding_model": currentModel,
        "reranker": map[string]interface{}{
            "enabled": true,
            "healthy": rerankHealthy,
            "model":   s.rerankModel,
            "port":    s.rerankPort,
        },
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(httpStatus)
    json.NewEncoder(w).Encode(response)
}
```

### 1.7 Remove rerank_endpoint from code-scout Config

**File:** `internal/config/config.go`

Remove field:
```go
type Config struct {
    Endpoint        string `json:"endpoint"`
    APIKey          string `json:"api_key,omitempty"`
    CodeModel       string `json:"code_model"`
    TextModel       string `json:"text_model"`
    RerankModel     string `json:"rerank_model,omitempty"`
    // DELETE: RerankEndpoint  string `json:"rerank_endpoint,omitempty"`
    RerankTopK      int    `json:"rerank_top_k,omitempty"`
    SingleModelMode *bool  `json:"single_model_mode,omitempty"`
}
```

Remove from `mergeConfig()`:
```go
// DELETE these lines:
// if src.RerankEndpoint != "" {
//     dst.RerankEndpoint = src.RerankEndpoint
// }
```

**File:** `cmd/code-scout/embeddings_factory.go`

Simplify `newRerankClient`:
```go
newRerankClient = func() *embeddings.RerankClient {
    if globalConfig != nil && globalConfig.RerankModel != "" {
        // Always use main endpoint - tei-wrapper handles routing
        return embeddings.NewRerankClient(globalConfig.Endpoint, globalConfig.APIKey, globalConfig.RerankModel)
    }
    return nil
}
```

### 1.8 Error Handling Matrix

| Condition | HTTP Status | Response |
|-----------|-------------|----------|
| Reranker healthy | 200 | TEI response (proxied) |
| Reranker TEI down | 503 | "Reranker TEI is not available" + Retry-After: 5 |
| TEI returns error | 502 | "Reranker request failed: {error}" |
| Invalid request | 400 | "Failed to read request: {error}" |
| Wrong HTTP method | 405 | "Method not allowed" |

---

## Phase 2: End-to-End Testing (code_scout-i5k)

### Goal
Verify reranking works correctly with real cross-encoder models.

### 2.1 Test Environment Setup

**Start tei-wrapper (manages both TEI instances):**
```bash
cd cmd/tei-wrapper
go build -o tei-wrapper .
./tei-wrapper \
    --port 11434 \
    --tei-port 8080 \
    --model nomic-ai/nomic-embed-text-v1.5 \
    --rerank-port 8081 \
    --rerank-model BAAI/bge-reranker-base
```

**Configure code-scout:**
```json
{
  "endpoint": "http://localhost:11434",
  "code_model": "nomic-ai/CodeRankEmbed",
  "text_model": "nomic-ai/nomic-embed-text-v1.5",
  "rerank_model": "BAAI/bge-reranker-base",
  "rerank_top_k": 25
}
```

### 2.2 Test Cases

#### Primary Test: Default Model (BAAI/bge-reranker-base)

```bash
# Index test repo
./dist/code-scout-darwin_arm64/code-scout index

# Run test searches
./dist/code-scout-darwin_arm64/code-scout search "authentication logic" --limit 10
./dist/code-scout-darwin_arm64/code-scout search "error handling" --limit 10 --json
```

**Verify:**
- [ ] No errors in output
- [ ] Results show both vector and rerank scores
- [ ] Results sorted by rerank score (highest first)
- [ ] Output indicates reranking model name

#### Model Compatibility Matrix

| Model | Status | Notes |
|-------|--------|-------|
| BAAI/bge-reranker-base | MUST PASS | Default model |
| BAAI/bge-reranker-large | Test | Higher accuracy |
| BAAI/bge-reranker-v2-m3 | Test | Multilingual |
| cross-encoder/ms-marco-MiniLM-L-6-v2 | Test | Lightweight |
| jinaai/jina-reranker-v1-turbo-en | Test | Speed-optimized |

**Success Criteria:**
- [ ] Default model works end-to-end
- [ ] At least 3 of 5 models work
- [ ] Failed models documented with reasons
- [ ] No crashes or hangs

#### Test Queries
1. "authentication logic"
2. "error handling"
3. "database connection"
4. "API endpoint definitions"
5. "test fixtures"

### 2.3 Health Check Verification

```bash
# Verify health endpoint returns correct format
curl http://localhost:11434/health | jq

# Expected response:
{
  "status": "ok",
  "embedding_model": "nomic-ai/nomic-embed-text-v1.5",
  "reranker": {
    "enabled": true,
    "healthy": true,
    "model": "BAAI/bge-reranker-base",
    "port": 8081
  }
}
```

---

## Phase 3: Documentation (code_scout-pvs)

### Goal
Create comprehensive documentation for deploying and using reranking.

### 3.1 Create docs/guides/RERANKER_SETUP.md

**Structure:**

1. **Overview**
   - What is cross-encoder reranking
   - When to use it (accuracy vs latency trade-off)
   - Memory requirements

2. **Quick Start**
   ```bash
   # Build tei-wrapper
   cd cmd/tei-wrapper && go build -o tei-wrapper .

   # Start with reranking enabled (both TEI instances)
   ./tei-wrapper --rerank-model BAAI/bge-reranker-base

   # Configure code-scout
   cat > .code-scout.json << 'EOF'
   {
     "endpoint": "http://localhost:11434",
     "code_model": "nomic-ai/CodeRankEmbed",
     "text_model": "nomic-ai/nomic-embed-text-v1.5",
     "rerank_model": "BAAI/bge-reranker-base",
     "rerank_top_k": 25
   }
   EOF

   # Search with reranking
   code-scout search "authentication" --limit 10
   ```

3. **Model Selection Guide**
   - Table of models with size, speed, accuracy trade-offs
   - Memory requirements per model
   - Recommendations by use case

4. **Platform-Specific Instructions**
   - macOS (Homebrew TEI, Metal acceleration)
   - Linux (native TEI, CUDA)
   - Docker deployment option

5. **Configuration Reference**
   - tei-wrapper flags: `--rerank-port`, `--rerank-model`
   - code-scout config: `rerank_model`, `rerank_top_k`

6. **Performance Tuning**
   - `rerank_top_k` guidance (balance accuracy vs latency)
   - When to disable reranking
   - Memory optimization

7. **Troubleshooting**
   - Common errors and solutions
   - Health check interpretation
   - Log analysis

### 3.2 Update docs/guides/TEI_WRAPPER.md

Add sections:
- New `/rerank` endpoint documentation
- New command-line flags: `--rerank-port`, `--rerank-model`
- Dual TEI process architecture explanation
- Updated health check response format

### 3.3 Update docs/guides/README.md

Add RERANKER_SETUP.md to the guides index.

### 3.4 Update README.md

**Add to Features:**
- "Cross-encoder reranking for improved search relevance"

**Update Configuration section:**
- Add reranking config example
- Remove `rerank_endpoint` from examples
- Link to RERANKER_SETUP.md

**Update Self-Hosting section:**
- Note that tei-wrapper now manages reranker TEI
- Link to setup guide

---

## Files to Modify

| File | Changes |
|------|---------|
| `cmd/tei-wrapper/main.go` | Add reranker process management, `/rerank` handler, health check updates |
| `internal/config/config.go` | Remove `RerankEndpoint` field |
| `cmd/code-scout/embeddings_factory.go` | Simplify `newRerankClient` |
| `docs/guides/TEI_WRAPPER.md` | Add reranker documentation |
| `docs/guides/README.md` | Add RERANKER_SETUP.md to index |
| `README.md` | Add reranking feature, update config examples |

## Files to Create

| File | Purpose |
|------|---------|
| `docs/guides/RERANKER_SETUP.md` | Complete reranker deployment guide |

---

## Implementation Order

```
1. Phase 1.1-1.5: tei-wrapper reranker management
       │
       ▼
2. Phase 1.6: Health check updates
       │
       ▼
3. Phase 1.7: Remove rerank_endpoint from code-scout
       │
       ▼
4. Phase 2: End-to-end testing
       │
       ▼
5. Phase 3: Documentation
```

---

## Acceptance Criteria

Per the specification:

**Phase 1 - tei-wrapper Implementation (COMPLETE)**
- [x] tei-wrapper spawns and manages reranker TEI process
- [x] tei-wrapper has `/rerank` endpoint that proxies to managed reranker TEI
- [x] New flags: `--rerank-port`, `--rerank-model`
- [x] Health check includes reranker status with new response format
- [x] `rerank_endpoint` config field removed from code-scout

**Phase 2 - End-to-end Testing (COMPLETE)**
- [x] End-to-end testing passes with BAAI/bge-reranker-base
- [x] Testing validated with multiple queries from test suite
- [x] Identified and fixed truncation issue for long chunks (512 token limit)
- [ ] Additional models tested and documented (deferred to Phase 3 docs)

**Phase 3 - Documentation (PENDING)**
- [ ] RERANKER_SETUP.md guide published
- [ ] TEI_WRAPPER.md updated with reranker docs
- [ ] README.md updated with reranking examples

---

## Phase 2 Testing Results

### Successful E2E Validation

Testing confirmed end-to-end reranking functionality works correctly:

**✅ Health Check:**
```json
{
  "embedding_model": "nomic-ai/nomic-embed-text-v1.5",
  "reranker": {
    "enabled": true,
    "healthy": true,
    "model": "BAAI/bge-reranker-base",
    "port": 8081
  },
  "status": "ok"
}
```

**✅ Search Results:**
- Results show both `score` (vector similarity) and `rerank_score` (cross-encoder)
- Results correctly sorted by rerank score (highest first)
- Output indicates reranking model name and top_k setting
- Tested queries: "tree-sitter parsing", "authentication logic", "error handling", "API endpoint definitions"

### Issue Found: Token Limit Truncation

**Problem:** BAAI/bge-reranker-base has a hard 512 token limit. Large documentation chunks (>1200 chars) exceeded this limit and caused validation errors:
```
Input validation error: `inputs` must have less than 512 tokens. Given: 744
```

**Solution:** Implemented text truncation in `buildRerankText()` (search.go:417-423):
- Truncates code content to 1200 characters
- Preserves metadata (file path, language, chunk type, heading)
- Prevents errors but may impact semantic quality for long chunks

**Trade-off:** Truncation ensures functionality but risks losing context for large chunks.

**Recommendations:**
1. **Use higher-limit models** for production (documented in Phase 3):
   - `BAAI/bge-reranker-v2-m3`: 8192 tokens
   - `jinaai/jina-reranker-v1-turbo-en`: 8192 tokens
2. **Or configure smaller `rerank_top_k`** to avoid reranking very large chunks
3. **Or use `--code` mode** which has naturally smaller chunks

### Dual TEI Instances Validated

**✅ Confirmed:**
- Both TEI instances start successfully without port conflicts
- Embeddings TEI on port 8080, Reranker TEI on port 8081
- Memory usage is additive (~800MB total for both models)
- No resource contention observed (Metal GPU, CPU)
- Both health checks respond independently
- Hot-swapping embeddings model does not affect reranker

---

## Remaining Assumptions & Known Limitations

**Token Limit Truncation:**
- BAAI/bge-reranker-base limited to 512 tokens (~1200 chars with metadata)
- Large chunks are truncated, potentially affecting semantic quality
- Workaround: Use higher-limit models (bge-reranker-v2-m3, jina-reranker-v1-turbo-en)
- See Phase 2 Testing Results above for details

**Model Compatibility:**
- Only BAAI/bge-reranker-base tested in Phase 2
- Other models (bge-reranker-large, jina-reranker-v1-turbo-en, etc.) should be tested and documented in Phase 3

---

## Future Considerations (Not This Spec)

The state management being built here lays groundwork for:

1. **Three-instance mode** - Separate TEI for code, text, and reranker models
2. **Memory pressure optimization** - Shut down idle TEI instances
3. **Dynamic model switching** - Change models without full restart

These are explicitly out of scope for this implementation per the spec's Non-Goals section.
