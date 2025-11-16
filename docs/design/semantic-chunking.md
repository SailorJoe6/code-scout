# Semantic Chunking with Tree-sitter

## Overview

Semantic chunking is Code Scout's approach to dividing source code into meaningful units that preserve structure and context. Unlike naive line-based or blank-line splitting, semantic chunking uses tree-sitter to parse code and extract complete functions, methods, types, and other language constructs.

## Problem: Naive Chunking

Traditional approaches split code arbitrarily:

**Blank-line chunking**:
```go
// Chunk 1
package main

// Chunk 2
import "fmt"

// Chunk 3
func Add(a, b int) int {
    return a + b
}

// Chunk 4
func Multiply(a, b int) int {
    return a * b
}
```

**Problems**:
- Loses context (package, imports separated)
- Breaks logical units across chunks
- No semantic understanding
- Embedding doesn't capture "what this code does"

## Solution: Tree-sitter Semantic Extraction

Tree-sitter parses source code into an Abstract Syntax Tree (AST), allowing us to extract complete, meaningful units.

**Semantic chunking** (same code):
```go
// Chunk 1: Function
{
    ChunkType: "function",
    Name: "Add",
    Code: "func Add(a, b int) int {\n    return a + b\n}",
    Metadata: {
        "package": "main",
        "imports": "fmt",
        "params": "a, b int",
        "returns": "int",
    }
}

// Chunk 2: Function
{
    ChunkType: "function",
    Name: "Multiply",
    Code: "func Multiply(a, b int) int {\n    return a * b\n}",
    Metadata: {
        "package": "main",
        "imports": "fmt",
        "params": "a, b int",
        "returns": "int",
    }
}
```

