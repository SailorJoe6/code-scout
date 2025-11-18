# Code Scout - Planning Documentation

This directory contains the planning, design, and architectural documentation for Code Scout.

## 📚 Documentation Index

### Core Design Documents

- **[Overview](overview.md)** - Complete system architecture, key insights, and implementation plan
  - Dual-pass embedding strategy
  - System architecture diagrams
  - Component breakdown
  - Elephant Carpaccio implementation slices
  - MVP scope and success criteria

### Implementation Planning

- **[Language Detection](language-detection.md)** - Multi-language support strategy
  - File extension mapping for 11+ languages
  - C/C++ disambiguation heuristics
  - Detection algorithm and implementation
  - Edge cases and future enhancements

### Implementation Status

See [overview.md](overview.md) for the complete **Elephant Carpaccio** slice plan.

#### ✅ Completed Slices

- **Slice 1**: Simplest Possible End-to-End (MVP)
  - CLI: `code-scout index` and `code-scout search`
  - Go language support only
  - Naive chunking (split by blank lines)
  - JSON output with file paths and line numbers

- **Slice 2**: Semantic Code Chunking
  - Tree-sitter integration for Go
  - Function/class-level chunking
  - Metadata extraction (imports, context)

- **Slice 3a**: Documentation Indexing
  - Markdown/text file support (`.md`, `.txt`, `.rst`)
  - Header-based chunking (H1/H2/H3)
  - Unified code + docs search

- **Slice 3b**: Dual-Model Architecture
  - Second embedding model (code-scout-text)
  - Two-pass embedding pipeline
  - Search modes: `--code`, `--docs`, `--hybrid`
  - Complete dual-model system

- **Slice 5**: Configuration & Provider Flexibility (Partial)
  - Config file support (`.code-scout.json`, `~/.code-scout/config.json`)
  - Configurable endpoints and API keys
  - OpenAI-compatible API support
  - Multiple embedding providers

#### 🚧 In Progress

- **Slice 4**: Multi-Language Support
  - **Status**: Foundation complete (language detection)
  - **Next**: Parser factories, Tree-sitter queries
  - **Target Languages**: Python, JS/TS, Java, Rust, C, C++, Ruby, PHP, Scala
  - **Tracking**: `code_scout-72w` (beads epic)

#### 📋 Planned Slices

- **Slice 6**: Production Hardening
  - Incremental updates (done)
  - Progress indicators
  - Better error messages
  - Relevance score thresholds
  - File pattern filtering

- **Slice 7**: Extended Language Support
  - Additional languages as needed
  - Language-specific optimizations

## 🎯 Current Focus

**Slice 4: Multi-Language Support**

We're implementing semantic parsing for 9 additional languages beyond Go:
1. Python (`.py`)
2. JavaScript/TypeScript (`.js`, `.ts`, `.tsx`, `.jsx`)
3. Java (`.java`)
4. Rust (`.rs`)
5. C (`.c`, some `.h`)
6. C++ (`.cpp`, `.cc`, `.cxx`, `.hpp`, `.hxx`, most `.h`)
7. Ruby (`.rb`)
8. PHP (`.php`)
9. Scala (`.scala`)

**Progress**:
- ✅ Language detection with C/C++ heuristics
- ✅ Tree-sitter dependencies installed
- ✅ Test suite (23 tests passing)
- ⏳ Parser factories (next)
- ⏳ Tree-sitter query files (next)
- ⏳ Integration with semantic chunker (next)

## 📖 How to Use This Documentation

1. **New to Code Scout?** Start with [overview.md](overview.md) to understand the architecture
2. **Working on language support?** See [language-detection.md](language-detection.md)
3. **Implementing a new feature?** Check [overview.md](overview.md) for the vertical slice plan
4. **Need implementation details?** Review the specific planning docs for that area

## 🔗 Related Documentation

- **Root README**: `/README.md` - User-facing documentation and quickstart
- **Developer Guide**: `/DEVELOPERS.md` - Build requirements and setup
- **Agent Guide**: `/AGENTS.md` - AI agent workflow and beads usage

## 📝 Document Conventions

All planning documents in this directory are:
- **Ephemeral**: May be updated or replaced as the project evolves
- **Informative**: Provide context for current and future development
- **Version controlled**: Committed to preserve decision history
- **Separated from code**: Keep the repository root clean

These docs are primarily for:
- AI coding agents working on the project
- Human developers reviewing architectural decisions
- Future contributors understanding the design rationale
