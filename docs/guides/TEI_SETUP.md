# TEI (Text Embeddings Inference) Setup for Code Scout

This guide covers setting up HuggingFace Text Embeddings Inference (TEI) for Code Scout on Apple Silicon (M1/M2/M3 Macs).

## Why TEI?

**TEI is the recommended embedding server for Apple Silicon** because:

- ✅ **Native Metal support** - Runs efficiently on M1/M2/M3 GPUs
- ✅ **Fast startup** - Models load in ~2-3 seconds
- ✅ **Purpose-built for embeddings** - Optimized specifically for embedding inference
- ✅ **OpenAI-compatible API** - Exposes `/v1/embeddings` endpoint out of the box
- ✅ **Lightweight models** - CodeRankEmbed + nomic-embed-text use only ~524MB RAM total
- ✅ **Excellent performance** - CodeRankEmbed achieves 77.9 MRR on CodeSearchNet (SOTA for its size)

**Alternatives that DON'T work on M2:**
- ❌ TGI (Text Generation Inference) - Requires CUDA/ROCm, no Metal support
- ❌ vLLM - Requires CUDA/ROCm, slow startup (30-60s)

## Prerequisites

- macOS with Apple Silicon (M1/M2/M3)
- Rust toolchain (will be installed in this guide)
- ~2GB free disk space for TEI and models
- ~1GB free RAM

## Installation

### Step 1: Install Rust

```bash
# Install Rust toolchain
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Activate Rust environment
source $HOME/.cargo/env

# Verify installation
rustc --version
cargo --version
```

### Step 2: Clone and Build TEI

```bash
# Clone HuggingFace TEI repository
git clone https://github.com/huggingface/text-embeddings-inference
cd text-embeddings-inference

# Build with Metal support (takes ~5-10 minutes)
cargo install --path router -F metal

# Verify installation
text-embeddings-router --help
```

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

## Running TEI

### Why Two Instances?

TEI does not support dynamic model switching at runtime. The model is specified at startup and remains loaded for the lifetime of the process. Therefore, Code Scout's two-pass embedding system requires **two separate TEI instances** running on different ports.

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

Code Scout needs to know about your two TEI instances. Currently, the CLI expects a single endpoint and switches models. **We need to update Code Scout to support separate endpoints for code vs text models.**

### Temporary Workaround (Until Code Scout Supports Dual Endpoints)

Run TEI with the same model for both passes:

```bash
# Use CodeRankEmbed for both code and docs
text-embeddings-router --model-id nomic-ai/CodeRankEmbed --port 11434
```

Then configure Code Scout:

```json
{
  "endpoint": "http://localhost:11434/v1",
  "code_model": "nomic-ai/CodeRankEmbed",
  "text_model": "nomic-ai/CodeRankEmbed"
}
```

**Note:** This is not optimal but works. The proper solution requires Code Scout to support separate `code_endpoint` and `text_endpoint` configuration.

### Future Configuration (When Dual Endpoints Supported)

```json
{
  "code_endpoint": "http://localhost:8001/v1",
  "text_endpoint": "http://localhost:8002/v1",
  "code_model": "nomic-ai/CodeRankEmbed",
  "text_model": "nomic-ai/nomic-embed-text-v1.5"
}
```

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

*Estimated based on SOTA claim

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

## Comparison: TEI vs Ollama

| Feature | TEI | Ollama |
|---------|-----|--------|
| **Apple Silicon** | ✅ Metal | ✅ Native |
| **Startup Time** | ~2-3s | ~1-2s |
| **Concurrency** | High (6-10 workers) | Low (2 workers max) |
| **Model Switching** | ❌ Need 2 instances | ✅ Automatic |
| **Memory (both models)** | ~524MB | ~524MB |
| **Indexing Speed** | Fast | Slow |
| **Setup Complexity** | Moderate (Rust build) | Easy (binary install) |
| **Use Case** | Production, performance | Development, simplicity |

## Next Steps

- See [OLLAMA_SETUP.md](OLLAMA_SETUP.md) for simpler alternative
- See [DEVELOPERS.md](../DEVELOPERS.md) for contributing to Code Scout
- See [README.md](../../README.md) for general usage

## Future Improvements

**Code Scout enhancements needed:**

1. **Dual endpoint support** - Allow separate `code_endpoint` and `text_endpoint` in config
2. **Automatic TEI detection** - Auto-detect TEI vs Ollama and adjust defaults
3. **Process management** - Built-in commands to start/stop TEI instances
4. **Health checks** - Verify TEI endpoints are responding before indexing

**TEI wishlist:**

1. **Dynamic model loading** - Runtime model switching via API (may never happen)
2. **Multi-model serving** - Single process serving multiple models (alternative solution)
