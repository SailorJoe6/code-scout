# TEI (Text Embeddings Inference) Setup for Code Scout

This guide covers setting up HuggingFace Text Embeddings Inference (TEI) for Code Scout on all platforms.

## Why TEI?

**TEI is the recommended embedding server for Code Scout** because:

- ✅ **Cross-platform** - Works on Mac (Metal), Linux/Windows (CUDA), even CPU-only
- ✅ **Fast startup** - Models load in ~2-3 seconds
- ✅ **Purpose-built for embeddings** - Optimized specifically for embedding inference
- ✅ **OpenAI-compatible API** - Exposes `/v1/embeddings` endpoint out of the box
- ✅ **Excellent performance** - CodeRankEmbed achieves 77.9 MRR on CodeSearchNet
- ✅ **Model hot-swapping** - Via Code Scout's TEI wrapper (single model at a time, lower memory)

**Why not vLLM?**
- ❌ Requires CUDA (no Metal/CPU support)
- ❌ Slow startup (30-60s)
- ❌ Higher memory usage

## Installation

Choose your installation method based on your platform and hardware:

### macOS (Apple Silicon: M1/M2/M3/M4)

**TEI has native Apple Metal acceleration** - no Docker required!

#### Option 1: Homebrew (Recommended)

```bash
# Install TEI via Homebrew
brew install text-embeddings-inference

# Verify installation
text-embeddings-router --version
```

**Why this is best for Mac:**
- ✅ Native Metal GPU acceleration
- ✅ Automatic updates via `brew upgrade`
- ✅ Cleanest installation
- ✅ No Docker overhead
- ✅ Model switching: **2-5 seconds**

**Note:** Current version is 1.8.3 (as of Jan 2026)

#### Option 2: Build from Source

See "Build from Source (Advanced)" section below for instructions.

#### Example Usage

```bash
text-embeddings-router \
  --model-id nomic-ai/nomic-embed-text-v1.5 \
  --port 8080 \
  --max-batch-tokens 2048
```

**Notes for Mac:**
- Run **one model at a time** due to unified memory architecture
- Metal acceleration provides excellent performance
- Typical model load time: 2-5 seconds

---

### Linux / Windows with NVIDIA GPU

**Docker is the recommended method** for NVIDIA GPU systems.

**Note for Linux ARM64 (aarch64):** The official TEI Docker images do not currently publish ARM64 manifests, so `docker pull ghcr.io/huggingface/text-embeddings-inference:latest` will fail with "no matching manifest for linux/arm64". In that case, use the "Build from Source (Advanced)" section below and a CUDA-enabled feature like `candle-cuda-volta`.

#### Requirements

```bash
# Install NVIDIA Container Toolkit (Ubuntu/Debian)
sudo apt-get install nvidia-container-toolkit
sudo systemctl restart docker
```

#### Run TEI with CUDA Acceleration

```bash
# Pull the GPU-enabled image
docker pull ghcr.io/huggingface/text-embeddings-inference:latest

# Run with GPU acceleration
docker run --gpus all -p 8080:80 \
  -v $HOME/.cache/huggingface:/data \
  ghcr.io/huggingface/text-embeddings-inference:latest \
  --model-id nomic-ai/nomic-embed-text-v1.5 \
  --max-batch-tokens 32768
```

**Why Docker for NVIDIA GPU:**
- ✅ **Full CUDA acceleration**
- ✅ Pre-configured runtime environment
- ✅ Best performance for high-throughput pipelines
- ✅ Fastest model load (1-3 seconds) + batching
- ✅ Easy deployment and updates

---

### CPU-Only Systems

If no GPU is available, TEI still works (slower but functional).

#### CPU Docker Image

```bash
# Pull CPU-only image
docker pull ghcr.io/huggingface/text-embeddings-inference:cpu-latest

# Run without GPU
docker run -p 8080:80 \
  -v $HOME/.cache/huggingface:/data \
  ghcr.io/huggingface/text-embeddings-inference:cpu-latest \
  --model-id nomic-ai/nomic-embed-text-v1.5
```

