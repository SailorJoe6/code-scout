# Code Scout Guides

This directory contains setup and usage guides for Code Scout users and developers.

## Setup Guides

### Embedding Providers

- **[TEI_SETUP.md](TEI_SETUP.md)** - Installing TEI (Text Embeddings Inference) for local embedding generation
  - Platform-specific instructions (macOS, Linux, Windows)
  - GPU acceleration setup (Metal, CUDA)
  - CPU-only deployment options

- **[TEI_WRAPPER.md](TEI_WRAPPER.md)** - Using the TEI wrapper for automatic model switching
  - OpenAI-compatible API access to TEI
  - Automatic model hot-swapping (code vs documentation)
  - Idle preloading optimization
  - Dual TEI instance management (embeddings + reranker)

### Development Environment

- **[DEV_CONTAINER.md](DEV_CONTAINER.md)** - Docker-based development container setup
- **[LINUX_ARM64_BUILD.md](LINUX_ARM64_BUILD.md)** - Building LanceDB native libraries on Linux ARM64

### Advanced Features

- **[RERANKER_SETUP.md](RERANKER_SETUP.md)** - Cross-encoder reranking for improved search relevance
  - What is reranking and when to use it
  - Model selection guide (BAAI, Jina, cross-encoder models)
  - Platform-specific deployment
  - Performance tuning and optimization
  - Troubleshooting
- **[TESTED_MODELS.md](TESTED_MODELS.md)** - Verified reranker model compatibility
  - TEI versions and validation status
  - Notes on tested models and gaps

- **[BACKGROUND_DAEMON.md](BACKGROUND_DAEMON.md)** - Automatic background indexing
  - File watcher setup
  - Auto-indexing on code changes
  - Process management

## Quick Start Recommendations

**New users:**
1. Start with [TEI_SETUP.md](TEI_SETUP.md) - Install TEI for your platform
2. Use [TEI_WRAPPER.md](TEI_WRAPPER.md) - Set up automatic model switching
3. Optionally add [RERANKER_SETUP.md](RERANKER_SETUP.md) - Improve search quality

**Production deployment:**
1. Use TEI with wrapper ([TEI_WRAPPER.md](TEI_WRAPPER.md))
2. Enable reranking ([RERANKER_SETUP.md](RERANKER_SETUP.md))
3. Configure background daemon ([BACKGROUND_DAEMON.md](BACKGROUND_DAEMON.md))

## Architecture Overview

Code Scout uses a two-stage semantic search pipeline:

1. **Embedding Generation** (TEI)
   - Code model: `nomic-ai/CodeRankEmbed`
   - Documentation model: `nomic-ai/nomic-embed-text-v1.5`

2. **Vector Search** (LanceDB)
   - Stores embeddings in local database
   - Fast similarity search

3. **Optional: Reranking** (Cross-Encoder)
   - Re-scores top-K results for better relevance
   - Adds 100-500ms latency but significantly improves accuracy

See the individual guides for detailed setup instructions.
