# Cross-Encoder Reranking Setup Guide

This guide covers deploying and using cross-encoder reranking to improve Code Scout's search relevance. Reranking is an optional feature that provides significantly better search results at the cost of slightly increased latency and memory usage.

## Overview

### What is Cross-Encoder Reranking?

Code Scout performs semantic search in two stages:

**Stage 1: Bi-Encoder Vector Search (Fast)**
- User query → embedding model → query vector
- Vector similarity search in LanceDB → top-K results
- Simple cosine distance scoring
- **Limitation:** Encodes query and documents independently, missing complex relevance patterns

**Stage 2: Cross-Encoder Reranking (Accurate)**
- Take top-K results from Stage 1
- Cross-encoder model scores each query-document pair together
- Re-sort results by cross-encoder scores
- **Advantage:** Learns complex query-document interactions for better relevance

### When to Use Reranking

**Use reranking when:**
- ✅ Search result relevance is critical
- ✅ You have GPU acceleration (Metal, CUDA)
- ✅ You can spare 500MB-2GB additional memory
- ✅ Acceptable latency increase (~100-500ms)

**Skip reranking when:**
- ❌ Memory is very constrained (<4GB available)
- ❌ You need ultra-low latency (<50ms)
- ❌ Simple vector search is "good enough"

### Performance Trade-offs

| Configuration | Memory | Latency | Relevance Quality |
|---------------|--------|---------|-------------------|
| **Vector-only** | 4-8GB | ~50ms | Good |
| **+ Reranking (base)** | +500MB | +100ms | Excellent |
| **+ Reranking (large)** | +2GB | +300ms | Best |

## Quick Start

### 1. Build and Start TEI Wrapper with Reranker

The tei-wrapper automatically manages both embedding and reranker TEI instances:

```bash
# Build tei-wrapper (if not already built)
cd cmd/tei-wrapper
go build -o tei-wrapper .

# Start with reranker enabled (spawns two TEI instances)
./tei-wrapper \
  --model nomic-ai/nomic-embed-text-v1.5 \
  --rerank-model BAAI/bge-reranker-base
```

**What happens:**
1. Wrapper starts **embedding TEI** on port 8080 (hot-swappable models)
2. Wrapper starts **reranker TEI** on port 8081 (dedicated instance)
3. Wrapper exposes unified API on port 11434:
   - `/v1/embeddings` → routes to embedding TEI
   - `/rerank` → routes to reranker TEI
   - `/health` → shows status of both instances

### 2. Configure Code Scout

Create or update `.code-scout.json`:

```json
{
  "endpoint": "http://localhost:11434",
  "code_model": "nomic-ai/CodeRankEmbed",
  "text_model": "nomic-ai/nomic-embed-text-v1.5",
  "rerank_model": "BAAI/bge-reranker-base",
  "rerank_top_k": 25
}
```

**Configuration fields:**
- `endpoint`: tei-wrapper endpoint (single entry point for all services)
- `rerank_model`: Cross-encoder model name (empty = reranking disabled)
- `rerank_top_k`: Number of top results to rerank (higher = better but slower)

### 3. Search with Reranking

```bash
# Index your codebase (if not already indexed)
code-scout index

# Search - reranking happens automatically
code-scout search "authentication logic" --limit 10
```

**Expected output:**
```
Found 10 unique hybrid results (reranked by BAAI/bge-reranker-base, from 47 total) for: authentication logic

1. internal/auth/handler.go:23-45 (vector: 0.234, rerank: 0.95)
   Language: go | Source: code | Chunk: function
   func AuthHandler(w http.ResponseWriter, r *http.Request) {
   ...
```

Notice:
- ✅ Both `vector` and `rerank` scores shown
- ✅ Results sorted by rerank score (highest first)
- ✅ Output indicates reranking model used

## Model Selection Guide

### Recommended Models

All models listed below are tested and supported by Code Scout:

| Model | Size | Speed | Accuracy | Use Case |
|-------|------|-------|----------|----------|
| **BAAI/bge-reranker-base** | 278M | Fast | Good | **Default** - Balanced speed/accuracy |
| **BAAI/bge-reranker-large** | 560M | Moderate | Excellent | Higher accuracy needed |
| **BAAI/bge-reranker-v2-m3** | 568M | Moderate | Excellent | Multilingual support, 8K context |
| **cross-encoder/ms-marco-MiniLM-L-6-v2** | 22M | Very Fast | Good | Low memory, CPU-only systems |
| **jinaai/jina-reranker-v1-turbo-en** | 137M | Fast | Very Good | Speed-optimized, 8K context |

### Token Limits and Context Windows

**Important:** Reranker models have hard token limits that affect how much code can be scored:

