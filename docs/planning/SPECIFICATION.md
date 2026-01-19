# Reranker Dynamic Model Loading - Specification

**Status**: Draft
**Created**: 2026-01-18
**Issue**: Reranker models cannot be configured dynamically via `.code-scout.json`

## Problem Statement

The reranker feature is inconsistent with the embedding model architecture. Embedding models support dynamic loading (client specifies model in config, tei-wrapper switches models on-demand), but reranker models are hardcoded at tei-wrapper startup via CLI flags. This violates the design principle of configuration-driven model selection.

## Current Behavior (Incorrect)

### Configuration Flow

1. **User configures reranker** in `.code-scout.json`:
   ```json
   {
     "endpoint": "http://localhost:11435",
     "rerank_model": "BAAI/bge-reranker-base",
     "rerank_top_k": 25
   }
   ```

2. **User starts tei-wrapper** WITHOUT the reranker model:
   ```bash
   ./tei-wrapper
   ```

3. **User runs search**:
   ```bash
   code-scout search "authentication logic"
   ```

4. **Search fails** with error:
   ```
   Error: failed to rerank results: failed after 3 attempts:
   rerank API returned status 404: Reranker not configured.
   Start tei-wrapper with --rerank-model flag
   ```

### Why It Fails

**Client Side** ([cmd/code-scout/embeddings_factory.go:24-30](../../cmd/code-scout/embeddings_factory.go#L24-L30)):
- Creates `RerankClient` with `globalConfig.RerankModel`
- Stores model name in client struct
- **BUT**: Never sends model name to server

**Request Format** ([internal/embeddings/reranker.go:20-26](../../internal/embeddings/reranker.go#L20-L26)):
```go
type RerankRequest struct {
    Query      string   `json:"query"`
    Texts      []string `json:"texts"`
    RawScores  bool     `json:"raw_scores,omitempty"`
    ReturnText bool     `json:"return_text,omitempty"`
    // NO MODEL FIELD!
}
```

**Server Side** ([cmd/tei-wrapper/main.go:594-639](../../cmd/tei-wrapper/main.go#L594-L639)):
- Checks if `--rerank-model` CLI flag was set at startup (line 602)
- If not set, returns HTTP 404
- If set, proxies to pre-configured TEI reranker instance
- **NO dynamic model loading capability**

### Comparison: Embedding Models (Correct Behavior)

**Embeddings work correctly** because they follow the dynamic model loading pattern:

**Client Side**:
```go
// Sends model name with EVERY request
reqBody := openAIEmbedRequest{
    Model: c.model,  // ✅ Model included in request
    Input: truncatedTexts,
}
```

**Server Side** ([cmd/tei-wrapper/main.go:280-355](../../cmd/tei-wrapper/main.go#L280-L355)):
```go
// Receives model from request
var req EmbeddingRequest
json.NewDecoder(r.Body).Decode(&req)

// Dynamically switches if needed (line 313-319)
if req.Model != s.currentModel {
    s.switchModel(req.Model)  // ✅ Dynamic switching
}
```

## Expected Behavior (Correct)

### Design Principle

**Single Source of Truth**: Model configuration should live ONLY in `.code-scout.json`, not in tei-wrapper CLI flags.

**Dynamic Loading**: tei-wrapper should load/switch models on-demand based on client requests, just like it does for embeddings.

### Configuration Flow (Fixed)

1. **User configures models** in `.code-scout.json`:
   ```json
   {
     "endpoint": "http://localhost:11435",
     "code_model": "nomic-ai/CodeRankEmbed",
     "text_model": "nomic-ai/nomic-embed-text-v1.5",
     "rerank_model": "BAAI/bge-reranker-large",
     "rerank_top_k": 25
   }
   ```

2. **User starts tei-wrapper** without any flags:
   ```bash
   ./tei-wrapper
   ```
   - tei-wrapper searches for `.code-scout.json` (walks up directory tree from PWD, like git)
   - Falls back to `~/.code-scout/config.json` if project config not found
   - Logs which config file was loaded (or warns if using defaults)
   - Starts embedding TEI with `text_model` (default: `nomic-ai/nomic-embed-text-v1.5`)
   - Does NOT start reranker TEI yet (waits for first rerank request)
   - Ready to switch embedding models and load reranker on-demand

3. **User runs search**:
   ```bash
   code-scout search "authentication logic"
   ```

4. **Search succeeds**:
   - code-scout sends embedding request with `text_model` → tei-wrapper uses already-loaded model
   - code-scout sends rerank request with `rerank_model` → tei-wrapper loads reranker TEI with configured model
   - Subsequent requests reuse loaded models (or switch if different model requested)

**Key Design Points**:
- ✅ `.code-scout.json` is the single source of truth for ALL model configuration
- ✅ tei-wrapper reads the same config file as code-scout for consistency
- ✅ Config file discovery searches up directory tree from PWD (git-like behavior)
- ✅ Works from any subdirectory - finds project root's `.code-scout.json`
- ✅ Logs which config was loaded (or warns when using defaults)
- ✅ tei-wrapper has sensible defaults and works without any CLI flags
- ✅ Embedding models preload at startup (based on config)
- ✅ Reranker models load on-demand (first rerank request)
- ✅ CLI flags remain available for backward compatibility but are deprecated

**Config File Discovery** (applies to both code-scout and tei-wrapper):
1. Search up from current working directory (PWD) for `.code-scout.json`
   - Checks: `./`, `../`, `../../`, etc. until filesystem root
   - Works regardless of where binary is installed
2. Fall back to user-level: `~/.code-scout/config.json`
3. Fall back to built-in defaults (with warning)
4. Log output shows which config was loaded:
   ```
   Loaded config from: /Users/alice/projects/my-app/.code-scout.json
   ```
   Or if no config found:
   ```
   WARNING: No .code-scout.json found (searched up directory tree) and no ~/.code-scout/config.json found. Using default configuration.
   ```

### Required Request Format

**Rerank request must include model** ([internal/embeddings/reranker.go:20-26](../../internal/embeddings/reranker.go#L20-L26)):
```go
type RerankRequest struct {
    Query      string   `json:"query"`
    Texts      []string `json:"texts"`
    RawScores  bool     `json:"raw_scores,omitempty"`
    ReturnText bool     `json:"return_text,omitempty"`
    Model      string   `json:"model"`  // ✅ ADD THIS
}
```

**Server must handle model field** ([cmd/tei-wrapper/main.go:594-639](../../cmd/tei-wrapper/main.go#L594-L639)):
```go
func (s *Server) handleRerank(w http.ResponseWriter, r *http.Request) {
    var req RerankRequest
    json.NewDecoder(r.Body).Decode(&req)

    // ✅ Check if we need to start/switch reranker
    if s.rerankModel != req.Model {
        s.switchRerankModel(req.Model)  // Start or switch reranker TEI
    }

    // Proxy to reranker TEI
    // ...
}
```

### Backward Compatibility

**Config Priority Order** (for smooth migration):
1. **Request model field** (highest priority) - If client sends model in request, use that
2. **`.code-scout.json`** (second priority) - If no model in request, use config file value
3. **CLI flag** (backward compat) - If neither above, fall back to `--rerank-model` CLI flag (deprecated)
4. **Built-in default** (lowest priority) - If nothing else specified, use hardcoded default

**Examples**:
```bash
# Modern approach (preferred): Just run tei-wrapper, it reads .code-scout.json
./tei-wrapper

# Legacy approach (deprecated but still works): CLI flags override config
./tei-wrapper --model nomic-ai/nomic-embed-text-v1.5 --rerank-model BAAI/bge-reranker-large

# Hybrid: CLI for embedding, config for reranker
./tei-wrapper --model nomic-ai/CodeRankEmbed  # reranker from .code-scout.json
```

**Recommendation**: Document config-based approach as primary, mark CLI flags as deprecated but maintain support for one major version.

## Secondary Issue: Model Compatibility

**Problem (sandboxed)**: In the sandboxed environment, TEI logged 404 errors while trying to resolve `config_sentence_transformers.json` for `BAAI/bge-reranker-base`.

**Update (unsandboxed)**: `text-embeddings-router` runs without panic for `BAAI/bge-reranker-large` and `BAAI/bge-reranker-v2-m3` and serves `/rerank` successfully. User report: `BAAI/bge-reranker-base` also starts without panic. This suggests the earlier 404/panic behavior was a sandbox artifact rather than a TEI/model incompatibility.

**Root Cause (likely)**: Sandbox restrictions (network/filesystem) prevented TEI from downloading or initializing models reliably.

**Tested Alternatives**:
- ✅ `BAAI/bge-reranker-large` - Verified working in unsandboxed environment
- ✅ `BAAI/bge-reranker-v2-m3` - Verified working in unsandboxed environment
- ✅ `BAAI/bge-reranker-base` - User-reported working in unsandboxed environment
- ❓ `Alibaba-NLP/gte-multilingual-reranker-base` - Unverified (retest unsandboxed)
- ❓ `Alibaba-NLP/gte-reranker-modernbert-base` - Unverified (retest unsandboxed)

**Action Required**: Re-run remaining model compatibility tests in the unsandboxed environment and record TEI version plus exact commands used.

## Success Criteria

### Functional Requirements

1. ✅ **User can configure reranker model** in `.code-scout.json` without touching tei-wrapper CLI
2. ✅ **First rerank request loads the model** specified in config
3. ✅ **Subsequent requests reuse loaded model** (no restart needed)
4. ✅ **Model switching works** if user changes config and runs new search
5. ✅ **Graceful fallback** if model fails to load (error message, not crash)

### Testing Requirements

1. ✅ **Unit tests** for updated `RerankRequest` struct
2. ✅ **Integration tests** for dynamic model loading in tei-wrapper
3. ✅ **Local wrapper test setup**: start tei-wrapper locally and set `.code-scout.json` `endpoint` to the local wrapper (default `http://localhost:11435`) before running code-scout searches
4. ✅ **End-to-end tests** with at least 2 different reranker models:
   - `BAAI/bge-reranker-large` (primary)
   - `Alibaba-NLP/gte-multilingual-reranker-base` (alternative)
5. ✅ **Model switching test**: Start with model A, switch to model B mid-session
6. ✅ **Error handling test**: Request with invalid/unavailable model

### Documentation Requirements

1. ✅ **Update README.md** - Remove references to `--rerank-model` CLI flag
2. ✅ **Update RERANKER_SETUP.md** - New dynamic loading workflow
3. ✅ **Update TEI_WRAPPER.md** - Mark `--rerank-model` as deprecated
4. ✅ **Add tested models table** - List verified-working reranker models with versions

## Out of Scope

- Performance optimization of model switching (acceptable to have startup latency on first request)
- Caching multiple reranker models simultaneously (only one reranker loaded at a time)
- Auto-detection of "best" reranker model (user must specify in config)

## References

**Code Locations**:
- [cmd/code-scout/embeddings_factory.go](../../cmd/code-scout/embeddings_factory.go) - Client factory
- [internal/embeddings/reranker.go](../../internal/embeddings/reranker.go) - Rerank client
- [cmd/tei-wrapper/main.go](../../cmd/tei-wrapper/main.go) - Server implementation
- [internal/config/config.go](../../internal/config/config.go) - Config struct

**External Documentation**:
- [TEI Supported Models](https://huggingface.co/docs/text-embeddings-inference/supported_models)
- [TEI Quick Tour - Reranking](https://huggingface.co/docs/text-embeddings-inference/en/quick_tour)
- [BAAI/bge-reranker-large](https://huggingface.co/BAAI/bge-reranker-large) - Working model
- [Alibaba-NLP/gte-multilingual-reranker-base](https://huggingface.co/Alibaba-NLP/gte-multilingual-reranker-base) - Alternative
