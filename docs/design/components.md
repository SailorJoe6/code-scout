# Code Scout Components

## Overview

Code Scout is organized into focused components with clear responsibilities. This document describes each component's purpose, interface, and implementation.

## Component Hierarchy

```
cmd/code-scout/          # CLI interface
├── main.go             # Entry point, command registration
├── index.go            # Index command implementation
└── search.go           # Search command implementation

internal/
├── scanner/            # File discovery
│   └── scanner.go
├── chunker/            # Code segmentation
│   ├── chunker.go      # Basic chunker (blank-line)
│   └── semantic.go     # Semantic chunker (tree-sitter)
├── parser/             # Tree-sitter integration
│   ├── treesitter.go   # Parser wrapper
│   ├── extractor.go    # Go code extraction
│   └── chunk.go        # Chunk type definitions
├── embeddings/         # Vector generation
│   └── ollama.go       # Ollama API client
└── storage/            # Persistence
    ├── lancedb.go      # Vector database operations
    └── metadata.go     # Incremental indexing metadata
```

## CLI Layer

### main.go
**Purpose**: Application entry point and command registration

**Responsibilities**:
- Register Cobra commands (index, search)
- Handle global flags
- Execute command router

**Interface**:
```go
// No exported interface - CLI entry point
func main()
```

**Implementation**: cmd/code-scout/main.go:1-30

---

### index.go
**Purpose**: Implement the `code-scout index` command

**Responsibilities**:
- Orchestrate the full indexing pipeline
- Manage incremental updates
- Coordinate all components
- Report progress to user

**Interface**:
```go
// Cobra command definition
var indexCmd = &cobra.Command{
    Use:   "index",
    Short: "Index the current directory for semantic search",
    RunE:  func(cmd *cobra.Command, args []string) error
}

// Flags
--workers int  // Number of concurrent embedding workers (default: 10)
```

**Key Functions**:
```go
// Content hash computation for deduplication
func computeContentHash(content string) string
```

**Implementation**: cmd/code-scout/index.go:20-241

**Pipeline Steps**:
1. Get working directory
2. Initialize storage and load metadata
3. Scan for code files
4. Determine files to index (incremental)
5. Delete old chunks for modified files
6. Chunk files with semantic chunker
7. Hash content for deduplication
8. Generate embeddings (worker pool)
9. Copy embeddings to duplicates
10. Store in LanceDB
11. Update metadata

---

### search.go
**Purpose**: Implement the `code-scout search` command

**Responsibilities**:
- Process search queries
- Generate query embeddings
- Execute vector search
- Deduplicate results
- Format output (JSON or human-readable)

**Interface**:
```go
// Cobra command definition
var searchCmd = &cobra.Command{
    Use:   "search [query]",
    Short: "Search the codebase semantically",
    Args:  cobra.ExactArgs(1),
    RunE:  func(cmd *cobra.Command, args []string) error
}

// Flags
--json        bool  // Output as JSON
--limit       int   // Max results (default: 10)
```

**Key Types**:
```go
type SearchResult struct {
    ChunkID   string
    FilePath  string
    LineStart int
    LineEnd   int
    Language  string
    Code      string
    Score     float64  // Cosine distance (lower is better)
}
```

**Key Functions**:
```go
// Format raw LanceDB results
func formatResults(results []map[string]interface{}) []SearchResult

// Remove duplicate code chunks
func deduplicateResults(results []SearchResult) []SearchResult
```

**Implementation**: cmd/code-scout/search.go:1-211

## Internal Components

### Scanner

**Purpose**: Discover code files in directory tree

**Package**: `internal/scanner`

**Interface**:
```go
type Scanner struct {
    rootDir string
}

type FileInfo struct {
    Path     string
    Language string
    ModTime  time.Time
}

func New(rootDir string) *Scanner
func (s *Scanner) ScanCodeFiles() ([]FileInfo, error)
```

**Responsibilities**:
- Walk directory tree recursively
- Filter using .gitignore patterns
- Filter using .code-scout-ignore patterns
- Detect language from file extension
- Return file metadata

**Language Detection**:
```go
".go"   → "go"
".py"   → "python"
".js"   → "javascript"
".ts"   → "typescript"
// etc.
```

**Implementation**: internal/scanner/scanner.go:16-123

**Example Usage**:
```go
scanner := scanner.New("/path/to/repo")
files, err := scanner.ScanCodeFiles()
// files = []FileInfo with all .go, .py, etc. files
```

---

### Chunker

**Purpose**: Segment code files into meaningful units

