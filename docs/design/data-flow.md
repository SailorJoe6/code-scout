# Code Scout Data Flow

## Overview

This document traces how data flows through Code Scout from source code to search results, highlighting transformations at each stage.

## Indexing Flow

### 1. File Discovery
**Component**: Scanner (`internal/scanner/scanner.go`)

```
Input: Root directory path
       ↓
Process: Walk directory tree
         Filter by .gitignore
         Filter by .code-scout-ignore
         Detect language from extension
       ↓
Output: []FileInfo{Path, Language, ModTime}
```

Example output:
```go
FileInfo{
    Path:     "/path/to/cmd/code-scout/index.go",
    Language: "go",
    ModTime:  time.Parse("2025-11-15T..."),
}
```

**Code**: `scanner.ScanCodeFiles()` at internal/scanner/scanner.go:45-123

### 2. Incremental Filtering
**Component**: Index Command (`cmd/code-scout/index.go:54-83`)

```
Input: []FileInfo from scanner
       Metadata{FileModTimes}  // From previous index
       ↓
Process: For each file:
           if new OR modtime > stored modtime:
               add to filesToIndex
               if previously indexed:
                   add to filesToDelete (old chunks)
       ↓
Output: filesToIndex []FileInfo
        filesToDelete []string (file paths)
```

**Efficiency**: Only processes changed files. Example:
- 1000 files in repo
- 10 modified since last index
- Only index those 10 files

### 3. Semantic Chunking
**Component**: Chunker (`internal/chunker/semantic.go`)

```
Input: FilePath, Language
       ↓
Decision: Language supported by tree-sitter?
          ↓
        Yes                    No
         ↓                      ↓
    Parse with           Blank-line
    tree-sitter          chunking
         ↓                      ↓
    Extract semantic      Split on
    units (func,          blank lines
    method, type)              ↓
         ↓                      ↓
       Chunks              Chunks
         ↓                      ↓
         └──────────┬───────────┘
                    ↓
Output: []Chunk{ID, FilePath, Lines, Code, Type, Name, Metadata}
```

Example semantic chunk (Go function):
```go
Chunk{
    ID:        "uuid-abc-123",
    FilePath:  "/path/to/file.go",
    LineStart: 45,
    LineEnd:   67,
    Language:  "go",
    Code:      "func Add(a, b int) int {\n  return a + b\n}",
    ChunkType: "function",
    Name:      "Add",
    Metadata:  {"package": "main", "params": "a, b int"},
}
```

**Code**: `semantic.ChunkFile()` at internal/chunker/semantic.go:29-80

### 4. Content Hashing
**Component**: Index Command (`cmd/code-scout/index.go:147-161`)

```
Input: []Chunk from chunker
       ↓
Process: For each chunk:
           hash = SHA256(chunk.Code)
           if hash not in hashToFirstIndex:
               hashToFirstIndex[hash] = index
               mark as unique
           else:
               mark as duplicate of hashToFirstIndex[hash]
       ↓
Output: hashToFirstIndex map[string]int
        chunkHashes []string

Stats: uniqueCount, duplicateCount
```

Example:
```
Input:  1111 chunks
Hash:   983 unique hashes, 128 duplicates (11.5%)
Result: Only generate 983 embeddings
```

**Code**: `computeContentHash()` at cmd/code-scout/index.go:23-26

### 5. Embedding Generation
**Component**: Embeddings (`internal/embeddings/ollama.go`)

```
Input: Unique chunks only (from hash deduplication)
       Number of concurrent workers (default: 10)
       ↓
Process: Worker Pool Pattern:
           jobs channel ← unique chunks
           results channel ← embeddings

           For each worker (goroutine):
               chunk = <-jobs
               POST /api/embeddings to Ollama
               embedding = response.Embedding
               results <- embedding
       ↓
Output: [][]float64 (embeddings for unique chunks)
        Copy to duplicate chunks
```

API call example:
```json
POST http://localhost:11434/api/embeddings
{
  "model": "code-scout-code",
  "prompt": "func Add(a, b int) int {\n  return a + b\n}"
}

Response:
{
  "embedding": [0.123, -0.456, 0.789, ...] // 3584 dimensions
}
```

**Code**:
- `OllamaClient.Embed()` at internal/embeddings/ollama.go:46-78
- Worker pool at cmd/code-scout/index.go:169-224

### 6. Vector Storage
**Component**: Storage (`internal/storage/lancedb.go`)

```
Input: []Chunk, [][]float64 (embeddings)
       ↓
Process: Convert to Arrow format:
           chunk_id    → StringArray
           file_path   → StringArray
           line_start  → Int32Array
           line_end    → Int32Array
           language    → StringArray
           code        → LargeStringArray
           vector      → FixedSizeList<Float32>[3584]

         Create Arrow RecordBatch
         Append to LanceDB table
       ↓
Output: Data persisted to .code-scout/code_chunks.lance
```

LanceDB storage format:
```
.code-scout/
├── code_chunks.lance/
│   ├── data/
│   │   ├── 0.parquet    # Arrow/Parquet format
│   │   └── 1.parquet
│   └── _versions/
│       └── 1.manifest
└── metadata.json
```

**Code**: `LanceDBStore.StoreChunks()` at internal/storage/lancedb.go:111-168

