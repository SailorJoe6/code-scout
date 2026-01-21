# TEI Wrapper Guide

The TEI wrapper is a lightweight HTTP server that provides OpenAI-compatible API access to TEI (Text Embeddings Inference) with automatic model hot-swapping. This guide covers installation, usage, and configuration.

## Overview

The TEI wrapper solves a key problem with using TEI for Code Scout's two-pass embedding architecture (code embeddings + documentation embeddings):

### The Problem

Code Scout uses two different embedding models:
- **Code model** (nomic-ai/CodeRankEmbed) - For code files
- **Text model** (nomic-ai/nomic-embed-text-v1.5) - For documentation

**Option 1: Run two TEI instances**
- Memory usage: 8-16GB (both models loaded simultaneously)
- Performance: Fastest (no model switching)
- Complexity: Must manage two processes, two ports

**Option 2: Use Ollama**
- Memory usage: 4-8GB (one model at a time)
- Performance: Slow (limited concurrency, model switching overhead)
- Complexity: Simple (single process)

### The Solution: TEI Wrapper

The TEI wrapper provides the best of both worlds:

| Feature | TEI Wrapper | Dual TEI | Ollama |
|---------|-------------|----------|--------|
| **Memory Usage** | 4-8GB | 8-16GB | 4-8GB |
| **Performance** | Fast | Fastest | Slow |
| **Concurrency** | High (6-10 workers) | High (6-10) | Low (2 max) |
| **Model Switching** | Automatic (~2-3s) | N/A | Automatic |
| **Setup Complexity** | Moderate | Moderate | Easy |
| **API** | OpenAI-compatible | Native TEI | Ollama API |

## How It Works

The wrapper is a thin HTTP proxy that:

1. **Exposes OpenAI-compatible `/v1/embeddings` endpoint** on port 11434 (Ollama-compatible)
2. **Exposes TEI-compatible `/rerank` endpoint** for cross-encoder reranking (optional)
3. **Manages TEI processes** - one for embeddings (port 8080), optionally one for reranking (port 8081)
4. **Detects model changes** in incoming embedding requests
5. **Automatically restarts TEI** with the requested model
6. **Optional: Preloads preferred model** when idle to minimize switching delays

### Model Switching Flow

```
Client Request (model: text-v1.5)
       ↓
Wrapper detects model change
       ↓
Stop current TEI (code model)
       ↓
Start new TEI (text model) ← 2-3 seconds
       ↓
Wait for TEI ready
       ↓
Forward request to TEI
       ↓
Return embeddings to client
```

During model switching (~2-3 seconds), the wrapper returns `503 Service Unavailable` with `Retry-After: 5` header.

## Installation

### Prerequisites

1. **TEI installed** - See [TEI_SETUP.md](TEI_SETUP.md) for installation instructions
2. **Go 1.17+** - For building the wrapper

### Build the Wrapper

```bash
cd cmd/tei-wrapper
go build -o tei-wrapper .

# Move to PATH (optional)
sudo mv tei-wrapper /usr/local/bin/
```

## Usage

### Basic Usage

Start the wrapper with default settings:

```bash
./tei-wrapper
```

**Default configuration:**
- Listen port: `11434` (Ollama-compatible)
- TEI internal port: `8080`
- Initial model: `nomic-ai/nomic-embed-text-v1.5` (from config defaults)
- Idle preload: Disabled

### Command Line Options

```bash
./tei-wrapper [options]
```

**Available flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--port` | int | `11434` | Wrapper listen port (Ollama-compatible) |
| `--tei-port` | int | `8080` | TEI internal port |
| `--tei-binary` | string | `text-embeddings-router` | Path to TEI binary |
| `--model` | string | config `text_model` | Initial model to load (CLI overrides config) |
| `--idle-preload` | bool | `false` | Enable idle-based model preloading |
| `--idle-timeout` | duration | `30s` | Idle time before preloading preferred model |
| `--max-batch-tokens` | int | `8192` | Maximum batch tokens (controls memory usage, lower = less RAM) |
| `--rerank-port` | int | `8081` | Port for dedicated reranker TEI instance (optional) |
| `--rerank-model` | string | `""` | Model ID for reranker (deprecated; prefer `.code-scout.json`) |

### Example Configurations

**Development (with idle preload):**
```bash
./tei-wrapper \
  --idle-preload \
  --idle-timeout 30s
```

**Production (custom ports):**
```bash
./tei-wrapper \
  --port 8081 \
  --tei-port 9090 \
  --model nomic-ai/CodeRankEmbed
```

**Custom TEI binary:**
```bash
./tei-wrapper \
  --tei-binary /opt/homebrew/bin/text-embeddings-router