**Notes:**
- ⚠️ **No GPU acceleration** - slower than Metal or CUDA
- ✅ Works on any x86_64 system
- ⏱️ Model load time: 5-10 seconds
- 💡 Fine for development and small-scale use

---

### Build from Source (Advanced)

For developers who need custom builds:

**Linux ARM64 + NVIDIA GPU (tested):** build from source with CUDA and avoid flash-attn if it fails to compile on your GPU/driver. The `candle-cuda-volta` feature uses CUDA without flash-attn and is a good fallback.

```bash
# Prereqs (Ubuntu/Debian)
sudo apt-get update
sudo apt-get install -y libssl-dev pkg-config

# Install Rust toolchain
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source $HOME/.cargo/env

# Clone TEI repository
git clone https://github.com/huggingface/text-embeddings-inference
cd text-embeddings-inference

# Build for your platform:

# Mac (Apple Silicon) - Metal support
cargo install --path router -F metal

# Linux/Windows - CUDA support
cargo install --path router -F candle-cuda

# Linux ARM64 + CUDA fallback (no flash-attn)
cargo install --path router -F candle-cuda-volta

# Any platform - CPU only
cargo install --path router

# Verify
text-embeddings-router --version
```

---

### Platform Comparison

| Platform | Install Method | GPU Acceleration | Model Load Time | Best For |
|----------|---------------|------------------|-----------------|----------|
| **Mac (Apple Silicon)** | `brew install` | **Metal (Yes!)** | 2-5s | Development, best Mac experience |
| **NVIDIA GPU** | Docker `:latest` | **CUDA (Full)** | 1-3s | Production, highest performance |
| **CPU-only** | Docker `:cpu-latest` | No | 5-10s | Development, no GPU available |

## Model Selection

Code Scout uses a **two-model architecture** for optimal results:

### For Code Embeddings: nomic-ai/CodeRankEmbed

- **Size:** 137M parameters, 521MB
- **Performance:** 77.9 MRR on CodeSearchNet (SOTA for size class)
- **Languages:** Python, Java, Ruby, PHP, JavaScript, Go
- **Context:** 8192 tokens
- **Use for:** Indexing code files (.py, .go, .js, .java, etc.)

### For Documentation Embeddings: nomic-ai/nomic-embed-text-v1.5

- **Size:** 137M parameters, 262MB
- **Performance:** Excellent for text/documentation retrieval
- **Context:** 2048 tokens
- **Use for:** Indexing documentation (.md, .txt, .rst, etc.)

**Total memory footprint:** ~524MB for both models running simultaneously

**Note:** The larger `nomic-ai/nomic-embed-code` model is not currently supported by TEI. It requires a powerful GPU and lots of RAM in Ollama; for TEI or lower-power machines, prefer `nomic-ai/CodeRankEmbed`.

## Running TEI with Code Scout

Code Scout needs different embedding models for code vs documentation. There are two approaches:

### Option A: TEI Wrapper (Recommended)

**Use the Code Scout TEI wrapper** for automatic model hot-swapping:

```bash
# Build the wrapper (from Code Scout repo)
cd cmd/tei-wrapper
go build -o tei-wrapper .

# Start the wrapper (defaults to port 11435 to avoid Ollama conflicts)
./tei-wrapper
```

**How it works:**
- Single TEI process with one model loaded at a time
- Automatically detects model changes and restarts TEI
- Lower memory usage (~4-8GB for single model vs 8-16GB for dual)
- Ollama-compatible API on port 11435 (configurable)

**Advantages:**
- ✅ Lower memory usage (single model at a time)
- ✅ Automatic model switching
- ✅ Simpler process management
- ✅ Better for development machines

**Disadvantages:**
- ⏱️ ~2-3 second delay during model switches

See [cmd/tei-wrapper/README.md](../../cmd/tei-wrapper/README.md) for detailed wrapper documentation.

### Option B: Dual TEI Instances (Advanced)

**Run two separate TEI instances** for maximum performance (no switching delay):

**Why two instances?** TEI does not support dynamic model switching at runtime. The model is specified at startup and remains loaded for the lifetime of the process. To switch models, you must stop the current TEI process and start a new one.

