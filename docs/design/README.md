# Code Scout Design Documentation

This directory contains design documentation for Code Scout, optimized for AI agents to understand the system architecture, implementation details, and extension points.

## Purpose

Code Scout is a semantic code search tool that helps AI agents understand codebases by:
1. Parsing code into meaningful semantic units (functions, methods, types)
2. Generating embeddings for semantic similarity search
3. Storing in a vector database for fast retrieval
4. Deduplicating results to reduce noise

## Quick Navigation

### Core Architecture
- [**architecture.md**](architecture.md) - High-level system architecture and design principles
- [**data-flow.md**](data-flow.md) - How data flows from source code to search results
- [**components.md**](components.md) - Component responsibilities and interfaces

### Implementation Details
- [**semantic-chunking.md**](semantic-chunking.md) - Tree-sitter based code parsing and chunking
- [**embedding-strategy.md**](embedding-strategy.md) - Embedding generation and deduplication
- [**vector-storage.md**](vector-storage.md) - LanceDB schema and query patterns

### User Interface
- [**cli-interface.md**](cli-interface.md) - CLI commands and workflows

### Extension
- [**extension-points.md**](extension-points.md) - How to add new languages and features

## Key Concepts

**Semantic Chunking**: Using tree-sitter to parse code and extract meaningful units (functions, methods, types) rather than splitting on arbitrary boundaries like blank lines.

**Content-Based Deduplication**:
- Index-time: Hash code content to skip generating duplicate embeddings
- Search-time: Collapse identical results to reduce noise

**Incremental Indexing**: Track file modification times and only re-index changed files.

**Vector Search**: Use embedding similarity (cosine distance) to find semantically related code.

## Document Conventions

Each document follows this structure:
1. **Overview** - What problem does this solve?
2. **Concepts** - Key abstractions and terminology
3. **Implementation** - How it's implemented in code
4. **Examples** - Concrete examples from the codebase
5. **Extension** - How to extend or modify this component

## Audience

These documents are written for:
- **AI agents** (primary) - Need to understand the codebase to assist with development
- **Human developers** (secondary) - Need to understand design decisions and implementation details

## Related Documentation

- [CLAUDE.md](../../CLAUDE.md) - AI agent guidelines for working on this project
- [DEVELOPERS.md](../../DEVELOPERS.md) - Build setup and development workflow
- [README.md](../../README.md) - User-facing project overview