```

**With reranking enabled (config-first):**
```bash
# .code-scout.json
{
  "endpoint": "http://localhost:11434",
  "text_model": "nomic-ai/nomic-embed-text-v1.5",
  "rerank_model": "BAAI/bge-reranker-base"
}

./tei-wrapper
```

The wrapper starts the embedding TEI immediately and starts the reranker TEI on the first `/rerank` request.

**Legacy (deprecated):**
```bash
./tei-wrapper --rerank-model BAAI/bge-reranker-base
```

See [RERANKER_SETUP.md](RERANKER_SETUP.md) for complete reranking documentation.

## Idle Preloading

The wrapper supports optional idle-based preloading to minimize model switching overhead.

### How Idle Preload Works

1. After each request, the wrapper starts an idle timer
2. If no requests arrive within the timeout period (default: 30s), the wrapper detects idle state
3. The wrapper automatically switches to the **preferred model** (always `nomic-ai/nomic-embed-text-v1.5`)
4. Next search run starts immediately with the text model already loaded

**Note:** The wrapper now defaults to the text model (for search-heavy workflows in single-model mode). During indexing, it switches to `nomic-ai/CodeRankEmbed` only when needed.

### When to Use Idle Preload

**✅ Enable when:**
- Development workflows with periodic searching (search → code → search → repeat)
- You search more frequently than you index
- You want faster search startup

**❌ Disable when:**
- Production environments with constant traffic (never idle)
- You need both models equally often
- Memory usage is critical (preload triggers extra model switches)

### Example Usage

```bash
# Enable with 30-second timeout
./tei-wrapper --idle-preload --idle-timeout 30s

# Enable with longer timeout
./tei-wrapper --idle-preload --idle-timeout 2m
```

**Behavior:**
- 30 seconds after last request → switches to code model
- Next indexing run → code model already loaded (instant start)
- Documentation pass → switches to text model (~2-3s delay)
- 30 seconds after docs pass → switches back to code model

## API Reference

### POST /v1/embeddings

OpenAI-compatible endpoint for generating embeddings from the embedding TEI instance.

**Request:**
```json
{
  "model": "nomic-ai/nomic-embed-text-v1.5",
  "input": ["Hello world", "Semantic search"]
}
```

**Response (Success):**
```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [0.123, -0.456, ...],
      "index": 0
    },
    {
      "object": "embedding",
      "embedding": [0.789, -0.012, ...],
      "index": 1
    }
  ],
  "model": "nomic-ai/nomic-embed-text-v1.5",
  "usage": {
    "prompt_tokens": 2,
    "total_tokens": 2
  }
}
```

**Response (Model Switching):**
```
HTTP/1.1 503 Service Unavailable
Retry-After: 5

Model switch in progress, please retry
```

### POST /rerank

TEI-compatible endpoint for cross-encoder reranking. Available when `rerank_model` is configured (config or legacy flag); the reranker TEI starts on the first rerank request.

**Request:**
```json
{
  "query": "search query",
  "texts": ["document 1", "document 2", "document 3"],
  "raw_scores": false,
  "return_text": false
}
```

**Response (Success):**
```json
[
  {"index": 1, "score": 0.95},
  {"index": 0, "score": 0.87},
  {"index": 2, "score": 0.71}
]
```

Results are automatically sorted by relevance score (highest first).

**Response (Reranker Not Configured):**
```
HTTP/1.1 404 Not Found

Reranker not enabled
```

**Response (Reranker Unavailable):**
```
HTTP/1.1 503 Service Unavailable
Retry-After: 5

Reranker TEI is not available
```

See [RERANKER_SETUP.md](RERANKER_SETUP.md) for complete reranking setup and usage.

### GET /health

Health check endpoint with current embedding model and optional reranker status.

**Response (Healthy, No Reranker):**
```json
{
  "status": "ok",
  "embedding_model": "nomic-ai/CodeRankEmbed"
}
```

**Response (Healthy, With Reranker):**
```json
{
  "status": "ok",
  "embedding_model": "nomic-ai/CodeRankEmbed",
  "reranker": {
    "enabled": true,
    "healthy": true,
    "model": "BAAI/bge-reranker-base",
    "port": 8081
  }
}
```

**Response (Switching):**
```json
{
  "status": "switching",
  "embedding_model": "nomic-ai/CodeRankEmbed"
}
```

**Response (Unhealthy):**
```json
{
  "status": "unhealthy",
  "embedding_model": "nomic-ai/CodeRankEmbed",
  "error": "TEI is not responding"
}
```

**Reranker health fields:**
- `enabled`: `true` if `rerank_model` is configured (config or legacy flag)
- `healthy`: `true` if reranker TEI instance is responding
- `model`: The reranker model ID
- `port`: The port the reranker TEI is running on

## Using with Code Scout

The wrapper is Ollama-compatible, so Code Scout works without any configuration changes!

### Default Configuration (No Config Needed)

Code Scout's default endpoint is `http://localhost:11434`, which matches the wrapper's default port:

