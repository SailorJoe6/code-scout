# Extension Points

## Overview

Code Scout is designed for extensibility. This document describes how to add new languages, embedding models, vector databases, and other features.

## Adding New Languages

Code Scout uses Tree-sitter query files (.scm) for language-agnostic semantic chunking. Adding a new language requires three steps:

### Step 1: Add Tree-sitter Grammar Dependency

Add the tree-sitter grammar to **go.mod**:

```bash
go get github.com/tree-sitter/tree-sitter-<language>@latest
go mod tidy
```

Example for Rust:
```bash
go get github.com/tree-sitter/tree-sitter-rust@latest
```

### Step 2: Update Parser Factory

Add the language to **internal/parser/treesitter.go** in the `NewParser()` function:

```go
import tree_sitter_rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"

func NewParser(lang Language) (*Parser, error) {
    parser := sitter.NewParser()

    var tsLang *sitter.Language
    switch lang {
    case LanguageGo:
        tsLang = sitter.NewLanguage(tree_sitter_go.Language())
    case LanguageRust:  // NEW
        tsLang = sitter.NewLanguage(tree_sitter_rust.Language())
    // ... other languages
    default:
        return nil, fmt.Errorf("unsupported language: %s", lang.String())
    }

    if err := parser.SetLanguage(tsLang); err != nil {
        return nil, fmt.Errorf("failed to set language %s: %w", lang.String(), err)
    }

    return &Parser{parser: parser, language: lang}, nil
}
```

Add language constant to **internal/parser/language.go**:

```go
const (
    LanguageUnknown Language = iota
    LanguageGo
    LanguageRust  // NEW
    // ... other languages
)

func (l Language) String() string {
    switch l {
    case LanguageRust:
        return "rust"
    // ... other cases
    }
}

func DetectLanguage(filePath string, content []byte) Language {
    ext := strings.ToLower(filepath.Ext(filePath))
    switch ext {
    case ".rs":
        return LanguageRust
    // ... other cases
    }
}
```

### Step 3: Create Tree-sitter Query File

Create **internal/parser/queries/rust.scm** that extracts semantic chunks:

```scheme
; Rust Tree-sitter Query
; Extracts functions, structs, enums, traits for semantic chunking

; Function definitions
(function_item
  name: (identifier) @function.name
  parameters: (parameters) @function.parameters
  body: (block) @function.body) @function.definition

; Struct definitions
(struct_item
  name: (type_identifier) @struct.name
  body: (_)? @struct.body) @struct.definition

; Enum definitions
(enum_item
  name: (type_identifier) @enum.name
  body: (enum_variant_list) @enum.body) @enum.definition

; Trait definitions
(trait_item
  name: (type_identifier) @trait.name
  body: (declaration_list) @trait.body) @trait.definition

; Impl blocks
(impl_item
  type: (type_identifier) @impl.type
  body: (declaration_list) @impl.body) @impl.definition
```

**Query file guidelines:**
- Capture semantic units: functions, classes, methods, types
- Use `@name.type` capture names (e.g., `@function.name`, `@class.body`)
- Include both declarations and definitions
- Capture parameters, return types, and bodies where applicable
- Look at existing `.scm` files in `internal/parser/queries/` for examples

### Step 4: Test Your Language Support

Create **internal/parser/rust_test.go**:

```go
func TestRustParser(t *testing.T) {
    source := []byte(`
fn add(a: i32, b: i32) -> i32 {
    a + b
}

struct Point {
    x: f64,
    y: f64,
}
    `)

    parser, err := NewParser(LanguageRust)
    require.NoError(t, err)

    extractor := NewExtractor(parser, source)
    chunks, err := extractor.ExtractFunctions(context.Background())

    require.NoError(t, err)
    assert.GreaterOrEqual(t, len(chunks), 1)

    // Verify function extraction
    funcChunk := findChunkByName(chunks, "add")
    assert.NotNil(t, funcChunk)
    assert.Equal(t, ChunkTypeFunction, funcChunk.Type)
    assert.Contains(t, funcChunk.Content, "fn add")
}
```