| Model | Token Limit | Approximate Chars | Best For |
|-------|-------------|-------------------|----------|
| **BAAI/bge-reranker-base** | 512 | ~1200 chars | Short-medium chunks |
| **BAAI/bge-reranker-v2-m3** | 8192 | ~20K chars | Large chunks, whole files |
| **jinaai/jina-reranker-v1-turbo-en** | 8192 | ~20K chars | Large chunks, whole files |

**Code Scout behavior:** When chunks exceed the token limit, they are automatically truncated to fit. This preserves functionality but may impact semantic quality for very large chunks.

**Recommendations:**
1. **For production use:** Prefer models with 8K+ context (bge-reranker-v2-m3, jina-reranker-v1-turbo-en)
2. **For testing/development:** bge-reranker-base is fine for most code chunks
3. **For large documentation:** Use 8K context models to avoid truncation
4. **For `--code` mode:** Any model works well (code chunks are naturally small)

### Memory Requirements

Memory usage depends on model size and GPU acceleration:

| Configuration | CPU | GPU (Metal/CUDA) |
|---------------|-----|------------------|
| **base (278M)** | 1GB | 500MB |
| **large (560M)** | 2GB | 1GB |
| **v2-m3 (568M)** | 2GB | 1GB |
| **MiniLM (22M)** | 100MB | 50MB |
| **jina-turbo (137M)** | 500MB | 300MB |

**Total system memory:** Add embedding models (4-8GB) + reranker = 4.5-10GB total recommended.

## Platform-Specific Setup

### macOS (Homebrew + Metal GPU)

**Prerequisites:**
- Apple Silicon (M1/M2/M3) or Intel Mac with AMD GPU
- Homebrew installed
- Metal GPU acceleration available

**Install TEI:**
```bash
# Install TEI (if not already installed)
brew install text-embeddings-inference
```

**Start tei-wrapper:**
```bash
# Navigate to tei-wrapper directory
cd cmd/tei-wrapper

# Build (if not already built)
go build -o tei-wrapper .

# Start with reranker (Metal GPU acceleration automatic)
./tei-wrapper \
  --model nomic-ai/nomic-embed-text-v1.5 \
  --rerank-model BAAI/bge-reranker-base
```

**Verify:**
```bash
# Check health endpoint
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

### Linux (Native Binary + CUDA)

**Prerequisites:**
- NVIDIA GPU with CUDA support
- CUDA toolkit installed

**Install TEI:**
```bash
# Download TEI binary
curl -LO https://github.com/huggingface/text-embeddings-inference/releases/latest/download/text-embeddings-inference-linux-x86_64

# Make executable
chmod +x text-embeddings-inference-linux-x86_64

# Move to PATH
sudo mv text-embeddings-inference-linux-x86_64 /usr/local/bin/text-embeddings-router
```

**Start tei-wrapper:**
```bash
cd cmd/tei-wrapper
go build -o tei-wrapper .

# Start with reranker (CUDA acceleration automatic)
./tei-wrapper \
  --tei-binary /usr/local/bin/text-embeddings-router \
  --model nomic-ai/nomic-embed-text-v1.5 \
  --rerank-model BAAI/bge-reranker-base
```

### Linux/Windows (Docker + CUDA)

**Prerequisites:**
- Docker installed
- NVIDIA GPU with nvidia-docker2

**Option 1: Manual TEI Instances (Advanced)**

If you want to manage TEI instances yourself instead of using tei-wrapper:

```bash
# Start embedding TEI
docker run --gpus all -d -p 8080:80 \
  --name tei-embeddings \
  ghcr.io/huggingface/text-embeddings-inference:latest \
  --model-id nomic-ai/CodeRankEmbed

# Start reranker TEI
docker run --gpus all -d -p 8081:80 \
  --name tei-reranker \
  ghcr.io/huggingface/text-embeddings-inference:latest \
  --model-id BAAI/bge-reranker-base

# Configure code-scout to talk directly to TEI endpoints
# (Not recommended - no model switching, manual management)
```

**Option 2: TEI Wrapper in Container (Recommended)**

Use tei-wrapper with native TEI binaries for best performance.

### CPU-Only Deployment

For systems without GPU:

```bash
# Use lightweight model for better CPU performance
./tei-wrapper \
  --model nomic-ai/nomic-embed-text-v1.5 \
  --rerank-model cross-encoder/ms-marco-MiniLM-L-6-v2
```

**Performance expectations:**
- Embedding: ~200-500ms per batch
- Reranking: ~100-300ms for 25 results
- Total: ~300-800ms per search

## Configuration Reference

### tei-wrapper Command-Line Flags

Reranker-specific flags:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--rerank-port` | int | `8081` | Port for dedicated reranker TEI instance |
| `--rerank-model` | string | `""` | Reranker model ID (empty = reranker disabled) |

**Example:**
```bash
./tei-wrapper \
  --port 11434 \
  --tei-port 8080 \
  --model nomic-ai/nomic-embed-text-v1.5 \
  --rerank-port 8081 \
  --rerank-model BAAI/bge-reranker-base
```