**Package**: `internal/chunker`

**Core Types**:
```go
type Chunk struct {
    ID        string            // UUID
    FilePath  string            // Source file path
    LineStart int               // Starting line
    LineEnd   int               // Ending line
    Language  string            // Programming language
    Code      string            // Actual code content
    ChunkType string            // function, method, struct, etc.
    Name      string            // Name of function/type
    Metadata  map[string]string // Context info
}

type Chunker interface {
    ChunkFile(filePath, language string) ([]Chunk, error)
}
```

**Implementations**:

1. **Basic Chunker** (chunker.go)
   - Splits on blank lines
   - Fallback for unsupported languages
   - Simple, fast, no dependencies

2. **Semantic Chunker** (semantic.go)
   - Uses tree-sitter for Go and Python
   - Extracts complete functions, methods, types
   - Captures context metadata
   - Falls back to basic chunker if parsing fails

**Interface**:
```go
func New() *Chunker  // Basic chunker
func NewSemantic() *SemanticChunker  // Tree-sitter chunker
```

**Implementation**:
- Basic: internal/chunker/chunker.go:26-92
- Semantic: internal/chunker/semantic.go:21-80

**Example Usage**:
```go
chunker := chunker.NewSemantic()
chunks, err := chunker.ChunkFile("main.go", "go")
// chunks = []Chunk with functions, methods, types
```

---

### Parser

**Purpose**: Tree-sitter integration for semantic code understanding

**Package**: `internal/parser`

**Core Types**:
```go
type Parser struct {
    parser *sitter.Parser
}

type Chunk struct {
    StartLine int
    EndLine   int
    Content   string
    ChunkType string
    Name      string
    Metadata  map[string]string
}
```

**Interface**:
```go
// Create parser for specific language
func NewGoParser() (*Parser, error)
func NewPythonParser() (*Parser, error)

// Parse source code
func (p *Parser) Parse(ctx context.Context, source []byte) (*sitter.Tree, error)
```

**Extractor** (extractor.go):
```go
type Extractor struct {
    parser *Parser
}

// Extract semantic chunks from Go code
func (e *Extractor) ExtractGoChunks(source []byte, filePath string) ([]Chunk, error)

// Specific extractors (called internally)
func (e *Extractor) extractFunction(node *sitter.Node, source []byte) Chunk
func (e *Extractor) extractMethod(node *sitter.Node, source []byte) Chunk
func (e *Extractor) extractType(node *sitter.Node, source []byte) Chunk
```

**What it Extracts**:

**Functions**:
```go
// Input Go code:
func Add(a, b int) int {
    return a + b
}

// Extracted chunk:
Chunk{
    ChunkType: "function",
    Name:      "Add",
    Content:   "func Add(a, b int) int {\n    return a + b\n}",
    Metadata: {
        "package":    "math",
        "params":     "a, b int",
        "returns":    "int",
        "imports":    "...",
    },
}
```

**Methods**:
```go
// Input Go code:
func (c *Calculator) Add(a, b int) int {
    return a + b
}

// Extracted chunk:
Chunk{
    ChunkType: "method",
    Name:      "Add",
    Metadata: {
        "receiver":   "*Calculator",
        "receiver_type": "Calculator",
    },
}
```

**Types** (structs, interfaces, type aliases):
```go
// Input Go code:
type User struct {
    Name  string
    Email string
}

// Extracted chunk:
Chunk{
    ChunkType: "struct",
    Name:      "User",
    Metadata: {
        "fields": "Name string, Email string",
    },
}
```

**Implementation**:
- Parser wrapper: internal/parser/treesitter.go:10-28
- Go extractor: internal/parser/extractor.go:24-520
- Chunk types: internal/parser/chunk.go:9-17

---

### Embeddings

**Purpose**: Generate vector embeddings using Ollama API

**Package**: `internal/embeddings`

**Interface**:
```go
type OllamaClient struct {
    endpoint string  // Default: http://localhost:11434
    model    string  // Default: code-scout-code
    client   *http.Client
}

func NewOllamaClient() *OllamaClient
func (c *OllamaClient) Embed(text string) ([]float64, error)
func (c *OllamaClient) EmbedBatch(texts []string) ([][]float64, error)
```

**API Protocol**:
```go
// Request
POST http://localhost:11434/api/embeddings
{
  "model": "code-scout-code",
  "prompt": "func Add(a, b int) int {...}"
}

// Response
{
  "embedding": [0.123, -0.456, 0.789, ...] // 3584 floats
}
```

