# Code Scout

Code Scout is a specialized vector database solution designed to give AI coding agents full awareness of any codebase. By intelligently vectorizing and storing code alongside its documentation, Code Scout enables AI assistants like Claude and Codex to understand and work with your entire project context.

## Overview

Code Scout uses open source embedding models to create semantic representations of your codebase, storing them in a local vector database. Unlike traditional code search tools, Code Scout is specifically adapted to maintain the relationship between code and its markdown documentation, providing AI agents with comprehensive context for better assistance.

## Key Features

- **Open Source Embeddings**: Leverages open source embedding models for code vectorization
- **Code + Documentation**: Specially designed to store code alongside markdown documentation
- **Cross-Encoder Reranking**: Optional reranking for significantly improved search relevance
- **Local Vector Database**: All data stays local - no cloud dependencies
- **AI Agent Optimized**: Built specifically for AI coding assistants like Claude and Codex
- **Full Codebase Awareness**: Enables AI agents to understand the complete context of your project
- **Multi-Language Support**: Semantic chunking for 11 programming languages
- **Smart Filtering**: Respects `.gitignore` and `.code-scout-ignore` to exclude unwanted files
- **Background Indexing**: Automatic re-indexing on file changes

## Language Support

Code Scout provides semantic code chunking using Tree-sitter parsers. Each language extracts meaningful code units (functions, classes, methods, structs, etc.) for precise embedding and retrieval.

### Supported Languages

| Language | File Extensions | Semantic Chunking | Status |
|----------|----------------|-------------------|--------|
| **Go** | `.go` | Functions, methods, structs, interfaces, constants, variables | ✅ Fully Supported |
| **Python** | `.py` | Functions, classes, methods, async functions, decorators | ✅ Fully Supported |
| **JavaScript** | `.js`, `.jsx`, `.mjs`, `.cjs` | Functions, classes, methods, arrow functions, generators | ✅ Fully Supported |
| **TypeScript** | `.ts`, `.tsx` | Functions, classes, methods, arrow functions, generators | ✅ Fully Supported |
| **Java** | `.java` | Classes, interfaces, methods, constructors, enums, records | ✅ Fully Supported |
| **Rust** | `.rs` | Functions, structs, enums, traits, impls, modules | ✅ Fully Supported |
| **C** | `.c`, `.h` | Functions, structs, unions, enums, typedefs | ✅ Fully Supported |
| **C++** | `.cpp`, `.cc`, `.cxx`, `.hpp`, `.hxx`, `.h` | Functions, classes, namespaces, templates, methods | ✅ Fully Supported |
| **Ruby** | `.rb` | Methods, classes, modules, singleton methods | ✅ Fully Supported |
| **PHP** | `.php` | Functions, classes, methods, traits, interfaces, enums | ✅ Fully Supported |
| **Scala** | `.scala` | Functions, classes, objects, traits, case classes | ✅ Fully Supported |

### Semantic Chunking Benefits

Traditional code search tools split files by line count or character limits, often breaking functions and classes mid-definition. Code Scout's semantic chunking:

- **Preserves Code Boundaries**: Functions, classes, and methods are kept intact
- **Captures Context**: Includes docstrings, comments, and signatures
- **Enables Precise Search**: Find specific functions or classes, not arbitrary text snippets
- **Improves Embeddings**: Complete code units produce more meaningful semantic vectors

### Language Detection

Code Scout automatically detects the language of each file based on file extension. For files with ambiguous extensions (e.g., `.h` files could be C or C++), it uses heuristic analysis to determine the correct language.

## Ignore File Support

Code Scout respects `.gitignore` and `.code-scout-ignore` files to filter which files get indexed. This prevents indexing of build artifacts, dependencies, and other files you don't want in your semantic search.

### How It Works

- **`.gitignore`**: Automatically respected, just like git
- **`.code-scout-ignore`**: Project-specific patterns for code-scout only
- **Patterns combine**: Both files are read and patterns are merged
- **Gitignore syntax**: Standard gitignore pattern matching (wildcards, negation, etc.)
- **Always ignored**: `.code-scout/` directory (hardcoded)

### Example `.code-scout-ignore`

```
# Exclude examples and test data
examples/
test_data/

# Exclude generated files
*.gen.go
*.pb.go

# Exclude vendor dependencies
vendor/
node_modules/
```

