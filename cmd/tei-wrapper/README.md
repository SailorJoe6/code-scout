# TEI Wrapper

A lightweight HTTP wrapper around [Text Embeddings Inference (TEI)](https://github.com/huggingface/text-embeddings-inference) that provides an OpenAI-compatible API with model hot-swapping capabilities.

## Why This Wrapper?

**Problems:**
- Running two TEI instances (code + text models) uses 8-16GB RAM
- Ollama doesn't handle concurrent requests well for embedding models
- Model switching in Ollama kills performance

**Solution:**
- Single TEI process with smart model hot-swapping
- OpenAI-compatible API for easy integration
- Better concurrency than Ollama
- Lower memory usage than dual TEI instances

## Installation

### 1. Install TEI

TEI is a Rust-based embedding server from Hugging Face. Choose one method:

#### Option A: Pre-built Binary (Fastest)

```bash
# Download the latest release for your platform
# macOS (ARM64):
curl -LO https://github.com/huggingface/text-embeddings-inference/releases/latest/download/text-embeddings-router-aarch64-apple-darwin
chmod +x text-embeddings-router-aarch64-apple-darwin
sudo mv text-embeddings-router-aarch64-apple-darwin /usr/local/bin/text-embeddings-router

# macOS (Intel):
curl -LO https://github.com/huggingface/text-embeddings-inference/releases/latest/download/text-embeddings-router-x86_64-apple-darwin
chmod +x text-embeddings-router-x86_64-apple-darwin
sudo mv text-embeddings-router-x86_64-apple-darwin /usr/local/bin/text-embeddings-router

# Linux (x86_64):
curl -LO https://github.com/huggingface/text-embeddings-inference/releases/latest/download/text-embeddings-router-x86_64-unknown-linux-gnu
chmod +x text-embeddings-router-x86_64-unknown-linux-gnu
sudo mv text-embeddings-router-x86_64-unknown-linux-gnu /usr/local/bin/text-embeddings-router
```

#### Option B: Using Docker

```bash
# Run TEI in a container (managed by wrapper)
docker pull ghcr.io/huggingface/text-embeddings-inference:latest
```

#### Option C: Build from Source

Requires Rust toolchain:

```bash
cargo install text-embeddings-router
```

### 2. Verify TEI Installation

```bash
text-embeddings-router --version
```

### 3. Build the Wrapper

```bash
cd cmd/tei-wrapper
go build -o tei-wrapper .
```

## Usage

### Basic Usage (Default Settings)

```bash
# Start wrapper with default model (nomic-embed-text-v1.5)
./tei-wrapper

# Wrapper will:
# - Start TEI on port 8080 (internal)
# - Listen on port 11434 (Ollama-compatible)
# - Load nomic-ai/nomic-embed-text-v1.5 by default
```

### Custom Model

```bash
# Start with code model
./tei-wrapper --model nomic-ai/nomic-embed-code

# Start with different port
./tei-wrapper --port 8081 --model nomic-ai/CodeRankEmbed
```

### Command Line Options

```
-port int
    Port to listen on (default: 11434, Ollama-compatible)
-tei-port int
    TEI internal port (default: 8080)
-tei-binary string
    Path to TEI binary (default: "text-embeddings-router")
-model string
    Initial model to load (default: "nomic-ai/nomic-embed-text-v1.5")
```

## API

### POST /v1/embeddings

OpenAI-compatible endpoint for generating embeddings.

**Request:**
```json
{
  "model": "nomic-ai/nomic-embed-text-v1.5",
  "input": ["Hello world", "Semantic search"]
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

### GET /health

Health check endpoint.

**Response:**
```json
{
  "status": "ok",
  "model": "nomic-ai/nomic-embed-text-v1.5"
}
```

## Supported Models

- **nomic-ai/nomic-embed-text-v1.5** - General text embeddings (137M params, 262MB)
- **nomic-ai/nomic-embed-code** - Code embeddings (7B params, ~8GB RAM)
- **nomic-ai/CodeRankEmbed** - Alternative code embeddings (137M params, 521MB)

All models are automatically downloaded from Hugging Face on first use.

## Using with code-scout

Configure code-scout to use the wrapper instead of Ollama:

```bash
# Set environment variable
export EMBEDDINGS_BASE_URL=http://localhost:11434

# Or use command line flag
code-scout index --embeddings-url http://localhost:11434
```

The wrapper is API-compatible with Ollama, so code-scout will work without any code changes!

## Development Status

**Current (Slice 1):**
- ✅ OpenAI-compatible /v1/embeddings endpoint
- ✅ Basic TEI process management
- ✅ Health check endpoint
- ✅ Request forwarding and response translation

**Coming Next:**
- ⏳ Slice 2: Model hot-swapping (auto-restart TEI when model changes)
- ⏳ Slice 3: Background pre-loading of next expected model
- ⏳ Slice 4: Configuration file, request queuing, enhanced error handling

## Troubleshooting

### "TEI binary not found"

Make sure `text-embeddings-router` is in your PATH or specify the full path:

```bash
./tei-wrapper --tei-binary /usr/local/bin/text-embeddings-router
```

### "TEI failed to start"

Check that:
1. TEI binary is executable: `chmod +x $(which text-embeddings-router)`
2. Port 8080 is available: `lsof -i :8080`
3. Model ID is valid (check Hugging Face)

### "Out of memory" errors

The nomic-embed-code 7B model requires ~8GB RAM. Try:
1. Use the smaller CodeRankEmbed model instead
2. Reduce batch size in code-scout
3. Close other applications

## License

Same as code-scout (see root LICENSE file).