**Benefits**:
- Complete semantic units (full function/method/type)
- Context preserved in metadata
- Better embeddings (captures function's purpose)
- More relevant search results

## Tree-sitter Integration

### Parser Setup

Code Scout uses [go-tree-sitter](https://github.com/tree-sitter/go-tree-sitter) with language-specific grammars:

```go
// Create parser for Go
parser := sitter.NewParser()
lang := sitter.NewLanguage(tree_sitter_go.Language())
parser.SetLanguage(lang)

// Parse source code
tree, err := parser.ParseCtx(ctx, nil, source)
root := tree.RootNode()
```

**Supported Languages**:
- Go (`tree-sitter-go`)
- Python (`tree-sitter-python`)

**Implementation**: internal/parser/treesitter.go:16-28

### AST Traversal

Tree-sitter provides an AST where each node has:
- **Type**: e.g., "function_declaration", "method_declaration", "type_declaration"
- **Children**: Nested nodes (parameters, body, etc.)
- **Position**: Line and column numbers in source

Example AST for `func Add(a, b int) int { return a + b }`:
```
function_declaration
├── func [keyword]
├── identifier: "Add"
├── parameter_list
│   ├── parameter_declaration: "a, b int"
├── type_identifier: "int"
└── block
    └── return_statement
        └── binary_expression: "a + b"
```

Code Scout walks this tree looking for specific node types:
```go
func (e *Extractor) walkNode(node *sitter.Node, source []byte, chunks *[]Chunk) {
    switch node.Type() {
    case "function_declaration":
        chunk := e.extractFunction(node, source)
        *chunks = append(*chunks, chunk)
    case "method_declaration":
        chunk := e.extractMethod(node, source)
        *chunks = append(*chunks, chunk)
    case "type_declaration":
        chunk := e.extractType(node, source)
        *chunks = append(*chunks, chunk)
    // ... other types
    }

    // Recurse to children
    for i := 0; i < int(node.ChildCount()); i++ {
        e.walkNode(node.Child(i), source, chunks)
    }
}
```

**Implementation**: internal/parser/extractor.go:52-100

## Go-Specific Extraction

### Functions

**Tree-sitter Node Type**: `function_declaration`

**Extracted Information**:
```go
func Add(a, b int) int {
    return a + b
}

→

Chunk{
    ChunkType: "function",
    Name:      "Add",
    Content:   "func Add(a, b int) int {\n    return a + b\n}",
    StartLine: 10,
    EndLine:   12,
    Metadata: {
        "package":  "math",
        "imports":  "...",
        "params":   "a, b int",
        "returns":  "int",
    },
}
```

**Extraction Logic**:
1. Find `identifier` child → function name
2. Find `parameter_list` → extract parameter string
3. Find return type (type_identifier or parameter_list after params)
4. Get full text from start to end position
5. Extract package and imports from file context

**Implementation**: internal/parser/extractor.go:155-269

### Methods

**Tree-sitter Node Type**: `method_declaration`

**Difference from Functions**: Methods have a receiver

```go
func (c *Calculator) Add(a, b int) int {
    return a + b
}

→

Chunk{
    ChunkType: "method",
    Name:      "Add",
    Content:   "func (c *Calculator) Add(a, b int) int {\n    return a + b\n}",
    Metadata: {
        "receiver":       "*Calculator",
        "receiver_type":  "Calculator",
        "receiver_name":  "c",
        "params":         "a, b int",
        "returns":        "int",
    },
}
```

**Extraction Logic**:
1. Find `parameter_list` (first one is receiver)
2. Extract receiver type (e.g., `*Calculator`)
3. Extract receiver name (e.g., `c`)
4. Continue like function extraction for params and return type

**Implementation**: internal/parser/extractor.go:271-341

### Types

Tree-sitter recognizes several type declarations in Go:

**1. Struct Types**:
```go
type User struct {
    Name  string
    Email string
}

→

Chunk{
    ChunkType: "struct",
    Name:      "User",
    Content:   "type User struct {\n    Name  string\n    Email string\n}",
    Metadata: {
        "fields": "Name string, Email string",
    },
}
```

**2. Interface Types**:
```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

→

Chunk{
    ChunkType: "interface",
    Name:      "Reader",
    Content:   "type Reader interface {...}",
    Metadata: {
        "methods": "Read",
    },
}
```

**3. Type Aliases**:
```go
type UserID string

→

Chunk{
    ChunkType: "type_alias",
    Name:      "UserID",
    Content:   "type UserID string",
    Metadata: {
        "underlying": "string",
    },
}
```

**Implementation**: internal/parser/extractor.go:343-469

### Context Metadata

Beyond the code itself, semantic chunking captures context:

**Package Information**:
```go
// Extracted from file's package declaration
"package": "handlers"
```

**Imports**:
```go
// Extracted from file's import block
"imports": "fmt, strings, github.com/user/pkg"
```

**Function Signature Details**:
```go
"params":  "ctx context.Context, id string",
"returns": "(User, error)",
```

**Struct Field Information**:
```go
"fields": "ID int, Name string, CreatedAt time.Time",
```

This metadata enriches the embedding, making searches more precise.

**Implementation**: internal/parser/extractor.go:472-520

## Fallback Behavior

If tree-sitter parsing fails (unsupported language, syntax errors), Code Scout falls back to basic blank-line chunking:

```go
chunks, err := semanticChunker.ChunkFile(filePath, language)
if err != nil {
    // Fall back to basic chunker
    chunks = basicChunker.ChunkFile(filePath, language)
}
```

This ensures Code Scout works with all files, even if semantic extraction isn't available.

**Implementation**: internal/chunker/semantic.go:40-62

## Performance Considerations

**Parsing Speed**:
- Tree-sitter is fast: ~500-1000 LOC/ms
- Parsing is usually <1% of total indexing time
- Embedding generation is the bottleneck

**Memory Usage**:
- Tree-sitter builds in-memory AST
- Released after extraction completes
- Minimal memory overhead

**Accuracy**:
- Tree-sitter handles most valid Go/Python syntax
- Gracefully handles minor syntax errors
- Falls back to basic chunking on parse failures

## Extending to New Languages

To add semantic chunking for a new language:

1. **Add tree-sitter grammar dependency**:
```go
import tree_sitter_rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
```

2. **Create parser function**:
```go
func NewRustParser() (*Parser, error) {
    parser := sitter.NewParser()
    lang := sitter.NewLanguage(tree_sitter_rust.Language())
    if err := parser.SetLanguage(lang); err != nil {
        return nil, err
    }
    return &Parser{parser: parser}, nil
}
```

3. **Implement extractor**:
```go
func (e *Extractor) ExtractRustChunks(source []byte, filePath string) ([]Chunk, error) {
    // Walk AST looking for:
    // - function_item
    // - struct_item
    // - impl_item
    // etc.
}
```

4. **Update semantic chunker**:
```go
case "rust":
    parser, _ := parser.NewRustParser()
    extractor := parser.NewExtractor(parser)
    return extractor.ExtractRustChunks(source, filePath)
```

See [extension-points.md](extension-points.md) for detailed guide.

## Testing

Semantic chunking has comprehensive tests:

**Unit Tests** (per language):
- `internal/parser/extractor_test.go` - Go extraction
- `internal/parser/types_test.go` - Type declarations
- `internal/parser/metadata_test.go` - Context metadata

**Integration Tests**:
- `internal/chunker/integration_test.go` - End-to-end chunking
- `internal/chunker/semantic_test.go` - Semantic chunker behavior

Run tests:
```bash
go test ./internal/parser/...
go test ./internal/chunker/...
```

## Examples from Codebase

### Example 1: Extracting `Embed` Method

**Source** (internal/embeddings/ollama.go:46-78):
```go
func (c *OllamaClient) Embed(text string) ([]float64, error) {
    reqBody := ollamaEmbedRequest{
        Model:  c.model,
        Prompt: text,
    }
    // ... HTTP request logic
    return embedResp.Embedding, nil
}
```

**Extracted Chunk**:
```go
{
    ChunkType: "method",
    Name:      "Embed",
    Content:   "func (c *OllamaClient) Embed(text string) ([]float64, error) {...}",
    StartLine: 46,
    EndLine:   78,
    Metadata: {
        "receiver":      "*OllamaClient",
        "receiver_type": "OllamaClient",
        "params":        "text string",
        "returns":       "([]float64, error)",
        "package":       "embeddings",
    },
}
```

### Example 2: Extracting `Chunk` Struct

**Source** (internal/chunker/chunker.go:13-23):
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
}
```

**Extracted Chunk**:
```go
{
    ChunkType: "struct",
    Name:      "Chunk",
    Content:   "type Chunk struct {...}",
    StartLine: 13,
    EndLine:   23,
    Metadata: {
        "fields": "ID string, FilePath string, LineStart int, ...",
        "package": "chunker",
    },
}
```

## Benefits for Search Quality

Semantic chunking dramatically improves search results:

**Query: "embed function"**

Without semantic chunking:
```
Result 1: "func (c *Ollama"  // Truncated
Result 2: "Client) Embed(tex" // Truncated
Result 3: "t string) ([]floa" // Truncated
```

With semantic chunking:
```
Result 1: "func (c *OllamaClient) Embed(text string) ([]float64, error) {...}"
  - Complete function
  - Context: OllamaClient receiver
  - Clear purpose: Generate embeddings
```

Better chunks → Better embeddings → Better search results