```bash
# Just run code-scout with the wrapper running
./code-scout index --workers 6 --batch-size 6
```

### Custom Configuration

If you changed the wrapper's port:

```bash
# Via environment variable
export CODE_SCOUT_ENDPOINT=http://localhost:8081
./code-scout index

# Via command line flag
./code-scout index --endpoint http://localhost:8081
```

### Configuration File

Create `.code-scout.json` in your repo:

```json
{
  "endpoint": "http://localhost:11434",
  "code_model": "nomic-ai/CodeRankEmbed",
  "text_model": "nomic-ai/nomic-embed-text-v1.5",
  "rerank_model": "BAAI/bge-reranker-base",
  "rerank_top_k": 25
}
```

See [RERANKER_SETUP.md](RERANKER_SETUP.md) for reranking configuration details.

## Supported Models

Any TEI-compatible embedding model works. Recommended models:

### For Code Embeddings

**nomic-ai/CodeRankEmbed** (137M params, 521MB)
- Default code model (fast, compact)
- Requires 1-2GB RAM
- Best for memory-constrained systems

**nomic-ai/nomic-embed-code** (7B params, 26GB)
- Higher accuracy for code (large model)
- Requires a powerful GPU and lots of RAM in Ollama
- Not currently supported by TEI (use CodeRankEmbed instead)
- Optimized for: Python, Java, Ruby, PHP, JavaScript, Go

### For Documentation Embeddings

**nomic-ai/nomic-embed-text-v1.5** (137M params, 262MB)
- Excellent for general text
- Requires 1-2GB RAM
- Optimized for: Markdown, plain text, documentation

## Process Management

### Running in Foreground

```bash
# Start wrapper (blocks terminal)
./tei-wrapper

# Stop with Ctrl+C
^C
Shutting down...
TEI stopped gracefully
```

### Running in Background

```bash
# Start in background
nohup ./tei-wrapper > tei-wrapper.log 2>&1 &

# Check process
ps aux | grep tei-wrapper

# View logs
tail -f tei-wrapper.log

# Stop
pkill tei-wrapper
```

### systemd Service (Linux)

Create `/etc/systemd/system/tei-wrapper.service`:

```ini
[Unit]
Description=TEI Wrapper for Code Scout
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/home/your-user
ExecStart=/usr/local/bin/tei-wrapper
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable tei-wrapper
sudo systemctl start tei-wrapper
sudo systemctl status tei-wrapper
```

### launchd (macOS)

Create `~/Library/LaunchAgents/com.codescout.tei-wrapper.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.codescout.tei-wrapper</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/tei-wrapper</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/tei-wrapper.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/tei-wrapper.err</string>
</dict>
</plist>
```

Load and start:

```bash
launchctl load ~/Library/LaunchAgents/com.codescout.tei-wrapper.plist
launchctl start com.codescout.tei-wrapper
```

## Troubleshooting

### "TEI binary not found"

**Problem:** Wrapper can't find `text-embeddings-router`

**Solution:**
```bash
# Check if TEI is installed
which text-embeddings-router

# If not in PATH, specify full path
./tei-wrapper --tei-binary /opt/homebrew/bin/text-embeddings-router
```

### "TEI failed to start"

**Problem:** TEI process won't start