**Model Switching (Serial Pipelines):**

If your workflow processes code embeddings first, then text embeddings sequentially:

```bash
# Process code embeddings
text-embeddings-router --model-id nomic-ai/CodeRankEmbed

# Stop when done (Ctrl+C or pkill)
pkill text-embeddings-router

# Switch to text model
text-embeddings-router --model-id nomic-ai/nomic-embed-text-v1.5
```

**Typical Model Load Times:**
- **Mac (Metal):** 2-5 seconds
- **NVIDIA GPU (CUDA):** 1-3 seconds
- **CPU-only:** 5-10 seconds

### Start Code Embeddings Server

```bash
# Terminal 1: Code embeddings on port 8001
text-embeddings-router \
  --model-id nomic-ai/CodeRankEmbed \
  --port 8001 \
  --json-output
```

**First run:** TEI will download the model (~521MB) from HuggingFace. This takes a few minutes.

**Subsequent runs:** Model loads in ~2-3 seconds from cache.

### Start Text Embeddings Server

```bash
# Terminal 2: Text embeddings on port 8002
text-embeddings-router \
  --model-id nomic-ai/nomic-embed-text-v1.5 \
  --port 8002 \
  --json-output
```

**First run:** Downloads model (~262MB).

**Subsequent runs:** Loads in ~2-3 seconds.

### Verify TEI is Running

```bash
# Check code embeddings endpoint
curl http://localhost:8001/health

# Check text embeddings endpoint
curl http://localhost:8002/health

# Test embedding generation
curl http://localhost:8001/v1/embeddings \
  -H "Content-Type: application/json" \
  -d '{
    "model": "nomic-ai/CodeRankEmbed",
    "input": "def hello(): print(\"world\")"
  }'
```

## Configure Code Scout

### With TEI Wrapper (Option A)

The wrapper is Ollama-compatible, so Code Scout works automatically:

```bash
# The wrapper runs on port 11435 by default
# No additional configuration needed (or set --port 11434 if Ollama is stopped)!

# Index your repository
code-scout index

# Search
code-scout search "authentication middleware"
```

The wrapper automatically switches between models as needed:
- Code files → Uses code embedding model
- Documentation → Uses text embedding model

### With Dual TEI Instances (Option B)

Configure Code Scout to use separate endpoints:

```bash
# Set environment variables
export CODE_EMBEDDINGS_URL=http://localhost:8001
export TEXT_EMBEDDINGS_URL=http://localhost:8002

# Or use command-line flags
code-scout index \
  --code-embeddings-url http://localhost:8001 \
  --text-embeddings-url http://localhost:8002
```

**Note:** Dual endpoint support may require Code Scout configuration updates.

## OpenAI-Compatible API

TEI exposes an **OpenAI-compatible `/v1/embeddings` endpoint** - any HTTP client works!

### API Format

**Request:**
```json
{
  "model": "nomic-ai/nomic-embed-text-v1.5",
  "input": "Hello world"
}
```

**Response:**
```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [0.123, -0.456, ...],
      "index": 0
    }
  ],
  "model": "nomic-ai/nomic-embed-text-v1.5",
  "usage": {
    "prompt_tokens": 1,
    "total_tokens": 1
  }
}
```

### Example: cURL

```bash
curl http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -d '{"input":"Hello world","model":"nomic-ai/nomic-embed-text-v1.5"}'
```

### Language Support

Any language with HTTP support can use TEI:
- **Go:** `net/http`, any HTTP client
- **Python:** `requests`, `httpx`, OpenAI SDK
- **Node.js:** `fetch`, `axios`, OpenAI SDK
- **Rust:** `reqwest`, `hyper`

**No special SDK required** - just standard HTTP requests!

## Usage

### Index Repository

```bash
./code-scout index --workers 6 --batch-size 6
```

**Performance:** With TEI on M2, you can use higher concurrency than Ollama:
- `--workers 6-10` for most repos
- `--batch-size 6-8` for optimal throughput

### Search Repository