See [TEI_WRAPPER.md](TEI_WRAPPER.md) for complete flag reference.

### code-scout Configuration

Reranking is configured in `.code-scout.json` or `~/.code-scout/config.json`:

```json
{
  "endpoint": "http://localhost:11434",
  "code_model": "nomic-ai/CodeRankEmbed",
  "text_model": "nomic-ai/nomic-embed-text-v1.5",
  "rerank_model": "BAAI/bge-reranker-base",
  "rerank_top_k": 25
}
```

**Configuration fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `endpoint` | string | Yes | tei-wrapper endpoint (single unified endpoint) |
| `rerank_model` | string | No | Cross-encoder model name (empty = disabled) |
| `rerank_top_k` | int | No | Number of results to rerank (default: search `--limit`) |

**Important:** There is **no** `rerank_endpoint` field. The wrapper handles routing to the reranker TEI instance internally.

## Performance Tuning

### Optimizing `rerank_top_k`

The `rerank_top_k` parameter controls how many results get reranked:

| rerank_top_k | Latency | Accuracy | Use Case |
|--------------|---------|----------|----------|
| **10** | ~50ms | Good | Fast searches, low `--limit` |
| **25** | ~100ms | Better | **Default** - balanced |
| **50** | ~200ms | Best | High accuracy needs |
| **100** | ~400ms | Excellent | Exhaustive reranking |

**Rule of thumb:** Set `rerank_top_k` to 2-3x your typical search `--limit`.

**Example configurations:**

```json
// Fast searches (limit: 10)
{
  "rerank_model": "BAAI/bge-reranker-base",
  "rerank_top_k": 10
}

// Balanced (limit: 10-20)
{
  "rerank_model": "BAAI/bge-reranker-base",
  "rerank_top_k": 25
}

// High accuracy (limit: 20-50)
{
  "rerank_model": "BAAI/bge-reranker-v2-m3",
  "rerank_top_k": 100
}
```

### When to Disable Reranking

Temporarily disable reranking for specific searches:

```bash
# Quick search without reranking - just remove rerank_model from config
# Or use a separate config file
code-scout search "quick lookup" --limit 5
```

To disable globally:
```json
{
  "endpoint": "http://localhost:11434",
  "code_model": "nomic-ai/CodeRankEmbed",
  "text_model": "nomic-ai/nomic-embed-text-v1.5"
  // rerank_model removed or empty = reranking disabled
}
```

### Memory Optimization

If memory is constrained:

1. **Use smaller reranker model:**
   ```bash
   ./tei-wrapper --rerank-model cross-encoder/ms-marco-MiniLM-L-6-v2
   ```

2. **Reduce max batch tokens** (embedding TEI):
   ```bash
   ./tei-wrapper --max-batch-tokens 4096
   ```

3. **Lower rerank_top_k:**
   ```json
   {
     "rerank_top_k": 10
   }
   ```

## Troubleshooting

### Reranker Not Working

**Symptom:** Search results don't show rerank scores, or only show vector scores.

**Diagnosis:**
```bash
# Check health endpoint
curl http://localhost:11434/health | jq

# Look for reranker section
{
  "reranker": {
    "enabled": true,    # Should be true
    "healthy": true,    # Should be true
    ...
  }
}
```

**Solutions:**

1. **reranker.enabled = false:**
   - Check `--rerank-model` flag is set when starting tei-wrapper
   - Verify model name is not empty string

2. **reranker.healthy = false:**
   - Check tei-wrapper logs for reranker TEI startup errors
   - Verify port 8081 is not already in use: `lsof -i :8081`
   - Try different model: `--rerank-model BAAI/bge-reranker-base`

3. **rerank_model not in code-scout config:**
   - Add `"rerank_model": "BAAI/bge-reranker-base"` to `.code-scout.json`

### Token Limit Errors

**Symptom:** Errors like "Input validation error: inputs must have less than 512 tokens"

**Cause:** Large code chunks exceed reranker's token limit (512 for bge-reranker-base).

**Solutions:**

1. **Use higher-limit model (recommended):**
   ```bash
   ./tei-wrapper --rerank-model BAAI/bge-reranker-v2-m3  # 8192 tokens
   ```

2. **Reduce chunk sizes** (advanced - requires code changes):
   - Chunks are automatically truncated to fit token limits
   - This is a fallback - prefer using higher-limit models

3. **Lower `rerank_top_k`** to avoid reranking very large chunks:
   ```json
   {
     "rerank_top_k": 10
   }
   ```

### Port Conflicts

**Symptom:** "address already in use" error when starting tei-wrapper.

**Solutions:**

1. **Find process using port:**
   ```bash
   # macOS/Linux
   lsof -i :8081

   # Kill process
   kill -9 <PID>
   ```