**Checklist:**
1. TEI binary is executable: `chmod +x $(which text-embeddings-router)`
2. Port 8080 is available: `lsof -i :8080`
3. Model ID is valid (check [Hugging Face](https://huggingface.co/models))
4. Sufficient RAM available
5. `nomic-ai/nomic-embed-code` is Ollama-only and will not load in TEI

**Check TEI manually:**
```bash
text-embeddings-router \
  --model-id nomic-ai/CodeRankEmbed \
  --port 8080
```

### "Out of memory" errors

**Problem:** Not enough RAM for model or excessive memory usage during searches

**Solution:**
1. **Lower `--max-batch-tokens`** (most effective): `--max-batch-tokens 4096` or `2048`
2. Use smaller model: `--model nomic-ai/CodeRankEmbed`
3. Reduce batch size in code-scout: `--batch-size 4`
4. Close other applications
5. Check system RAM: `free -h` (Linux) or Activity Monitor (macOS)

**Memory tuning examples:**
```bash
# Low memory mode (2-4GB RAM)
./tei-wrapper --max-batch-tokens 2048

# Balanced mode (4-8GB RAM, default)
./tei-wrapper --max-batch-tokens 8192

# High throughput mode (12-16GB RAM)
./tei-wrapper --max-batch-tokens 16384
```

Memory usage scales quadratically with `--max-batch-tokens`. The default (8192) is optimized for single-query searches. If you see 20+ GB memory usage during searches, reduce to 4096 or 2048.

### Model switching is slow

**Problem:** 5+ seconds to switch models

**Likely causes:**
- Slow disk I/O (model cache reading)
- CPU-only mode (no GPU acceleration)
- Large model (nomic-embed-code 7B vs CodeRankEmbed 137M; nomic-embed-code is Ollama-only)

**Solutions:**
1. Enable idle preload: `--idle-preload`
2. Use smaller model: `--model nomic-ai/CodeRankEmbed`
3. Check TEI has GPU acceleration (Mac: Metal, Linux: CUDA)
4. Use SSD for model cache (`~/.cache/huggingface`)

### Requests failing during model switch

**Problem:** Getting 503 errors

**Expected behavior:** The wrapper returns `503 Service Unavailable` during model switching (~2-3 seconds)

**Solutions:**
1. Client should retry on 503 (most HTTP clients do this automatically)
2. Enable idle preload to minimize switching: `--idle-preload`
3. For high-throughput needs, use dual TEI instances instead

## Architecture: Dual TEI Instances

When reranking is enabled (via `rerank_model`), the wrapper manages two separate TEI processes:

```
┌─────────────────┐
│   code-scout    │
│                 │
│  Config:        │
│  endpoint:      │
│   localhost:    │
│   11434         │
└────────┬────────┘
         │
         │ Single endpoint
         ▼
┌─────────────────────────────────┐
│       tei-wrapper:11434         │
│                                 │
│  Routes:                        │
│  • /v1/embeddings → TEI:8080   │
│  • /rerank        → TEI:8081   │
│  • /health                      │
└────────┬────────────────┬───────┘
         │                │
         ▼                ▼
┌────────────────┐  ┌──────────────────┐
│  TEI:8080      │  │  TEI:8081        │
│                │  │                  │
│  Embeddings    │  │  Reranker        │
│  (hot-swap)    │  │  (dedicated)     │
└────────────────┘  └──────────────────┘
```

**Key points:**
- Code Scout only needs to know about the wrapper endpoint (localhost:11434)
- The wrapper internally routes `/v1/embeddings` to the embedding TEI
- The wrapper internally routes `/rerank` to the reranker TEI
- Embedding TEI hot-swaps between code and text models
- Reranker TEI runs a dedicated cross-encoder model (no swapping)

**Memory usage:**
- Single-model mode (no reranker): 4-8GB
- With reranker: +500MB to +2GB depending on model

See [RERANKER_SETUP.md](RERANKER_SETUP.md) for deployment details.

## Development Status

**Implemented:**
- ✅ OpenAI-compatible `/v1/embeddings` endpoint
- ✅ TEI-compatible `/rerank` endpoint (optional)
- ✅ Dual TEI instance management (embeddings + reranker)
- ✅ Model hot-swapping for embeddings (automatic TEI restart on model change)
- ✅ Health endpoint with embedding and reranker status
- ✅ 503 response during model switches
- ✅ Background preloading of preferred model on idle (optional)
- ✅ Idle detection with configurable timeout
- ✅ Graceful shutdown (SIGTERM/SIGINT)

**Future Enhancements (Slice 4 - Deferred):**
- Configuration file support (YAML/TOML)
- Request queuing during model switches
- Enhanced error handling and retry logic
- Metrics and monitoring endpoints (Prometheus)

## Architecture

For technical details on the wrapper's implementation, see:
- [cmd/tei-wrapper/README.md](../../cmd/tei-wrapper/README.md) - Developer documentation
- [cmd/tei-wrapper/main.go](../../cmd/tei-wrapper/main.go) - Implementation code

## Next Steps

- **For TEI installation:** See [TEI_SETUP.md](TEI_SETUP.md)
- **For reranking setup:** See [RERANKER_SETUP.md](RERANKER_SETUP.md)
- **For background daemon:** See [BACKGROUND_DAEMON.md](BACKGROUND_DAEMON.md)
- **For comparison with Ollama:** See [OLLAMA_SETUP.md](OLLAMA_SETUP.md)
- **For contributing:** See [DEVELOPERS.md](../../DEVELOPERS.md)
