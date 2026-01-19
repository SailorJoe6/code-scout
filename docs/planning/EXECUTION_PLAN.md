# Implementation Plan: Reranker Dynamic Model Loading

**Specification**: [SPECIFICATION.md](SPECIFICATION.md)
**Created**: 2026-01-18
**Status**: Ready for Implementation

## Overview

Implement dynamic reranker model loading in tei-wrapper to match the architecture used for embedding models. This eliminates the need for `--rerank-model` CLI flags and makes `.code-scout.json` the single source of truth for all model configuration.

## Prerequisites

Before implementing dynamic model loading, we must verify which reranker models actually work with TEI.
This testing is **critical** and must be performed (not optional), since model compatibility is a blocker for implementation and documentation.
When testing tei-wrapper locally, start tei-wrapper locally and set the `endpoint` in `.code-scout.json` to that wrapper (default `http://localhost:11435`).

### Phase 0: Troubleshoot Reranker Model Loading

**Goal**: Understand why `BAAI/bge-reranker-base` fails to load despite being documented as supported by TEI, and determine the correct way to use reranker models.

**Problem**:
ALL tested reranker models fail to load with identical 404 errors:
- `BAAI/bge-reranker-base`
- `BAAI/bge-reranker-large`
- `Alibaba-NLP/gte-multilingual-reranker-base`
- `Alibaba-NLP/gte-reranker-modernbert-base`

Error message:
```
WARN: Download failed: request error: HTTP status client error (404 Not Found)
for url (https://huggingface.co/<model>/resolve/main/config_sentence_transformers.json)
```

**Critical Finding**: Since ALL models fail identically (across different families/providers), this is almost certainly an issue with how we're invoking TEI or a fundamental configuration problem with TEI 1.8.3, NOT an issue with the models themselves.

**Immediate Actions to Try**:

```bash
# 1. Check TEI help for reranker-specific flags
text-embeddings-router --help | grep -i rerank

# 2. Try running TEI directly (bypass tei-wrapper)
text-embeddings-router --model-id BAAI/bge-reranker-base --port 8080

# 3. Check TEI version and features
text-embeddings-router --version
brew info text-embeddings-inference  # Check if newer version available

# 4. Compare with working embedding model command
text-embeddings-router --model-id nomic-ai/nomic-embed-text-v1.5 --port 8080  # This works
```

**Local Testing Requirement**:
- If validating via code-scout + tei-wrapper, tei-wrapper must be running locally and `.code-scout.json` must point to its `endpoint` (default `http://localhost:11435`).

**Local Findings (2026-01-18, sandboxed environment)**:
- `text-embeddings-router --version` reports `1.8.3`
- `text-embeddings-router --help` shows no rerank-specific flags (no `--task`/`--rerank`/`--model-type`)
- Attempting to start the TEI server (embedding or reranker model) aborts with `Signal 6` in this sandbox, so model downloads and `/rerank` behavior were not validated here
- This phase should be rerun on a machine with network access and a working TEI server process (sandbox limitation, not a TEI limitation)
  - Local HF cache has `models--BAAI--bge-reranker-base` but only `config.json`, `model.safetensors`, and `tokenizer.json` (no `config_sentence_transformers.json`)
  - `text-embeddings-router --model-id <local snapshot path>` for bge-reranker-base still aborts with `Signal 6` in this sandbox
  - SentenceTransformers caches (e.g., `sentence-transformers/all-MiniLM-L6-v2`) include `config_sentence_transformers.json`, suggesting TEI may be expecting that file

**Local Findings (2026-01-19, sandboxed environment)**:
- TEI can run locally when the Hugging Face cache is redirected to a writable path: `--huggingface-hub-cache /tmp/hf`.
- `BAAI/bge-reranker-base` works end-to-end:
  - Command: `text-embeddings-router --model-id BAAI/bge-reranker-base --port 8090 --huggingface-hub-cache /tmp/hf --json-output`
  - `/health` returns 200 and `/rerank` returns scores.
  - Non-fatal log line: `nice(5) failed: operation not permitted`.
