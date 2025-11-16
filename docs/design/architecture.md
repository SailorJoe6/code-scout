# Code Scout Architecture

## Overview

Code Scout is a semantic code search tool built on four core principles:
1. **Semantic understanding** over text matching
2. **Local-first** - all data stays on the developer's machine
3. **AI-optimized** - designed for AI agent consumption
4. **Incremental** - fast updates for changed files

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI Layer                            │
│  (cmd/code-scout/)                                          │
│  - index command: scan → chunk → embed → store             │
│  - search command: embed query → search → deduplicate       │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
┌───────▼────────┐   ┌────────▼────────┐   ┌───────▼────────┐
│    Scanner     │   │    Chunker      │   │   Embeddings   │
│  (internal/    │   │  (internal/     │   │  (internal/    │
│   scanner)     │   │   chunker)      │   │   embeddings)  │
│                │   │                 │   │                │
│ Find code      │   │ Parse with      │   │ Generate with  │
│ files          │   │ tree-sitter     │   │ Ollama         │
│                │   │ Extract chunks  │   │ Deduplicate    │
└────────────────┘   └─────────────────┘   └────────────────┘
                              │
                     ┌────────▼────────┐
                     │     Parser      │
                     │  (internal/     │
                     │   parser)       │
                     │                 │
                     │ Tree-sitter     │
                     │ Go/Python       │
                     │ Extract types   │
                     └─────────────────┘
                              │
        ┌─────────────────────┴─────────────────────┐
        │                                           │
┌───────▼────────┐                         ┌────────▼────────┐
│    Storage     │                         │    Metadata     │
│  (internal/    │                         │  (.code-scout/  │
│   storage)     │                         │   metadata.json)│
│                │                         │                 │
│ LanceDB        │                         │ File mod times  │
│ Vector search  │                         │ Last index time │
└────────────────┘                         └─────────────────┘
```

## Design Principles

### 1. Separation of Concerns

**Scanner** - File discovery
- Finds code files in directory tree
- Respects .gitignore and .code-scout-ignore
- Returns file paths with language detection

**Parser** - Semantic understanding
- Uses tree-sitter for language parsing
- Extracts functions, methods, types, interfaces
- Captures context metadata (imports, package, etc.)

**Chunker** - Code segmentation
- Delegates to semantic chunker for supported languages (Go, Python)
- Falls back to blank-line chunking for unsupported languages
- Each chunk has: code, file path, line range, type, metadata

**Embeddings** - Semantic representation
- Calls Ollama API to generate vector embeddings
- Content-based deduplication (hash code → skip if seen)
- Worker pool for concurrent generation

**Storage** - Persistence
- LanceDB for vector similarity search
- Arrow format for efficient storage
- Metadata JSON for incremental updates

### 2. Incremental Updates

Code Scout tracks file modification times to avoid re-indexing unchanged files:

1. Load metadata.json with file→modtime map
2. Compare current modtimes with stored modtimes
3. Only index new/modified files
4. Delete old chunks for modified files
5. Update metadata after indexing

This makes re-indexing fast (seconds instead of minutes).

### 3. Content Deduplication

**Index-time** (embedding generation):
- Compute SHA256 hash of code content
- Track first occurrence of each hash
- Only generate embedding for unique hashes
- Copy embedding to duplicate chunks

**Search-time** (result filtering):
- Group results by identical code content
- Keep highest-scoring result per group
- Return deduplicated results

Benefits:
- Reduces Ollama API calls by ~11%
- Reduces search noise by 30-80%
- Saves tokens for AI agents

### 4. Semantic Chunking

Traditional approach: Split code on blank lines
- Problem: Breaks functions across chunks, loses context

Code Scout approach: Tree-sitter semantic extraction
- Extract complete functions/methods/types
- Include context (imports, package, receiver type)
- Preserve code structure and meaning

Example from internal/parser/extractor.go:155-269
```go
func (e *Extractor) extractFunction(node *sitter.Node, source []byte) parser.Chunk
```

This function extracts a complete Go function including:
- Function signature
- Full body
- Line numbers
- Name, parameters, return type metadata

## Data Model

### Chunk Structure
```go
type Chunk struct {
    ID        string            // UUID for this chunk
    FilePath  string            // Path to source file
    LineStart int               // Starting line number
    LineEnd   int               // Ending line number
    Language  string            // Programming language (go, python, etc)
    Code      string            // The actual code content
    ChunkType string            // function, method, struct, interface, etc
    Name      string            // Name of function/type
    Metadata  map[string]string // Imports, package, receiver, etc
}
```

### LanceDB Schema
```
Table: code_chunks
Columns:
- chunk_id: String (UUID)
- file_path: String
- line_start: Int32
- line_end: Int32
- language: String
- code: LargeString
- vector: FixedSizeList<Float32>[3584]  // Embedding dimension
```

Metadata stored separately in `.code-scout/metadata.json`:
```json
{
  "last_index_time": "2025-11-15T...",
  "file_mod_times": {
    "/path/to/file.go": "2025-11-15T..."
  }
}
```

## Query Flow

### Index Command

1. **Scan**: Find all code files
2. **Filter**: Determine which files need indexing (new/modified)
3. **Delete**: Remove old chunks for modified files
4. **Chunk**: Parse with tree-sitter, extract semantic units
5. **Hash**: Compute content hash for deduplication
6. **Embed**: Generate embeddings (only for unique hashes)
7. **Store**: Save to LanceDB
8. **Update**: Save metadata

### Search Command

1. **Embed Query**: Generate embedding for search query
2. **Vector Search**: Find similar chunks in LanceDB
3. **Format**: Convert to SearchResult structs
4. **Deduplicate**: Collapse identical code content
5. **Sort**: Order by relevance score
6. **Return**: Output as JSON or human-readable

## Technology Stack

- **Language**: Go 1.21+
- **Parsing**: tree-sitter (via go-tree-sitter)
- **Embeddings**: Ollama API (nomic-embed-code model)
- **Vector DB**: LanceDB (with Arrow/Parquet)
- **CLI**: Cobra framework

## Performance Characteristics

**Indexing**:
- ~1000 chunks/minute with 10 concurrent workers
- Content deduplication: 11% reduction in API calls
- Incremental updates: Only re-index changed files

**Search**:
- Sub-second query response
- Deduplication: 30-80% reduction in duplicate results
- Default limit: 10 results

## Extension Points

See [extension-points.md](extension-points.md) for details on:
- Adding new languages
- Custom chunk types
- Alternative embedding models
- Different vector databases