### 7. Metadata Update
**Component**: Storage (`internal/storage/metadata.go`)

```
Input: []FileInfo (indexed files)
       Current time
       ↓
Process: metadata.LastIndexTime = now
         For each file:
             metadata.FileModTimes[file.Path] = file.ModTime
         Remove deleted files from map
         Marshal to JSON
       ↓
Output: .code-scout/metadata.json
```

Metadata structure:
```json
{
  "last_index_time": "2025-11-15T14:30:45Z",
  "file_mod_times": {
    "/path/to/file1.go": "2025-11-15T10:20:30Z",
    "/path/to/file2.go": "2025-11-14T16:45:12Z"
  }
}
```

**Code**: `SaveMetadata()` at internal/storage/metadata.go:34-52

## Search Flow

### 1. Query Embedding
**Component**: Search Command + Embeddings

```
Input: Query string (e.g., "error handling")
       ↓
Process: POST to Ollama /api/embeddings
         Same model as indexing (code-scout-code)
       ↓
Output: []float64 query embedding (3584 dimensions)
```

**Code**: cmd/code-scout/search.go:46-50

### 2. Vector Search
**Component**: Storage (`internal/storage/lancedb.go`)

```
Input: Query embedding []float64
       Limit (default: 10)
       ↓
Process: LanceDB nearest neighbor search:
           metric: cosine distance
           SELECT TOP 10 *
           ORDER BY vector_distance(query, vector)
       ↓
Output: []map[string]interface{} raw results
```

Result format:
```go
map[string]interface{}{
    "chunk_id":   "uuid-...",
    "file_path":  "/path/to/file.go",
    "line_start": 45,
    "line_end":   67,
    "language":   "go",
    "code":       "func Add...",
    "_distance":  0.123,  // Lower is better
}
```

**Code**: `LanceDBStore.Search()` at internal/storage/lancedb.go:170-182

### 3. Result Formatting
**Component**: Search Command

```
Input: []map[string]interface{} from LanceDB
       ↓
Process: Convert to typed structs:
           Extract fields with type conversion
           Score = _distance
       ↓
Output: []SearchResult
```

**Code**: `formatResults()` at cmd/code-scout/search.go:107-121

### 4. Result Deduplication
**Component**: Search Command

```
Input: []SearchResult (may have duplicates)
       ↓
Process: Group by Code content:
           For each unique code:
               Keep result with lowest score (best match)
           Sort by score (ascending)
       ↓
Output: []SearchResult (deduplicated)

Stats: Before and after counts for user feedback
```

Example deduplication:
```
Before: 10 results
  - "package storage" × 2  (scores: 0.0, 0.1)
  - "package parser" × 6   (scores: 3012, 3015, 3018, ...)
  - "package scanner" × 1  (score: 3038)
  - Other code × 1         (score: 4000)

After: 4 results
  - "package storage" (score: 0.0)    ← kept best
  - "package parser"  (score: 3012)   ← kept best
  - "package scanner" (score: 3038)
  - Other code        (score: 4000)
```

**Code**: `deduplicateResults()` at cmd/code-scout/search.go:123-165

### 5. Output Formatting
**Component**: Search Command

```
Input: []SearchResult (deduplicated)
       Output format (JSON or human-readable)
       ↓
Decision: --json flag?
          ↓
       Yes                    No
        ↓                      ↓
    JSON format          Human-readable
    {                    "Found X unique
      "query": "...",     results (from Y
      "results": [...]    total)..."
    }                          ↓
        ↓                      ↓
        └──────────┬───────────┘
                   ↓
Output: Printed to stdout
```

**Code**: cmd/code-scout/search.go:72-91

## Data Transformations Summary

```
Source Code (.go, .py)
    ↓ [Scanner]
FileInfo {path, language, modtime}
    ↓ [Incremental Filter]
Files to Index
    ↓ [Tree-sitter Parser]
AST Nodes
    ↓ [Chunker]
Chunks {code, metadata, lines}
    ↓ [Content Hash]
Unique Chunks (deduplicated)
    ↓ [Ollama API]
Embeddings []float64[3584]
    ↓ [Arrow Conversion]
RecordBatch
    ↓ [LanceDB]
Persisted Vectors

─────── SEARCH ───────

Query String
    ↓ [Ollama API]
Query Embedding []float64[3584]
    ↓ [LanceDB Vector Search]
Raw Results (with duplicates)
    ↓ [Format + Deduplicate]
SearchResults (unique)
    ↓ [JSON/Human Format]
Output
```

## Performance Bottlenecks

1. **Embedding Generation** (slowest)
   - Mitigated by: Worker pool (10 concurrent)
   - Mitigated by: Content deduplication (11% reduction)

2. **File Scanning** (fast, but grows with repo size)
   - Mitigated by: .gitignore filtering
   - Mitigated by: Incremental updates

3. **Tree-sitter Parsing** (medium)
   - Fast for most files
   - Fallback to simple chunking if needed

4. **Vector Search** (very fast)
   - LanceDB optimized for ANN search
   - Sub-second response times

## Error Handling Points

Each stage has error handling:
- Scanner: Permission errors, invalid paths
- Parser: Malformed code, unsupported syntax
- Embeddings: Ollama connection, API errors
- Storage: Disk space, corrupted database
- Search: Empty database, invalid query

See individual component docs for detailed error handling.