Run tests:
```bash
go test ./internal/parser/...
```

### Understanding Tree-sitter Query Syntax

Tree-sitter queries use S-expressions to match AST patterns:

**Basic pattern matching:**
```scheme
; Match any function
(function_item) @function

; Match function with specific field
(function_item
  name: (identifier) @function.name)

; Match nested patterns
(class_definition
  body: (block
    (function_definition
      name: (identifier) @method.name)))
```

**Predicates:**
```scheme
; Only match functions named "main"
(function_item
  name: (identifier) @name
  (#match? @name "^main$"))

; Only match public functions
(function_item
  (visibility_modifier) @vis
  (#eq? @vis "pub"))
```

**Finding node names:**

Use the Tree-sitter playground or inspect the grammar:
```bash
# Install tree-sitter CLI
npm install -g tree-sitter-cli

# Parse a file and see AST node names
tree-sitter parse examples/test.rs
```

Or use the online playground: https://tree-sitter.github.io/tree-sitter/playground

### Currently Supported Languages

Code Scout currently has query files for:
- Go (`.go`)
- Python (`.py`)
- JavaScript/TypeScript (`.js`, `.jsx`, `.ts`, `.tsx`)
- Java (`.java`)
- Rust (`.rs`)
- C (`.c`, `.h`)
- C++ (`.cpp`, `.cc`, `.cxx`, `.hpp`, `.hxx`, `.h`)
- Ruby (`.rb`)
- PHP (`.php`)
- Scala (`.scala`)

Each has a corresponding `.scm` file in `internal/parser/queries/` that you can use as a reference.

## Changing Embedding Model

### Option 1: Different Ollama Model

**internal/embeddings/ollama.go**:
```go
const (
    DefaultOllamaEndpoint = "http://localhost:11434"
    DefaultCodeModel      = "custom-model"  // Change this
)
```

Create custom model:
```bash
# ollama-models/custom-model.Modelfile
FROM nomic-embed-text
PARAMETER num_ctx 8192

ollama create custom-model -f ollama-models/custom-model.Modelfile
```

**Update vector dimensions** if model has different size:

**internal/storage/lancedb.go**:
```go
// Change from 3584 to new dimension
Type: arrow.FixedSizeListOf(768, arrow.PrimitiveTypes.Float32)  // Example: 768 for nomic-embed-text
```

**Re-index** from scratch:
```bash
rm -rf .code-scout/
code-scout index
```

### Option 2: OpenAI Embeddings

Create new file **internal/embeddings/openai.go**:
```go
package embeddings

import (
    "bytes"
    "encoding/json"
    "net/http"
    "os"
)

type OpenAIClient struct {
    apiKey string
    model  string
    client *http.Client
}

func NewOpenAIClient() *OpenAIClient {
    return &OpenAIClient{
        apiKey: os.Getenv("OPENAI_API_KEY"),
        model:  "text-embedding-3-small",  // 1536 dimensions
        client: &http.Client{},
    }
}

func (c *OpenAIClient) Embed(text string) ([]float64, error) {
    reqBody := map[string]interface{}{
        "model": c.model,
        "input": text,
    }

    jsonData, _ := json.Marshal(reqBody)
    req, _ := http.NewRequest("POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonData))
    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Data []struct {
            Embedding []float64 `json:"embedding"`
        } `json:"data"`
    }

    json.NewDecoder(resp.Body).Decode(&result)
    return result.Data[0].Embedding, nil
}
```

**Update index command**:
```go
// Replace
embedClient := embeddings.NewClient()
// With
embedClient := embeddings.NewClientWithEndpoint("https://api.openai.com", "text-embedding-3-small")
```

**Update schema dimensions** (as above).

## Changing Vector Database

### Creating Storage Interface

Extract common interface **internal/storage/interface.go**:
```go
package storage

type VectorStore interface {
    CreateTable() error
    OpenTable() error
    StoreChunks(chunks []Chunk, embeddings [][]float64) error
    DeleteChunksByFilePath(filePaths []string) error
    Search(queryEmbedding []float64, limit int) ([]map[string]interface{}, error)
    Close() error
}

type MetadataStore interface {
    LoadMetadata() (*Metadata, error)
    SaveMetadata(metadata *Metadata) error
}
```

