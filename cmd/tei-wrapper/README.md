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

See [TEI_SETUP.md](../../docs/guides/TEI_SETUP.md) for complete TEI installation instructions for all platforms (macOS, Linux, Windows).

### 2. Build the Wrapper

```bash
cd cmd/tei-wrapper
go build -o tei-wrapper .
```

## Usage

### Basic Usage (Default Settings)

```bash
# Start wrapper with default model (nomic-embed-code)
./tei-wrapper

# Wrapper will:
# - Start TEI on port 8080 (internal)
# - Listen on port 11434 (Ollama-compatible)
# - Load nomic-ai/nomic-embed-code by default
# - Automatically switch models when requested via API
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
    Initial model to load (default: "nomic-ai/nomic-embed-code")
-idle-preload bool
    Enable idle-based preloading of code model (default: false)
-idle-timeout duration
    Idle time before preloading code model (default: 30s)
```

### Idle Preloading

The wrapper supports optional idle-based preloading to minimize model switch delays:

```bash
# Enable idle preload with 30-second timeout
./tei-wrapper --idle-preload --idle-timeout 30s
```

**How it works:**
1. After each request, the wrapper starts an idle timer
2. If no requests arrive within the idle timeout period, it automatically switches to the preferred model (code model)
3. This ensures the code model is ready for the next indexing run
4. Reduces subsequent startup time to near-zero

**When to use:**
- ✅ Development workflows with periodic indexing
- ✅ When you index code more frequently than docs
- ❌ Production environments with constant traffic (no idle time)
- ❌ When you need both models equally often

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

Health check endpoint with current model information.

**Response (Healthy):**
```json
{
  "status": "ok",
  "model": "nomic-ai/nomic-embed-code"
}
```

**Response (Model Switching):**
```json
{
  "status": "switching",
  "model": "nomic-ai/nomic-embed-code",
  "switching": true
}
```

**Response (Unhealthy):**
```json
{
  "status": "unhealthy",
  "model": "nomic-ai/nomic-embed-code",
  "error": "TEI is not responding"
}
```

## Supported Models

See [TEI_SETUP.md](../../docs/guides/TEI_SETUP.md#model-selection) for supported models and recommendations.

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

**Implemented (Slices 1-3):**
- ✅ OpenAI-compatible /v1/embeddings endpoint
- ✅ Basic TEI process management
- ✅ Health check endpoint with model status
- ✅ Request forwarding and response translation
- ✅ Model hot-swapping (auto-restart TEI when model changes)
- ✅ 503 Service Unavailable response during model switches
- ✅ Background pre-loading of preferred model on idle (optional)
- ✅ Idle detection with configurable timeout

**Future Enhancements (Slice 4):**
- ⏳ Configuration file support (YAML/TOML)
- ⏳ Request queuing during model switches (currently returns 503)
- ⏳ Enhanced error handling and retry logic
- ⏳ Metrics and monitoring endpoints

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