2. **Use different port:**
   ```bash
   ./tei-wrapper --rerank-port 8082
   ```

### High Memory Usage

**Symptom:** System runs out of memory or becomes slow.

**Diagnosis:**
```bash
# Check memory usage (macOS)
top -l 1 | grep -E "PhysMem|tei|wrapper"

# Linux
free -h
ps aux | grep -E "tei|wrapper"
```

**Solutions:**

1. **Use smaller reranker model:**
   ```bash
   ./tei-wrapper --rerank-model cross-encoder/ms-marco-MiniLM-L-6-v2  # 22M params
   ```

2. **Reduce embedding model batch tokens:**
   ```bash
   ./tei-wrapper --max-batch-tokens 4096  # Default: 8192
   ```

3. **Disable idle preload** (if enabled):
   ```bash
   ./tei-wrapper  # Don't use --idle-preload
   ```

### Slow Search Performance

**Symptom:** Searches take >1 second.

**Diagnosis:** Check if you have GPU acceleration:

```bash
# macOS - Check Metal GPU
system_profiler SPDisplaysDataType | grep "Metal"

# Linux - Check CUDA
nvidia-smi
```

**Solutions:**

1. **Ensure GPU acceleration:**
   - macOS: Metal should be automatic with Homebrew TEI
   - Linux: Use `--gpus all` with Docker, or CUDA-enabled binary

2. **Use faster reranker model:**
   ```bash
   ./tei-wrapper --rerank-model jinaai/jina-reranker-v1-turbo-en
   ```

3. **Lower `rerank_top_k`:**
   ```json
   {
     "rerank_top_k": 10
   }
   ```

4. **CPU-only systems:** Use lightweight model:
   ```bash
   ./tei-wrapper --rerank-model cross-encoder/ms-marco-MiniLM-L-6-v2
   ```

## Health Check Interpretation

The `/health` endpoint provides reranker status:

```bash
curl http://localhost:11434/health | jq
```

**Example response:**
```json
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

**Status meanings:**

| Field | Value | Meaning |
|-------|-------|---------|
| `status` | `"ok"` | Wrapper and embedding TEI healthy |
| `status` | `"switching"` | Embedding model switch in progress (2-3s) |
| `status` | `"unhealthy"` | Embedding TEI is down |
| `reranker.enabled` | `true` | Reranker configured (--rerank-model set) |
| `reranker.enabled` | `false` | Reranker not configured |
| `reranker.healthy` | `true` | Reranker TEI is responding |
| `reranker.healthy` | `false` | Reranker TEI is down |

## Log Analysis

tei-wrapper logs to stdout/stderr. Watch for these messages:

**Successful startup:**
```
Starting embeddings TEI with model: nomic-ai/nomic-embed-text-v1.5
Embeddings TEI started (PID: 12345, port: 8080)
Starting reranker TEI with model: BAAI/bge-reranker-base
Reranker TEI started (PID: 12346, port: 8081)
Both TEI instances ready!
Wrapper listening on :11434
```

**Common errors:**

| Log Message | Cause | Solution |
|-------------|-------|----------|
| `Failed to start reranker TEI: exec: "text-embeddings-router": executable file not found` | TEI not in PATH | Install TEI or use `--tei-binary` flag |
| `address already in use` | Port conflict | Change `--rerank-port` or kill process using port |
| `Reranker TEI did not become ready within 90s` | Model download timeout or memory issue | Check network, increase memory, or use smaller model |
| `Reranker request failed: connection refused` | Reranker TEI crashed | Check logs, try different model, verify memory available |

## Advanced Configuration

### Using Separate TEI Instances (Not Recommended)

If you want to manage TEI instances manually instead of using tei-wrapper:

```bash
# Start embedding TEI (manual)
text-embeddings-router --model-id nomic-ai/CodeRankEmbed --port 8080 &

# Start reranker TEI (manual)
text-embeddings-router --model-id BAAI/bge-reranker-base --port 8081 &
```

**Limitations:**
- ❌ No automatic model switching (must run dual embedding instances)
- ❌ Must manage processes manually
- ❌ More memory usage (two embedding models loaded)
- ❌ More complex configuration

**Recommendation:** Use tei-wrapper for automatic management.

### Custom Reranker Endpoint

In rare cases, you might want to run the reranker on a separate machine:

**Not supported.** The tei-wrapper manages both embedding and reranker TEI instances on the same host. For distributed deployments, deploy separate tei-wrapper instances and load balance at the code-scout level.

## Next Steps

- **[TEI_WRAPPER.md](TEI_WRAPPER.md)** - Complete tei-wrapper documentation
- **[TEI_SETUP.md](TEI_SETUP.md)** - Platform-specific TEI installation
- **[Main README](../../README.md)** - Code Scout overview and features