- Missing `config_sentence_transformers.json` is logged as a warning and does not block startup for reranker models.
- In this sandboxed environment, the following reranker models consistently panic after downloading `modules.json` (TEI 1.8.3):
  - `BAAI/bge-reranker-large`
  - `BAAI/bge-reranker-v2-m3`
  - `Alibaba-NLP/gte-multilingual-reranker-base`
  - `Alibaba-NLP/gte-reranker-modernbert-base`
  - `cross-encoder/ms-marco-MiniLM-L-6-v2`
  - Panic signature: `swap_remove index (is 0) should be < len (is 0)` after `Dense modules downloaded`.
- Conclusion (sandboxed): TEI 1.8.3 reliably serves `bge-reranker-base` but crashes on the other tested reranker models in this sandbox. See unsandboxed update below.

**Local Findings (2026-01-19, follow-up in sandboxed environment on this host)**:
- With `--huggingface-hub-cache /tmp/hf`, every reranker model tested panics after `Dense modules downloaded` with `swap_remove index (is 0) should be < len (is 0)`.
  - Confirmed panics: `BAAI/bge-reranker-base`, `BAAI/bge-reranker-large`, `cross-encoder/ms-marco-MiniLM-L-6-v2`.
  - Logs captured at `/tmp/tei-rerank-base-run.log`, `/tmp/tei-rerank-large.log`, `/tmp/tei-rerank-marco-l6.log`.
- `mixedbread-ai/mxbai-rerank-xsmall-v1` fails earlier: TEI backend rejects `deberta-v2` as unsupported architecture.
  - Log captured at `/tmp/tei-rerank-mxbai-xsmall.log`.
- Hugging Face model repos for the above rerankers do not expose `modules.json` (API and `raw` lookups return "Entry not found"), which may be related to the panic.
- Net (sandboxed): TEI 1.8.3 does not successfully start any reranker model on this machine; Phase 0 was blocked in the sandbox. See unsandboxed update below.
- Repro (2026-01-19, follow-up): `RUST_BACKTRACE=full text-embeddings-router --model-id BAAI/bge-reranker-base --port 8091 --huggingface-hub-cache /tmp/hf --json-output` still panics after `Dense modules downloaded`.
  - Backtrace is address-only (release binary), but confirms the crash path; logs captured at `/tmp/tei-rerank-base-2026-01-19.log` and `/tmp/tei-rerank-base-backtrace-full-2026-01-19.log`.
- Repro (2026-01-19, this session): direct TEI runs for both reranker and embedding models panic immediately after `Dense modules downloaded`.
  - `RUST_BACKTRACE=full text-embeddings-router --model-id BAAI/bge-reranker-base --port 8090 --huggingface-hub-cache /tmp/hf --json-output` → panic; log `/tmp/tei-rerank-base.log`.
  - `RUST_BACKTRACE=full text-embeddings-router --model-id nomic-ai/nomic-embed-text-v1.5 --port 8091 --huggingface-hub-cache /tmp/hf --json-output` → panic; log `/tmp/tei-embed-nomic.log`.
  - Adding `--dense-path /tmp/does-not-exist` does not change behavior; log `/tmp/tei-embed-nomic-dense.log`.
- GitHub releases show `v1.8.3` as the latest TEI release (Homebrew is current). No newer release is available to validate yet.
- Related upstream PRs: #701 (parse `modules.json` for Dense modules) and #738 (fix reading `modules.json` for local models) are merged pre-`v1.8.3`, but the panic persists; likely a separate issue worth filing upstream with logs.
- Attempting to run `tei-wrapper` locally for indexing (model `nomic-ai/nomic-embed-text-v1.5`) also panics after `Dense modules downloaded`, blocking `code-scout index` on this host.
  - Log captured at `/tmp/tei-wrapper-20260119101538.log`.
  - Nomic model `modules.json` contains no `Dense` entries, suggesting TEI 1.8.3 may crash when no Dense modules are present (not just reranker models).
