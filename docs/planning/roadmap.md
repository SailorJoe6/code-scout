# Code Scout Implementation Roadmap

This document tracks the Elephant Carpaccio implementation slices for Code Scout. Each slice delivers a **working, shippable system** with incrementally increasing capability.

## Slice Overview

| Slice | Description | Status |
|-------|-------------|--------|
| 1 | Simplest End-to-End | ✅ Complete |
| 2 | Semantic Code Chunking | ✅ Complete |
| 3a | Documentation Indexing | ✅ Complete |
| 3b | Dual-Model Architecture | ✅ Complete |
| 4 | Multi-Language Support | ✅ Complete |
| 5 | Configuration & Providers | ✅ Complete |
| 6 | Production Hardening | 🚧 In Progress |
| 7 | Extended Language Support | 📋 Planned |

---

## Completed Slices

### Slice 1: Simplest Possible End-to-End

**Deliverable**: Index a single Go file and search it.

**Features:**
- CLI: `code-scout index` and `code-scout search "query" --json`
- Hardcoded config (local Ollama, code-scout-code model)
- Single language: Go only
- Naive chunking: Split by blank lines (no Tree-sitter yet)
- LanceDB: Store chunks with embeddings
- Search: Code-only mode (no docs yet)
- JSON output with file path and line numbers

---

### Slice 2: Semantic Code Chunking

**Deliverable**: Proper function/class-level chunking with Tree-sitter.

**Features:**
- Tree-sitter integration for Go
- Extract functions, classes, methods as chunks
- Include docstrings and context metadata
- Better relevance in search results

---

### Slice 3a: Documentation Indexing

**Deliverable**: Add markdown documentation indexing.

**Features:**
- Scan for `.md`, `.txt`, `.rst` files
- Markdown header-based chunking (H1/H2/H3)
- Index docs using existing code-scout-code model
- Basic search includes both code and docs

---

### Slice 3b: Dual-Model Architecture

**Deliverable**: Add second embedding model for documentation.

**Features:**
- Second embedding model (code-scout-text) for documentation
- Two-pass embedding pipeline (code model for code, text model for docs)
- Search modes: `--code`, `--docs`, `--hybrid` (default)
- Dual-query retrieval logic

---

### Slice 4: Multi-Language Support

**Deliverable**: Support 11 programming languages.

**Features:**
- Tree-sitter grammars for Go, Python, JS, TS, Java, Rust, C, C++, Ruby, PHP, Scala
- Language detection by file extension with C/C++ heuristics
- Language-specific query files
- Tested on multi-language repos

---

### Slice 5: Configuration & Provider Flexibility

**Deliverable**: Config file support and multiple embedding providers.

**Features:**
- `.code-scout.json` and `~/.code-scout/config.json` for settings
- Support OpenAI-compatible APIs (not just Ollama)
- API key management
- Model selection per provider
- Reranking support

---

## In Progress

### Slice 6: Production Hardening

**Deliverable**: Robust error handling, better UX, performance.

**Features:**
- ✅ Incremental updates (re-index only changed files)
- ⏳ Progress indicators for long-running operations
- ⏳ Better error messages and logging
- ⏳ `--limit`, `--files` flags for filtering
- ⏳ Relevance score thresholds

---

## Planned

### Slice 7: Extended Language Support

**Deliverable**: Add remaining languages (Kotlin, C#, Swift, etc.).

**Features:**
- Tree-sitter queries for additional languages
- Language-specific metadata extraction
- Test suite for each language

---

## Future Enhancements (Post-MVP)

- Batch query API
- Query result caching
- MCP server integration
- Parallel embedding for large repos
- Web UI dashboard (optional, low priority)

---

## MVP Definition

The Minimum Viable Product consisted of **Slices 1-3b**, delivering:
- ✅ `code-scout index` - Full repository indexing
- ✅ `code-scout search "query" --code/--docs/--hybrid` - All three search modes
- ✅ JSON output for AI agent consumption
- ✅ Go language support
- ✅ Markdown documentation support
- ✅ Tree-sitter based semantic chunking
- ✅ LanceDB vector storage
- ✅ Dual embedding models (code + text)