**Models Used**:
- `code-scout-code`: Based on nomic-embed-code
- Embedding dimension: 3584
- Context window: 32K tokens

**Implementation**: internal/embeddings/ollama.go:18-93

**Example Usage**:
```go
client := embeddings.NewOllamaClient()
embedding, err := client.Embed("func main() {...}")
// embedding = []float64{0.123, -0.456, ...} // 3584 dims
```

**Batch Processing**:
```go
// Note: EmbedBatch is sequential, not concurrent
// For concurrency, use worker pool pattern (see index.go:169-224)
texts := []string{"chunk1", "chunk2", "chunk3"}
embeddings, err := client.EmbedBatch(texts)
```

---

### Storage

**Purpose**: Persist chunks and embeddings in LanceDB vector database

**Package**: `internal/storage`

**Core Types**:
```go
type LanceDBStore struct {
    dbPath string
    db     *lancedb.DB
    table  lancedb.Table
}

type Metadata struct {
    LastIndexTime time.Time
    FileModTimes  map[string]time.Time
}
```

**Interface**:
```go
// Database lifecycle
func NewLanceDBStore(rootDir string) (*LanceDBStore, error)
func (s *LanceDBStore) Close() error

// Table operations
func (s *LanceDBStore) CreateTable() error
func (s *LanceDBStore) OpenTable() error

// Data operations
func (s *LanceDBStore) StoreChunks(chunks []chunker.Chunk, embeddings [][]float64) error
func (s *LanceDBStore) DeleteChunksByFilePath(filePaths []string) error
func (s *LanceDBStore) Search(queryEmbedding []float64, limit int) ([]map[string]interface{}, error)

// Metadata operations
func (s *LanceDBStore) LoadMetadata() (*Metadata, error)
func (s *LanceDBStore) SaveMetadata(metadata *Metadata) error
```

**LanceDB Schema**:
```go
// Arrow schema definition
schema := arrow.NewSchema([]arrow.Field{
    {Name: "chunk_id", Type: arrow.BinaryTypes.String},
    {Name: "file_path", Type: arrow.BinaryTypes.String},
    {Name: "line_start", Type: arrow.PrimitiveTypes.Int32},
    {Name: "line_end", Type: arrow.PrimitiveTypes.Int32},
    {Name: "language", Type: arrow.BinaryTypes.String},
    {Name: "code", Type: arrow.BinaryTypes.LargeString},
    {Name: "vector", Type: arrow.FixedSizeListOf(3584, arrow.PrimitiveTypes.Float32)},
}, nil)
```

**Storage Location**:
```
.code-scout/
├── code_chunks.lance/     # LanceDB table
│   ├── data/
│   │   └── *.parquet      # Arrow/Parquet files
│   └── _versions/
│       └── *.manifest     # Version metadata
└── metadata.json          # Incremental update tracking
```

**Implementation**:
- LanceDB operations: internal/storage/lancedb.go:23-182
- Metadata persistence: internal/storage/metadata.go:10-65

**Example Usage**:
```go
// Indexing
store, _ := storage.NewLanceDBStore(".")
store.CreateTable()
store.StoreChunks(chunks, embeddings)
metadata := &storage.Metadata{
    LastIndexTime: time.Now(),
    FileModTimes: map[string]time.Time{
        "file.go": modTime,
    },
}
store.SaveMetadata(metadata)

// Searching
store.OpenTable()
results, _ := store.Search(queryEmbedding, 10)
```

## Component Dependencies

```
CLI (index.go, search.go)
    ↓
    ├─→ Scanner (find files)
    ├─→ Chunker (segment code)
    │     ↓
    │     └─→ Parser (tree-sitter)
    ├─→ Embeddings (generate vectors)
    └─→ Storage (persist/query)
            ↓
            └─→ LanceDB (vector database)
```

## Testing

Each component has unit tests:
- `internal/chunker/semantic_test.go`
- `internal/chunker/integration_test.go`
- `internal/parser/extractor_test.go`
- `internal/parser/types_test.go`
- `internal/parser/metadata_test.go`
- `internal/parser/treesitter_test.go`

Run tests:
```bash
go test ./internal/...
```

## Extension Points

To modify component behavior:

1. **Add new language**: Implement in `internal/parser/`
2. **Change chunking strategy**: Modify `internal/chunker/semantic.go`
3. **Use different embedding model**: Update `internal/embeddings/ollama.go`
4. **Change vector DB**: Replace `internal/storage/lancedb.go`

See [extension-points.md](extension-points.md) for details.