### Implementing Alternative DB

Create **internal/storage/qdrant.go**:
```go
package storage

import (
    "context"
    qdrant "github.com/qdrant/go-client/qdrant"
)

type QdrantStore struct {
    client     *qdrant.Client
    collection string
}

func NewQdrantStore(url string) (*QdrantStore, error) {
    client, err := qdrant.NewClient(&qdrant.Config{
        Host: url,
    })
    if err != nil {
        return nil, err
    }

    return &QdrantStore{
        client:     client,
        collection: "code_chunks",
    }, nil
}

func (s *QdrantStore) CreateTable() error {
    ctx := context.Background()

    // Create collection with vector config
    return s.client.CreateCollection(ctx, &qdrant.CreateCollection{
        CollectionName: s.collection,
        VectorsConfig: qdrant.VectorParams{
            Size:     3584,
            Distance: qdrant.Distance_Cosine,
        },
    })
}

func (s *QdrantStore) StoreChunks(chunks []Chunk, embeddings [][]float64) error {
    ctx := context.Background()

    points := make([]*qdrant.PointStruct, len(chunks))
    for i, chunk := range chunks {
        points[i] = &qdrant.PointStruct{
            Id: qdrant.NewIDNum(uint64(i)),
            Vectors: qdrant.NewVectors(embeddings[i]...),
            Payload: map[string]interface{}{
                "chunk_id":   chunk.ID,
                "file_path":  chunk.FilePath,
                "line_start": chunk.LineStart,
                "line_end":   chunk.LineEnd,
                "language":   chunk.Language,
                "code":       chunk.Code,
            },
        }
    }

    _, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
        CollectionName: s.collection,
        Points:         points,
    })

    return err
}

func (s *QdrantStore) Search(queryEmbedding []float64, limit int) ([]map[string]interface{}, error) {
    ctx := context.Background()

    results, err := s.client.Search(ctx, &qdrant.SearchPoints{
        CollectionName: s.collection,
        Vector:         queryEmbedding,
        Limit:          uint64(limit),
        WithPayload:    qdrant.NewWithPayload(true),
    })

    if err != nil {
        return nil, err
    }

    // Convert to common format
    return convertQdrantResults(results), nil
}

// Implement other interface methods...
```

**Update command**:
```go
// Replace
store, err := storage.NewLanceDBStore(cwd)
// With
store, err := storage.NewQdrantStore("localhost:6333")
```

## Adding New Chunk Types

### Custom Chunk Metadata

Extend the Chunk struct **internal/chunker/chunker.go**:
```go
type Chunk struct {
    ID        string
    FilePath  string
    LineStart int
    LineEnd   int
    Language  string
    Code      string
    ChunkType string
    Name      string
    Metadata  map[string]string

    // NEW: Add domain-specific fields
    Complexity int     // Cyclomatic complexity
    TestCoverage float64  // If applicable
    Dependencies []string // Imported packages
}
```

Update extraction logic to populate new fields.

### New Chunk Types

For JavaScript/TypeScript:
```go
case "class_declaration":
    chunk := Chunk{
        ChunkType: "class",
        Name:      extractClassName(node),
        Metadata: map[string]string{
            "extends":    extractSuperclass(node),
            "implements": extractInterfaces(node),
            "methods":    listMethods(node),
        },
    }
```

For Python:
```go
case "decorated_definition":
    chunk := Chunk{
        ChunkType: "decorated_function",
        Name:      extractDecoratedName(node),
        Metadata: map[string]string{
            "decorators": extractDecorators(node),
            "is_async":   isAsyncFunction(node),
        },
    }
```

## Adding Search Filters

### Filter by Language

**cmd/code-scout/search.go**:
```go
var (
    jsonOutput bool
    limitFlag  int
    langFilter string  // NEW
)

searchCmd.Flags().StringVar(&langFilter, "language", "", "Filter by language (go, python, etc.)")
```