- Additional local evidence (2026-01-19):
  - `BAAI/bge-reranker-base` snapshot in `/tmp/hf` contains only `config.json`, `model.safetensors`, and `tokenizer.json` (no `modules.json` or `2_Dense` directory).
  - `nomic-ai/nomic-embed-text-v1.5` `modules.json` contains only Transformer/Pooling entries (no Dense modules), yet TEI still downloads `modules.json` before panicking.
  - `rg --files -g '2_Dense*' /tmp/hf` returned no matches.
  - Backtrace run log captured at `/tmp/tei-rerank-base-backtrace.log`.

**Local Findings (2026-01-19, sandboxed environment on this host)**:
- `text-embeddings-router` panics immediately at startup (before model load) with:
  `system-configuration-0.6.1/src/dynamic_store.rs:154: Attempted to create a NULL object.`
- Repro commands (both panic):
  - `text-embeddings-router --model-id BAAAI/bge-reranker-base --port 8090 --huggingface-hub-cache /Users/jlanders/.cache/huggingface/hub --json-output`
  - `text-embeddings-router --model-id nomic-ai/nomic-embed-text-v1.5 --port 8091 --huggingface-hub-cache /Users/jlanders/.cache/huggingface/hub --json-output`
- Logs captured at `/tmp/tei-rerank-base.log` and `/tmp/tei-embed-nomic.log`.
- Likely another sandbox-specific failure mode (SystemConfiguration API), preventing TEI from reaching the earlier "Dense modules downloaded" panic on this host.

**Local Findings (2026-01-20, unsandboxed environment)**:
- `text-embeddings-router --model-id BAAI/bge-reranker-large --port 8080` starts cleanly; `/health` returns 200 and `/rerank` returns scores.
- `text-embeddings-router --model-id BAAI/bge-reranker-v2-m3 --port 8080` starts cleanly; `/health` returns 200 and `/rerank` returns scores.
- User report: `BAAI/bge-reranker-base` also starts without panic in the unsandboxed environment.
- Hypothesis: prior crashes/Signal 6 were sandbox artifacts; rerun remaining model tests outside the sandbox.