### Pattern Syntax

Supports standard gitignore patterns:
- `*.log` - Wildcard matching
- `build/` - Directory matching (trailing slash)
- `temp.py` - Exact filename matching
- `# comment` - Comments (lines starting with #)
- Blank lines are ignored

Both files are optional. If neither exists, only hidden files/directories (those starting with `.`) are automatically skipped.

## Embedding Models

Code Scout uses custom-configured Ollama models with persistent context window settings to ensure reliable code embedding without silent truncation.

### Why Custom Model Files?

**Problem**: Ollama's default context window (2048 tokens) is too small for code files and silently discards content beyond this limit. Runtime API parameters must be sent with every request or the server reverts to defaults.

**Solution**: We use custom Modelfiles to persistently configure larger context windows that match each model's native capacity. This ensures:
- No silent truncation of code files
- Consistent behavior across all API calls
- Optimal context usage for embedding generation
- Simpler client code (no per-request parameter management)

### Supported Models

Code Scout uses two embedding models optimized for different purposes. Default model IDs for TEI (wrapper or any HuggingFace-backed endpoint):

**nomic-ai/CodeRankEmbed** (8K context)
- Code-optimized embeddings
- Context window: 8,192 tokens (~500-600 lines of code)
- Best for: Code files, larger modules

**nomic-ai/nomic-embed-text-v1.5** (2K context)
- General-purpose text embeddings
- Context window: 2,048 tokens
- Best for: Documentation, comments, shorter text files

Note: `nomic-embed-code` is a large Ollama model with higher accuracy but requires a powerful GPU and lots of RAM, and it is not currently supported by TEI. For TEI or lower-power machines, `nomic-ai/CodeRankEmbed` is the preferred code model.

If you're using Ollama, the custom Modelfiles below create `code-scout-code` and `code-scout-text` with larger context windows.

### Setting Up Custom Models (Ollama)

The `ollama-models/` directory contains Modelfiles with pre-configured context windows.

**1. Install Ollama** (if not already installed):
```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.com/install.sh | sh

# Start the Ollama service
brew services start ollama
```

**2. Pull the base models**:
```bash
# Pull nomic-embed-text
ollama pull nomic-embed-text

# Pull nomic-embed-code (large; requires a powerful GPU and lots of RAM)
ollama pull manutic/nomic-embed-code
```

**3. Create custom models from Modelfiles**:
```bash
# Navigate to repo root
cd /path/to/code_scout

# Create custom nomic-embed-text with 8K context
ollama create code-scout-text -f ollama-models/nomic-embed-text.Modelfile

# Create custom nomic-embed-code with 32K context
ollama create code-scout-code -f ollama-models/nomic-embed-code.Modelfile
```

**4. Verify the models**:
```bash
# List your models
ollama list

# Test the custom models
ollama run code-scout-text
ollama run code-scout-code
```

**5. Use in Code Scout**:
```python
import ollama

# Use custom models with persistent context settings
response = ollama.embeddings(
    model='code-scout-code',  # or 'code-scout-text'
    prompt='your code here'
)
# Context window is automatically set to 32K (no need for options parameter)
```

### Model Selection Guidelines

| File Type | Recommended Model | Reason |
|-----------|------------------|---------|
| `.md`, `.txt`, `.rst` | `nomic-ai/nomic-embed-text-v1.5` | General text content |
| `.py`, `.js`, `.java`, `.go`, `.rb`, `.php` | `nomic-ai/CodeRankEmbed` | Code-optimized embeddings |
| Mixed code+docs | `nomic-ai/nomic-embed-text-v1.5` | Balanced for both |
| Large files (>500 lines) | `nomic-ai/CodeRankEmbed` | 8K context handles larger files |

Ollama users: replace with `code-scout-text` and `code-scout-code` if you created the custom models.

## Use Cases

- Provide AI coding agents with instant access to your entire codebase
- Enable intelligent code navigation and search
- Help AI assistants understand project architecture and patterns
- Maintain context between code implementation and documentation
- Support code review and refactoring with full project awareness

## Project Status

This is a greenfield project currently in early development.

## Documentation

For AI agents working on this project:
- `AGENTS.md` - Contains workflow instructions for AI agents, including issue tracking with bd (beads)
- `CLAUDE.md` - Symlink to `AGENTS.md` for Claude-specific references

**Note on Symlink**: `CLAUDE.md` is a symbolic link pointing to `AGENTS.md`. This works natively on Unix-like systems (Linux, macOS). On Windows, Developer Mode may need to be enabled for proper symlink support, otherwise the file may appear as a text file containing the target path.

## Configuration

Code Scout can be configured to use custom embedding API endpoints, making it compatible with OpenAI-compatible services like OpenRouter, remote GPU hosts, or any other compatible API.

### Configuration Files

Configuration can be specified in two ways:

1. **User-level**: `~/.code-scout/config.json` - Global defaults for all projects
2. **Project-level**: `.code-scout.json` - Project-specific settings (overrides user-level)

**Config Discovery**: Both `code-scout` and `tei-wrapper` automatically search for `.code-scout.json` by walking up the directory tree from your current working directory (similar to how git finds `.git/`). This means you can run commands from any subdirectory within your project, and they'll find the project's configuration file. If no project config is found, the user-level config (`~/.code-scout/config.json`) is used as a fallback.

### Configuration Format

Create a JSON file with the following structure:

```json
{
  "endpoint": "http://localhost:11434",
  "api_key": "",
  "code_model": "nomic-ai/CodeRankEmbed",
  "text_model": "nomic-ai/nomic-embed-text-v1.5",
  "rerank_model": "",
  "rerank_top_k": 0,
  "single_model_mode": true
}
```

**Fields:**
- `endpoint`: The base URL of the tei-wrapper or OpenAI-compatible embedding API (no trailing slash)
- `api_key`: (Optional) API key for authentication. Sent as `Authorization: Bearer <api_key>` header
- `code_model`: Model name to use for code embeddings
- `text_model`: Model name to use for documentation embeddings
- `rerank_model`: (Optional) Cross-encoder model name for reranking (e.g., `BAAI/bge-reranker-base`). Requires tei-wrapper with `/rerank` endpoint (see [RERANKER_SETUP.md](docs/guides/RERANKER_SETUP.md))
- `rerank_top_k`: (Optional) Number of top results to rerank (defaults to search `--limit` when `rerank_model` is set)
- `single_model_mode`: (Optional, default: `true`) Use single TEI process with model switching. Set to `false` for multi-model mode (runs separate TEI processes for each model simultaneously, higher memory but faster)

Defaults target the TEI wrapper (CodeRankEmbed + nomic-embed-text-v1.5). If you use Ollama, set `code_model` and `text_model` to your local model names (for example, `code-scout-code` / `code-scout-text`).

### Example Configurations

**Default (TEI Wrapper Local - Single Model Mode)**:
```json
{
  "endpoint": "http://localhost:11434",
  "code_model": "nomic-ai/CodeRankEmbed",
  "text_model": "nomic-ai/nomic-embed-text-v1.5",
  "rerank_model": "",
  "rerank_top_k": 0,
  "single_model_mode": true
}
```

**Ollama Local (Custom Models)**:
```json
{
  "endpoint": "http://localhost:11434",
  "code_model": "code-scout-code",
  "text_model": "code-scout-text",
  "rerank_model": "",
  "rerank_top_k": 0
}
```

**Cloud Provider (OpenAI-compatible)**:
```json
{
  "endpoint": "https://api.provider.com/v1",
  "api_key": "your-api-key",
  "code_model": "provider-code-model",
  "text_model": "provider-text-model",
  "rerank_model": "provider-rerank-model",
  "rerank_top_k": 50
}
```

**Remote TEI Wrapper (GPU Server)**:
```json
{
  "endpoint": "http://my-gpu-server:11434",
  "code_model": "nomic-ai/nomic-embed-code",
  "text_model": "nomic-ai/nomic-embed-text-v1.5",
  "rerank_model": "",
  "rerank_top_k": 0
}
```
Note: `nomic-ai/nomic-embed-code` requires a powerful GPU and lots of RAM and is Ollama-only today. If your remote server runs TEI, use `nomic-ai/CodeRankEmbed` instead.

**With Cross-Encoder Reranking (Recommended)**:
```json
{
  "endpoint": "http://localhost:11434",
  "code_model": "nomic-ai/CodeRankEmbed",
  "text_model": "nomic-ai/nomic-embed-text-v1.5",
  "rerank_model": "BAAI/bge-reranker-base",
  "rerank_top_k": 25
}
```

This configuration enables reranking for significantly improved search relevance. The tei-wrapper automatically manages both embedding and reranker TEI instances.

**Start tei-wrapper with reranking:**
```bash
cd cmd/tei-wrapper
go build -o tei-wrapper .
./tei-wrapper \
  --model nomic-ai/nomic-embed-text-v1.5 \
  --rerank-model BAAI/bge-reranker-base
```

**Search with reranking:**
```bash
code-scout search "authentication logic" --limit 10

# Output shows both vector and rerank scores:
# Found 10 unique hybrid results (reranked by BAAI/bge-reranker-base) for: authentication logic
# 1. internal/auth/handler.go:23-45 (vector: 0.234, rerank: 0.95)
#    Language: go | Source: code | Chunk: function
#    ...
```

See [RERANKER_SETUP.md](docs/guides/RERANKER_SETUP.md) for complete reranking setup, model selection, and performance tuning.

### CLI Flag Override

You can override the endpoint for a single command using the `--endpoint` flag:

```bash
# Use a different endpoint for this indexing operation
code-scout index --endpoint http://remote-server:11434

# Use a different endpoint for searching
code-scout search "authentication" --endpoint https://api.example.com
```

### Setup Example

```bash
# Create user-level config directory
mkdir -p ~/.code-scout

# Create default configuration (local TEI wrapper, no API key needed)
cat > ~/.code-scout/config.json << 'EOF'
{
  "endpoint": "http://localhost:11434",
  "code_model": "nomic-ai/CodeRankEmbed",
  "text_model": "nomic-ai/nomic-embed-text-v1.5",
  "rerank_model": "nomic-ai/nomic-embed-text-v1.5",
  "rerank_top_k": 25
}
EOF

# Or create project-specific config with API key for cloud provider
cat > .code-scout.json << 'EOF'
{
  "endpoint": "https://api.provider.com/v1",
  "api_key": "your-api-key",
  "code_model": "provider-code-model",
  "text_model": "provider-text-model",
  "rerank_model": "provider-rerank-model",
  "rerank_top_k": 50
}
EOF
```

### Cloud Provider Setup

Code Scout works with any OpenAI-compatible embedding API endpoint. This includes services like OpenRouter, OpenAI, and other providers that expose a `/v1/embeddings` endpoint.

**Important:** Most cloud providers do NOT host the nomic-ai embedding models used in the self-hosted setup. You'll need to use the provider's available embedding models instead.

1. **Choose a provider** that offers OpenAI-compatible embedding APIs
2. **Create an account** and generate an API key
3. **Check available models** - Ensure they offer code and text embedding models
4. **Copy the sample config**: `cp .code-scout.json.example .code-scout.json`
5. **Edit `.code-scout.json`** with your provider's endpoint, API key, and model names
6. **Run indexing**:
   ```bash
   ./code-scout index --workers 6 --batch-size 6
   ```

**Note:** Cloud hosting typically incurs costs based on usage. For free, self-hosted options see below.

Git already ignores `.code-scout.json`, so your API keys stay local and never get committed.

## Self-Hosting Embedding Models (Recommended)

For the best performance and cost (free!), self-host your embedding models using TEI (Text Embeddings Inference).

### Recommended: TEI with Wrapper

**Best for:** Most users, especially with AI coding agents

TEI provides the fastest embedding generation with full GPU acceleration on all platforms:

| Platform | Acceleration | Speed | Installation |
|----------|-------------|-------|--------------|
| **Mac (Apple Silicon)** | Metal | Very Fast | `brew install text-embeddings-inference` |
| **Linux/Windows + NVIDIA** | CUDA | Blazing Fast | Docker with `--gpus all` |
| **CPU-only** | None | Moderate | Docker CPU image |

**Quick start:**

```bash
# 1. Install TEI (see TEI_SETUP.md for platform-specific instructions)
# macOS:
brew install text-embeddings-inference

# 2. Build and start TEI wrapper (handles model switching)
cd cmd/tei-wrapper
go build -o tei-wrapper .
./tei-wrapper --idle-preload &

# 3. Start background daemon (auto-indexes on file changes)
code-scout daemon start

# 4. Just search! (indexing happens automatically)
code-scout search "authentication"
```

**Features:**
- ✅ **3-4x faster** than Ollama
- ✅ **Automatic model switching** (code vs documentation)
- ✅ **Optional cross-encoder reranking** for improved search relevance
- ✅ **Background indexing** (no manual `code-scout index`)
- ✅ **GPU acceleration** (Metal on Mac, CUDA on Linux/Windows)
- ✅ **Lower memory** (4-8GB base, +500MB-2GB with reranking)

**Documentation:**
- [TEI Setup Guide](docs/guides/TEI_SETUP.md) - Install TEI for your platform
- [TEI Wrapper Guide](docs/guides/TEI_WRAPPER.md) - Setup and usage
- [Reranker Setup Guide](docs/guides/RERANKER_SETUP.md) - Cross-encoder reranking for improved relevance
- [Background Daemon Guide](docs/guides/BACKGROUND_DAEMON.md) - Auto-indexing

### Alternative: Ollama

**Best for:** Simplest setup, small repos, any platform

Ollama is the easiest option but requires reduced concurrency (`--workers 2 --batch-size 2`):

```bash
# Install Ollama
brew install ollama  # or see https://ollama.com/download

# Pull model
ollama pull nomic-embed-text

# Use with code-scout (reduced concurrency required)
code-scout index --workers 2 --batch-size 2
```

**Trade-offs:**
- ✅ Dead simple setup
- ✅ Works on all platforms
- ⚠️ 2-3x slower than TEI
- ⚠️ Limited concurrency (max 2 workers)

**Documentation:**
- [Ollama Setup Guide](docs/guides/OLLAMA_SETUP.md) - Installation and configuration

### Comparison

| Feature | TEI + Wrapper | Ollama |
|---------|--------------|--------|
| **Speed** | Fast (3-4x faster) | Slow |
| **Concurrency** | High (6-10 workers) | Low (2 workers) |
| **Memory** | 4-8GB | 4-8GB |
| **Setup** | Moderate | Very Easy |
| **GPU** | ✅ Metal/CUDA | ✅ Metal/CUDA |
| **Auto Model Switching** | ✅ Yes | ✅ Yes |

## Background Indexing Daemon

Code Scout includes a background daemon that automatically re-indexes your codebase when files change, eliminating the need to manually run `code-scout index` after every code change.

### Features

- **Automatic Re-indexing**: Watches for file changes and triggers indexing automatically
- **Debouncing**: Waits 5 seconds after the last file change before indexing (prevents excessive re-indexing during active editing)
- **Respects Ignore Patterns**: Uses `.gitignore` and `.code-scout-ignore` patterns
- **Graceful Shutdown**: Handles SIGTERM and SIGINT signals cleanly
- **Process Management**: Simple start/stop/status/logs commands

### Usage

**Start the daemon:**
```bash
code-scout daemon start
```

**Check daemon status:**
```bash
code-scout daemon status
```

**View daemon logs:**
```bash
code-scout daemon logs
```

**Stop the daemon:**
```bash
code-scout daemon stop
```

### How It Works

1. The daemon watches all directories in your repository (respecting ignore patterns)
2. When file changes are detected (create, modify, delete, rename), a 5-second debounce timer starts
3. If more changes occur during the 5 seconds, the timer resets
4. Once file activity settles, the daemon automatically runs the indexing process
5. The daemon performs incremental indexing (only changed files are re-indexed)

### Files

- **PID file**: `.code-scout/daemon.pid` - Tracks the running daemon process
- **Log file**: `.code-scout/daemon.log` - Contains daemon activity logs

### Benefits for AI Agents

With the daemon running, AI coding agents can:
- Use `code-scout search` without worrying about stale results
- Skip manual `code-scout index` commands
- Get instant semantic search across the latest code

**Documentation:** See [Background Daemon Guide](docs/guides/BACKGROUND_DAEMON.md) for complete setup and usage instructions.

## Getting Started

*Coming soon - installation and usage instructions*

## Contributing

*Coming soon - contribution guidelines*

## License

*To be determined*