**Filtering logic**:
```go
results, err := store.Search(queryEmbedding, limitFlag)

// Filter by language
if langFilter != "" {
    filtered := []map[string]interface{}{}
    for _, r := range results {
        if r["language"] == langFilter {
            filtered = append(filtered, r)
        }
    }
    results = filtered
}
```

### Filter by File Path

```go
searchCmd.Flags().StringVar(&pathFilter, "path", "", "Filter by file path pattern")

// Filter
if pathFilter != "" {
    filtered := []map[string]interface{}{}
    for _, r := range results {
        if strings.Contains(r["file_path"].(string), pathFilter) {
            filtered = append(filtered, r)
        }
    }
    results = filtered
}
```

### Threshold Filtering

```go
searchCmd.Flags().Float64Var(&minScore, "min-score", 0, "Minimum relevance score")
searchCmd.Flags().Float64Var(&maxScore, "max-score", 10000, "Maximum relevance score")

// Filter
filtered := []SearchResult{}
for _, r := range results {
    if r.Score >= minScore && r.Score <= maxScore {
        filtered = append(filtered, r)
    }
}
```

## Adding New Commands

### stats Command

**cmd/code-scout/stats.go**:
```go
package main

import (
    "fmt"
    "os"

    "github.com/jlanders/code-scout/internal/storage"
    "github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
    Use:   "stats",
    Short: "Show database statistics",
    RunE: func(cmd *cobra.Command, args []string) error {
        cwd, _ := os.Getwd()
        store, err := storage.NewLanceDBStore(cwd)
        if err != nil {
            return err
        }
        defer store.Close()

        if err := store.OpenTable(); err != nil {
            return err
        }

        // Get stats
        count, _ := store.table.Count(context.Background())
        metadata, _ := store.LoadMetadata()

        fmt.Printf("Database Statistics:\n")
        fmt.Printf("  Total chunks: %d\n", count)
        fmt.Printf("  Indexed files: %d\n", len(metadata.FileModTimes))
        fmt.Printf("  Last indexed: %s\n", metadata.LastIndexTime)

        return nil
    },
}

func init() {
    rootCmd.AddCommand(statsCmd)
}
```

Register in **cmd/code-scout/main.go**.

## Performance Optimizations

### Caching Query Embeddings

**internal/embeddings/cache.go**:
```go
type CachedClient struct {
    client *OllamaClient
    cache  map[string][]float64
    mu     sync.RWMutex
}

func (c *CachedClient) Embed(text string) ([]float64, error) {
    c.mu.RLock()
    if cached, ok := c.cache[text]; ok {
        c.mu.RUnlock()
        return cached, nil
    }
    c.mu.RUnlock()

    embedding, err := c.client.Embed(text)
    if err != nil {
        return nil, err
    }

    c.mu.Lock()
    c.cache[text] = embedding
    c.mu.Unlock()

    return embedding, nil
}
```

### Batch Processing

For very large codebases, batch the indexing:

```go
const BATCH_SIZE = 1000

for i := 0; i < len(chunks); i += BATCH_SIZE {
    end := min(i+BATCH_SIZE, len(chunks))
    batch := chunks[i:end]
    batchEmbeddings := embeddings[i:end]

    if err := store.StoreChunks(batch, batchEmbeddings); err != nil {
        return err
    }

    fmt.Printf("Stored %d/%d chunks\n", end, len(chunks))
}
```

## Testing Extensions

**Create test file** for new language:
```go
// internal/parser/rust_test.go
func TestRustExtraction(t *testing.T) {
    tests := []struct {
        name     string
        source   string
        expected []Chunk
    }{
        {
            name: "simple function",
            source: `fn add(a: i32, b: i32) -> i32 { a + b }`,
            expected: []Chunk{{ChunkType: "function", Name: "add"}},
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parser, _ := NewRustParser()
            chunks, err := parser.ExtractRustChunks([]byte(tt.source), "test.rs")
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, chunks)
        })
    }
}
```

---

These extension points make Code Scout adaptable to new languages, models, and use cases while maintaining a clean architecture.