```bash
./code-scout search "authentication middleware"
./code-scout search "error handling" --mode code
./code-scout search "architecture overview" --mode docs
```

## Process Management

### Running TEI in Background

```bash
# Start code embeddings in background
nohup text-embeddings-router \
  --model-id nomic-ai/CodeRankEmbed \
  --port 8001 \
  > tei-code.log 2>&1 &

# Start text embeddings in background
nohup text-embeddings-router \
  --model-id nomic-ai/nomic-embed-text-v1.5 \
  --port 8002 \
  > tei-text.log 2>&1 &

# Check processes
ps aux | grep text-embeddings-router

# View logs
tail -f tei-code.log
tail -f tei-text.log
```

### Stop TEI Servers

```bash
# Find and kill processes
pkill -f text-embeddings-router

# Or kill specific ports
lsof -ti:8001 | xargs kill
lsof -ti:8002 | xargs kill
```

## Troubleshooting

### TEI won't start: "command not found"

**Solution:** Ensure Rust cargo bin is in PATH:

```bash
source $HOME/.cargo/env
# Or add to ~/.zshrc or ~/.bashrc:
export PATH="$HOME/.cargo/bin:$PATH"
```

### Build fails: Metal framework not found

**Solution:** Ensure you're on Apple Silicon Mac:

```bash
uname -m  # Should show "arm64"
```

If on Intel Mac, TEI won't work. Use Ollama instead (see OLLAMA_SETUP.md).

### Build fails: OpenSSL headers not found (Linux)

**Symptom:**
```
error: failed to run custom build command for `openssl-sys`
The system library `openssl` required by crate `openssl-sys` was not found.
```

**Solution (Ubuntu/Debian):**
```bash
sudo apt-get update
sudo apt-get install -y libssl-dev pkg-config
```

### Build fails: flash-attn/CUTLASS errors (CUDA)

**Symptom:** NVCC errors mentioning `candle-flash-attn`, `cutlass`, or missing `PFN_cuTensorMap*` symbols.

**Solution:** Rebuild with the `candle-cuda-volta` feature to disable flash-attn:
```bash
cargo install --path router -F candle-cuda-volta
```

### Runtime error: compute cap not compatible (CUDA 12.x GPUs)

**Symptom:**
```
Could not start backend: Runtime compute cap 121 is not compatible with compile time compute cap 121
```

**Cause:** Older TEI/Candle versions only whitelist compute caps up to 9.0.

**Solution:** Patch TEI to allow 12.x compute caps, then rebuild:
```rust
// backends/candle/src/compute_cap.rs
// add:
(120..=129, 120..=129) => true,
```
Then rebuild with CUDA (e.g. `candle-cuda-volta`).

### Model download fails or is slow

**Solution:** HuggingFace downloads can be slow. Use a VPN or retry:

```bash
# Downloads are cached in ~/.cache/huggingface
ls -lh ~/.cache/huggingface/hub/
```

### High memory usage

**Expected memory per instance:**
- CodeRankEmbed: ~521MB
- nomic-embed-text-v1.5: ~262MB

**Total for both:** ~800MB including overhead

If using more, check for multiple processes:

```bash
ps aux | grep text-embeddings-router
```

### Port already in use

**Solution:** Change ports or kill existing process:

```bash
lsof -ti:8001 | xargs kill
# Then restart TEI on port 8001
```

## Performance Benchmarks

### Model Performance (CodeSearchNet)

| Model | Size | MRR | Memory |
|-------|------|-----|--------|
| CodeRankEmbed | 137M | 77.9 | 521MB |
| nomic-embed-code | 7B | ~83-85* | 26GB |

*Estimated based on SOTA claim. `nomic-embed-code` is Ollama-only and not currently supported by TEI.

**Trade-off:** CodeRankEmbed sacrifices ~5-7% accuracy for 50x smaller size and ability to run two models simultaneously on M2.

### Startup Times

- **TEI (first run):** 5-10 minutes (model download)
- **TEI (subsequent runs):** 2-3 seconds
- **Ollama (first run):** 2-5 minutes (model download)
- **Ollama (subsequent runs):** 1-2 seconds (but slower inference)