**Root Cause Investigation** (Focus: How we're using TEI):

1. **Verify TEI binary and invocation**
   - Are we using the correct TEI binary for reranking? (`text-embeddings-router` vs something else?)
   - Is there a separate binary for reranking vs embeddings?
   - Check TEI CLI help: `text-embeddings-router --help` for reranker-specific flags
   - Verify if reranker models require different command-line arguments
   - Check if we need to specify `--model-type` or similar flag

2. **Test TEI directly (bypass tei-wrapper)**
   ```bash
   # Test running TEI directly without our wrapper
   text-embeddings-router --model-id BAAI/bge-reranker-base --port 8080

   # Try with explicit flags that might be needed for rerankers
   text-embeddings-router --model-id BAAI/bge-reranker-base --port 8080 [other flags?]
   ```
   - Document exact command that works (if any)
   - Compare successful embedding model invocation vs reranker invocation

3. **Check TEI documentation and examples**
   - Read TEI docs for reranker setup examples
   - Find working examples from TEI project or community
   - Check if reranking requires different API endpoints
   - Verify if TEI 1.8.3 actually supports reranking (might be newer feature)
   - Look for TEI GitHub issues related to reranker model loading

4. **Verify TEI version compatibility**
   - Check TEI release notes for version 1.8.3
   - Determine when reranker support was added
   - Check if we need a newer/older version
   - Look for breaking changes in model loading between versions

5. **Analyze the 404 error in detail**
   - Why is TEI looking for `config_sentence_transformers.json`?
   - Is this file required for rerankers but not embeddings?
   - Check if embedding models we successfully use have this file
   - Determine if this is a TEI configuration issue or missing feature

6. **Check tei-wrapper implementation**
   - Review how we're calling TEI for rerankers in [cmd/tei-wrapper/main.go:498-515](../../cmd/tei-wrapper/main.go#L498-L515)
   - Compare with how we call TEI for embeddings in [cmd/tei-wrapper/main.go:179-198](../../cmd/tei-wrapper/main.go#L179-L198)
   - Are we missing flags or arguments for reranker mode?
   - Is our command-line construction correct?

**Deliverables**:
- ✅ Clear understanding of WHY ALL reranker models fail identically
- ✅ Correct TEI invocation method for reranker models
- ✅ Working command to load at least one reranker model
- ✅ List of verified-working reranker models with correct TEI configuration
- ✅ Recommended model(s) for documentation and tests
- ✅ Documentation of correct TEI flags/arguments for rerankers
- ✅ Decision on TEI version (upgrade/downgrade if needed)
- ✅ Fix to tei-wrapper implementation if we're invoking TEI incorrectly

**Success Criteria**:
- Root cause identified: either wrong TEI flags, wrong binary, version incompatibility, or missing configuration
- At least 2 different reranker models verified working with correct TEI invocation
- Clear documented procedure for starting TEI with reranker models
- tei-wrapper updated to use correct TEI invocation (if needed)
- Understanding of differences between embedding model and reranker model TEI setup

**Blocker**: This phase MUST be completed before implementing dynamic loading. We need to understand correct model usage patterns and identify reliable models for testing.

---

## Implementation Phases

### Phase 1: Add Config File Support to tei-wrapper ✅ COMPLETE

**Status**: Complete (2026-01-19)

**Goal**: Make tei-wrapper read `.code-scout.json` for model configuration, with sensible defaults. Also enhance the config package to search up directory tree (like git) and warn when using defaults.

**Changes Required**:

1. **Enhance config package with directory tree search** ([internal/config/config.go:35-53](../../internal/config/config.go#L35-L53))
   ```go
   // Load loads configuration from file paths in order of precedence:
   // 1. Project-level: .code-scout.json (searches up directory tree from PWD)
   // 2. User-level: ~/.code-scout/config.json
   // If no config file exists, returns default config with warning
   func Load() (*Config, error) {
       cfg := Default()
       foundConfig := false

       // Try user-level config first
       if userConfig, err := loadUserConfig(); err == nil && userConfig != nil {
           mergeConfig(cfg, userConfig)
           foundConfig = true
       }

       // Try project-level config (searches up from PWD)
       if projectConfig, configPath, err := loadProjectConfig(); err == nil && projectConfig != nil {
           mergeConfig(cfg, projectConfig)
           log.Printf("Loaded config from: %s", configPath)
           foundConfig = true
       }

       // Warn if no config found
       if !foundConfig {
           log.Println("WARNING: No .code-scout.json found (searched up directory tree) and no ~/.code-scout/config.json found. Using default configuration.")
       }

       return cfg, nil
   }

   // loadProjectConfig searches up the directory tree for .code-scout.json
   func loadProjectConfig() (*Config, string, error) {
       currentDir, err := os.Getwd()
       if err != nil {
           return nil, "", err
       }

       // Search up directory tree (like git does for .git/)
       for {
           configPath := filepath.Join(currentDir, ".code-scout.json")

           // Check if config exists
           if _, err := os.Stat(configPath); err == nil {
               cfg, err := loadFromFile(configPath)
               return cfg, configPath, err
           }

           // Move to parent directory
           parent := filepath.Dir(currentDir)
           if parent == currentDir {
               // Reached filesystem root
               break
           }
           currentDir = parent
       }

       return nil, "", nil // Not found
   }
   ```

2. **Import config package in tei-wrapper** ([cmd/tei-wrapper/main.go:1-18](../../cmd/tei-wrapper/main.go#L1-L18))
   ```go
   import (
       // ... existing imports ...
       "github.com/jlanders/code-scout/internal/config"
   )
   ```

3. **Add config loading at startup** ([cmd/tei-wrapper/main.go:83-116](../../cmd/tei-wrapper/main.go#L83-L116))
   ```go
   func main() {
       // Load config from .code-scout.json (or use defaults)
       cfg, err := config.Load()
       if err != nil {
           log.Fatalf("Failed to load config: %v", err)
       }

       // Command line flags (with config defaults)
       port := flag.Int("port", 11435, "Port to listen on")
       teiPort := flag.Int("tei-port", 8080, "TEI internal port")
       teiBinary := flag.String("tei-binary", "text-embeddings-router", "Path to TEI binary")

       // Model flags with config file as defaults (CLI overrides config)
       model := flag.String("model", cfg.TextModel, "Initial embedding model")
       rerankModel := flag.String("rerank-model", cfg.RerankModel, "Reranker model (empty = load on-demand from config)")

       // ... rest of flags ...
       flag.Parse()

       // Priority: CLI flag > config file > hardcoded default
       initialModel := *model
       if initialModel == "" {
           initialModel = "nomic-ai/nomic-embed-text-v1.5" // Hardcoded fallback
       }

       // Create server with config
       server := &Server{
           config:       cfg,  // Add config to server struct
           initialModel: initialModel,
           rerankModel:  *rerankModel,
           // ... rest of fields ...
       }
   }
   ```

3. **Update Server struct** ([cmd/tei-wrapper/main.go:54-81](../../cmd/tei-wrapper/main.go#L54-L81))
   ```go
   type Server struct {
       config       *config.Config  // ADD: Config from .code-scout.json
       // ... existing fields ...
   }
   ```

4. **Define default models** (add constants at top of file)
   ```go
   const (
       DefaultTextModel    = "nomic-ai/nomic-embed-text-v1.5"
       DefaultCodeModel    = "nomic-ai/CodeRankEmbed"
       DefaultRerankModel  = ""  // Empty = no default, load on-demand
   )
   ```

**Testing**:
- ✅ Unit test: Config file loads from PWD
- ✅ Unit test: Config file found by searching up directory tree (e.g., run from subdirectory)
- ✅ Unit test: Config file not found, defaults used with warning message
- ✅ Unit test: User-level config (~/.code-scout/config.json) loads correctly
- ✅ Unit test: Project-level config overrides user-level config
- ✅ Unit test: CLI flags override config file values
- ✅ Integration test: `./tei-wrapper` works without any flags
- ✅ Integration test: tei-wrapper with `.code-scout.json` uses configured models
- ✅ Integration test: Warning appears when no config found
- ✅ Integration test: tei-wrapper finds config from subdirectory

**Success Criteria**:
- tei-wrapper reads `.code-scout.json` at startup
- Config search works from any subdirectory (searches up tree like git)
- Warning logged when no config found and defaults used
- Log message shows which config file was loaded (with path)
- Sensible defaults when config file doesn't exist
- CLI flags still work (backward compatibility)
- Priority order: Request > Config > CLI > Default
- Both code-scout and tei-wrapper use same config loading logic

---

### Phase 2: Update Request/Response Formats ✅ COMPLETE

**Status**: Complete (2026-01-19)

**Goal**: Add model field to rerank requests so clients can specify which model to use.

**Changes Required**:

1. **Update RerankRequest struct in client** ([internal/embeddings/reranker.go:20-26](../../internal/embeddings/reranker.go#L20-L26))
   ```go
   type RerankRequest struct {
       Query      string   `json:"query"`
       Texts      []string `json:"texts"`
       RawScores  bool     `json:"raw_scores,omitempty"`
       ReturnText bool     `json:"return_text,omitempty"`
       Model      string   `json:"model"`  // ADD THIS
   }
   ```

2. **Update RerankRequest struct in server** ([cmd/tei-wrapper/main.go:580-585](../../cmd/tei-wrapper/main.go#L580-L585))
   - Same change as above for consistency

3. **Update client to send model in request** ([internal/embeddings/reranker.go:75-80](../../internal/embeddings/reranker.go#L75-L80))
   ```go
   reqBody := RerankRequest{
       Query:      query,
       Texts:      texts,
       RawScores:  false,
       ReturnText: false,
       Model:      c.model,  // ADD THIS - send model from config
   }
   ```

**Testing**:
- ✅ Unit test: RerankRequest marshals/unmarshals with model field
- ✅ Unit test: Client includes model in request body

**Implementation Summary**:
- ✅ Added `Model string` field to `RerankRequest` in `internal/embeddings/reranker.go`
- ✅ Updated client to send `c.model` in rerank requests
- ✅ Added `Model string` field to `RerankRequest` in `cmd/tei-wrapper/main.go`
- ✅ Added unit tests: `TestRerankModelField` and `TestRerankRequestMarshaling`
- ✅ All existing tests continue to pass

**Success Criteria**:
- ✅ RerankRequest includes model field in both client and server
- ✅ Client sends model name with every rerank request
- ✅ No breaking changes to existing code (model field is added, not replacing anything)

---

### Phase 3: Implement Dynamic Model Loading in tei-wrapper

**Goal**: Add model switching logic to tei-wrapper similar to how embedding models work.

**Changes Required**:

1. **Add fields to Server struct** ([cmd/tei-wrapper/main.go:54-81](../../cmd/tei-wrapper/main.go#L54-L81))
   ```go
   type Server struct {
       // ... existing fields ...

       // Reranker TEI management
       rerankPort           int
       currentRerankModel   string        // ADD: Currently loaded reranker model
       rerankCmd            *exec.Cmd
       rerankBaseURL        string
       rerankHealthy        bool
       rerankSwitching      bool          // ADD: True during reranker model switch
   }
   ```

2. **Implement switchRerankModel()** (new function, add after line 278)
   ```go
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
   ```

3. **Update handleRerank() for dynamic loading** ([cmd/tei-wrapper/main.go:594-639](../../cmd/tei-wrapper/main.go#L594-L639))
   ```go
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
   ```

4. **Update startup logic** ([cmd/tei-wrapper/main.go:132-145](../../cmd/tei-wrapper/main.go#L132-L145))
   - Keep CLI flag support for backward compatibility
   - If `--rerank-model` is set, start reranker at startup AND set `currentRerankModel`
   - If not set, leave reranker uninitialized (will start on first request)

5. **Update health endpoint** ([cmd/tei-wrapper/main.go:438-447](../../cmd/tei-wrapper/main.go#L438-L447))
   ```go
   // Include reranker status
   if s.currentRerankModel != "" || s.rerankModel != "" {
       rerankHealthy := s.checkRerankHealth()
       response["reranker"] = map[string]interface{}{
           "enabled":        true,
           "healthy":        rerankHealthy,
           "model":          s.currentRerankModel,
           "port":           s.rerankPort,
           "switching":      s.rerankSwitching,
           "startup_model":  s.rerankModel,  // Model from CLI flag (if any)
       }
   }
   ```

**Testing**:
- ✅ Unit test: switchRerankModel() changes current model
- ✅ Unit test: handleRerank() parses model from request
- ✅ Unit test: Backward compatibility with CLI flag
- ✅ Integration test: Start with no reranker, first request loads model
- ✅ Integration test: Switch between two different models mid-session
- ✅ Integration test: Concurrent requests during model switch return 503
- ✅ Integration test: CLI flag still works (backward compat)

**Success Criteria**:
- tei-wrapper can start without `--rerank-model` flag
- First rerank request loads the model specified in request
- Subsequent requests reuse loaded model
- Model switching works correctly with proper locking
- Backward compatibility maintained (CLI flag still works)
- Health endpoint shows current reranker model

---

### Phase 4: Testing & Validation

**Goal**: Verify the implementation works correctly across all scenarios.

**Test Plan**:

1. **Unit Tests**:
   - `internal/embeddings/reranker_test.go`: Test RerankRequest serialization with model field
   - `cmd/tei-wrapper/server_test.go`: Test model switching logic

2. **Integration Tests** (requires TEI installed):
   - Test 1: Dynamic loading from scratch
     ```bash
     # Start tei-wrapper without reranker
     ./tei-wrapper --model nomic-ai/nomic-embed-text-v1.5

     # Configure reranker in .code-scout.json
     {
      "endpoint": "http://localhost:11435",
       "rerank_model": "BAAI/bge-reranker-large"
     }

     # Run search - should load reranker on first use
     code-scout search "test query"
     ```

   - Test 2: Model switching
     ```bash
     # First search with model A
     # Edit .code-scout.json to use model B
     # Second search should switch to model B
     ```

   - Test 3: Backward compatibility
     ```bash
     # Start with CLI flag
     ./tei-wrapper --rerank-model BAAI/bge-reranker-large

     # Search should work without model in request
     ```

3. **End-to-End Tests**:
   - Test with at least 2 different reranker models:
     - `BAAI/bge-reranker-large` (primary)
     - `Alibaba-NLP/gte-multilingual-reranker-base` (alternative)
   - Verify reranking improves search results
   - Verify scores are properly returned

4. **Error Handling Tests**:
   - Invalid model name in request
   - TEI download fails for model
   - Model switch fails mid-request
   - Concurrent requests during switch

**Success Criteria**:
- All unit tests pass
- All integration tests pass
- At least 2 different reranker models verified working
- Error handling gracefully handles all failure modes
- No memory leaks during model switching

---

### Phase 5: Documentation Updates

**Goal**: Update all documentation to reflect new dynamic loading approach.

**Files to Update**:

1. **[README.md](../../README.md)**:
   - Remove `--rerank-model` from quick start example (line 319)
   - Update configuration example to show config-only approach (lines 303-312)
   - Update "With Cross-Encoder Reranking" example (lines 303-334)
   - Add note: "The `--rerank-model` CLI flag is deprecated. Use `.code-scout.json` instead."

2. **[docs/guides/RERANKER_SETUP.md](../../docs/guides/RERANKER_SETUP.md)**:
   - Rewrite "Starting tei-wrapper with reranking" section
   - Show config-first workflow (edit `.code-scout.json`, then just start tei-wrapper)
   - Add "Dynamic Model Loading" section explaining the new behavior
   - Add "Model Switching" section with example
   - Move CLI flag approach to "Legacy Setup (Deprecated)" section

3. **[docs/guides/TEI_WRAPPER.md](../../docs/guides/TEI_WRAPPER.md)**:
   - Mark `--rerank-model` as deprecated in CLI flags table
   - Add "Reranker Model Loading" section explaining dynamic loading
   - Update examples to use config-only approach

4. **Create [docs/guides/TESTED_MODELS.md](../../docs/guides/TESTED_MODELS.md)**:
   - Table of verified-working reranker models
   - Include TEI version, model name, download size, memory usage
   - Note any known issues or performance characteristics
   - Example:
     ```markdown
     | Model | TEI Version | Download Size | Memory | Notes |
     |-------|-------------|---------------|--------|-------|
     | BAAI/bge-reranker-large | 1.5.0 | 1.1GB | 2GB | Recommended |
     | Alibaba-NLP/gte-multilingual-reranker-base | 1.5.0 | 278MB | 1GB | Lighter alternative |
     ```

**Success Criteria**:
- All documentation updated and accurate
- No references to `--rerank-model` CLI flag in main examples
- CLI flag clearly marked as deprecated
- New workflow clearly documented with examples
- Tested models table created and populated

---

## Implementation Order

Follow phases in order:
0. **Phase 0 (PREREQUISITE)** → Troubleshoot and fix TEI reranker model loading
1. **Phase 1** → Add config file support to tei-wrapper (read `.code-scout.json`, set defaults)
2. **Phase 2** → Update request/response formats (add model field)
3. **Phase 3** → Implement dynamic model loading in tei-wrapper
4. **Phase 4** → Testing & validation
5. **Phase 5** → Update documentation

**CRITICAL**: Phase 0 is a blocker for all subsequent phases. We cannot implement dynamic loading without understanding how to correctly invoke TEI for reranker models.

Each phase should be completed and tested before moving to the next.

## Rollback Plan

If issues arise:
1. **Phase 0**: Continue using workarounds, defer feature until TEI issue resolved
2. **Phase 1**: Revert config loading, fall back to CLI flags only
3. **Phase 2-3**: Revert code changes, no user impact (backward compatible with CLI flags)
4. **Phase 4**: Fix tests, don't merge until all pass
5. **Phase 5**: Documentation can be updated iteratively

## Risk Mitigation

**Risk**: Model download failures during dynamic loading
**Mitigation**: Clear error messages, health endpoint shows status, retry logic with exponential backoff

**Risk**: Memory spikes during model switching
**Mitigation**: Stop old model before starting new one, mutex prevents concurrent switches

**Risk**: Breaking changes for existing users
**Mitigation**: Maintain CLI flag support, only deprecate (don't remove)

## Out of Scope

- Performance optimization of model switching (acceptable to have startup latency)
- Caching multiple reranker models simultaneously
- Auto-detection of "best" reranker model
- Supporting models incompatible with TEI 1.8.3 (e.g., fixing `BAAI/bge-reranker-base` if it's a TEI bug)

## Success Metrics

Implementation is complete when:
- ✅ Config file discovery searches up directory tree (like git) - **DONE (Phase 1)**
- ✅ Both code-scout and tei-wrapper find `.code-scout.json` from any subdirectory - **DONE (Phase 1)**
- ✅ Warning logged when no config found and defaults used - **DONE (Phase 1)**
- ✅ Log shows which config file was loaded (with full path) - **DONE (Phase 1)**
- ✅ tei-wrapper reads `.code-scout.json` for model configuration - **DONE (Phase 1)**
- ✅ tei-wrapper works without any CLI flags (`./tei-wrapper` just works) - **DONE (Phase 1)**
- ✅ RerankRequest includes Model field for dynamic loading - **DONE (Phase 2)**
- ✅ Client sends model in every rerank request - **DONE (Phase 2)**
- ✅ Unit tests for Model field marshaling and transmission - **DONE (Phase 2)**
- ⏳ User can configure all models (embedding + reranker) in `.code-scout.json` only - **Phase 3**
- ⏳ First rerank request loads the configured reranker model automatically - **Phase 3**
- ⏳ Model switching works mid-session for both embedding and reranker models - **Phase 3**
- ⏳ All tests pass (unit, integration, end-to-end) - **Phase 4**
- ✅ Documentation updated and accurate - **Phase 1-2 docs done, Phase 5 remaining**
- ⏳ At least 2 different reranker models verified working - **Phase 0/4**
- ⏳ Config priority order respected: Request > Config > CLI > Default - **Phase 3**

## Notes

- **Design Principle**: `.code-scout.json` is the single source of truth for configuration
- **Config Discovery**: Both tools search up directory tree (git-like behavior) - modify `internal/config/config.go` once, both tools benefit
- tei-wrapper and code-scout read the same config file for consistency
- Config loading always logs which file was loaded or warns if using defaults
- Follow the same pattern used for embedding model switching (proven to work)
- Maintain backward compatibility during transition period (CLI flags still work)
- CLI flags can be removed in a future major version
- Default models are sensible (nomic-ai/nomic-embed-text-v1.5 for embeddings)
- Reranker has no default (loads on-demand when first requested)