### Indexing Performance (M2 MacBook)

**Small repo (~50 files, ~5K chunks):**
- TEI (--workers 6 --batch-size 6): ~2-3 minutes
- Ollama (--workers 2 --batch-size 2): ~5-7 minutes

**Large repo (~500 files, ~50K chunks):**
- TEI (--workers 6 --batch-size 6): ~20-30 minutes
- Ollama (--workers 2 --batch-size 2): ~60-90 minutes

## Comparison: TEI Wrapper vs Dual TEI vs Ollama

| Feature | TEI Wrapper | Dual TEI | Ollama |
|---------|-------------|----------|--------|
| **Platforms** | All (Mac/Linux/Win) | All | All |
| **GPU Acceleration** | ✅ Metal/CUDA | ✅ Metal/CUDA | ✅ Metal/CUDA |
| **Startup Time** | ~2-3s (per switch) | ~2-3s (once) | ~1-2s |
| **Concurrency** | High (6-10 workers) | High (6-10) | Low (2 max) |
| **Model Switching** | ✅ Automatic | ❌ Manual | ✅ Automatic |
| **Memory (single model)** | ~4-8GB | ~8-16GB | ~4-8GB |
| **Switching Delay** | ~2-3s | None | Minimal |
| **Indexing Speed** | Fast | Fastest | Slow |
| **Setup Complexity** | Easy (brew/binary) | Moderate | Easy |
| **Best For** | Most users | Large repos, servers | Simplicity over speed |

**Recommendation:**
- **Development/Most Users:** TEI Wrapper (Option A)
- **Production/Large Repos:** Dual TEI (Option B)
- **Simplicity/Small Repos:** Ollama (see [OLLAMA_SETUP.md](OLLAMA_SETUP.md))

## Next Steps

- See [OLLAMA_SETUP.md](OLLAMA_SETUP.md) for simpler alternative
- See [DEVELOPERS.md](../DEVELOPERS.md) for contributing to Code Scout
- See [README.md](../../README.md) for general usage

## Automation Tools

Code Scout provides two tools to streamline your workflow:

### TEI Wrapper

The TEI wrapper provides Ollama-like model hot-swapping with better performance than Ollama:

- **Single TEI process** - Lower memory usage (4-8GB vs 8-16GB for dual TEI)
- **Automatic model switching** - Detects model changes and restarts TEI
- **Optional preloading** - Switches to preferred model when idle
- **OpenAI-compatible API** - Works with existing code

**Learn more:** [TEI_WRAPPER.md](TEI_WRAPPER.md)

### Background Daemon

The background daemon automatically re-indexes your codebase when files change:

- **Zero manual indexing** - AI agents just search, never index
- **Always fresh results** - Automatic debouncing and incremental updates
- **File watching** - Detects create, write, delete, and rename events
- **Respects ignore patterns** - Uses `.gitignore` and `.code-scout-ignore`

**Learn more:** [BACKGROUND_DAEMON.md](BACKGROUND_DAEMON.md)

### Recommended Setup for AI Agents

For the best experience with AI coding agents like Claude Code:

```bash
# 1. Start TEI wrapper (one-time setup)
./tei-wrapper --idle-preload &

# 2. Start background daemon (one-time per session)
code-scout daemon start

# 3. AI agents only use search (indexing is automatic)
code-scout search "authentication"
```

This setup provides:
- ✅ Automatic indexing on file changes
- ✅ Automatic model switching (code vs docs)
- ✅ Always fresh search results
- ✅ Minimal memory usage (~4-8GB)
- ✅ High performance (6-10 workers)

## What's Next?

**Related Documentation:**
- [TEI_WRAPPER.md](TEI_WRAPPER.md) - TEI wrapper setup and usage
- [BACKGROUND_DAEMON.md](BACKGROUND_DAEMON.md) - Background daemon for auto-indexing
- [OLLAMA_SETUP.md](OLLAMA_SETUP.md) - Simpler alternative to TEI
- [DEVELOPERS.md](../../DEVELOPERS.md) - Build and development setup
