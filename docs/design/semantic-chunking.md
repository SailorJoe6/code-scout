# Semantic Chunking with Tree-sitter

Semantic chunking is how Code Scout turns raw source files into semantically meaningful units that AI assistants can reason about. The current implementation supports 11 programming languages plus Markdown-style documentation while keeping the chunking pipeline uniform.

## End-to-End Pipeline

1. **Entry point** – `internal/chunker/semantic.go`
   - `SemanticChunker.ChunkFile(path, language)` receives every file selected by the scanner.
   - Markdown-like languages (`markdown`, `rst`, `text`) are routed to `MarkdownChunker`, which splits on headings and marks each chunk with `EmbeddingType: "docs"`.
   - Code files route to `chunkCode`, which reads the file, detects the precise language, and sets `EmbeddingType: "code"`.

2. **Language detection** – `internal/parser/language.go`
   - `parser.DetectLanguage(path, contents)` uses file extensions plus heuristics for ambiguous cases.
   - `.c` and `.h` files get promoted to C++ when C++-only markers (`class`, `namespace`, templates, etc.) appear; otherwise they stay C.
   - Detection output drives both parser selection and downstream metadata such as `Chunk.Language`.

3. **Parser construction** – `internal/parser/treesitter.go`
   - `parser.NewParser(lang)` instantiates a tree-sitter parser for the detected language (`tree-sitter-go`, `tree-sitter-python`, …).
   - TypeScript reuses the JavaScript grammar with TSX support enabled.

4. **Extraction** – `internal/parser/extractor.go`
   - `Extractor.ExtractFunctions(ctx)` uses query-based extraction for all 11 languages via `extractWithQuery`.
   - Tree-sitter query files (`internal/parser/queries/*.scm`) define language-specific patterns for extracting constructs.
   - `query_loader.go` embeds and caches compiled queries per language.
   - `query_executor.go` executes queries against the AST and builds chunks from captures.
   - Captures include names, parameters, signatures, doc comments, receivers, and other language-specific metadata.
   - If query loading/execution fails, falls back to minimal extraction via `extractGenericFallback`.

5. **Chunk normalization**
   - `SemanticChunker` wraps parser chunks into `chunker.Chunk` instances (UUID, file path, line span, chunk type, metadata, embedding type).
   - The indexer later batches `EmbeddingType: "code"` chunks with the code embedding model and `EmbeddingType: "docs"` chunks with the text model.

## Documentation Chunking

Markdown, reStructuredText, and plain text files never run through tree-sitter. `internal/chunker/markdown.go` evaluates heading depth, merges adjoining paragraphs, and emits document chunks that include the heading hierarchy in metadata. When a file does not contain headings (plain text/rst), the entire file becomes a single `ChunkType: "document"` segment so AI assistants retain context for design docs.

## Language Support Matrix

| Language    | Tree-sitter grammar                                | Query file                               | Chunk types emitted |
|-------------|----------------------------------------------------|------------------------------------------|--------------------|
| Go          | `tree-sitter-go`                                   | `internal/parser/queries/go.scm`         | functions, methods, structs, interfaces, const, var |
| Python      | `tree-sitter-python`                               | `internal/parser/queries/python.scm`     | functions, async functions, classes |
| JavaScript  | `tree-sitter-javascript`                           | `internal/parser/queries/javascript.scm` | functions, arrow functions, classes, methods |
| TypeScript  | `tree-sitter-javascript` (with TS queries)         | `internal/parser/queries/typescript.scm` | functions, arrow functions, classes, methods |
| Java        | `tree-sitter-java`                                 | `internal/parser/queries/java.scm`       | classes, interfaces, methods, constructors, enums |
| Rust        | `tree-sitter-rust`                                 | `internal/parser/queries/rust.scm`       | functions, impls, structs, enums, traits |
| C           | `tree-sitter-c`                                    | `internal/parser/queries/c.scm`          | functions, structs, unions, enums |
| C++         | `tree-sitter-cpp`                                  | `internal/parser/queries/cpp.scm`        | functions, classes, namespaces, templates |
| Ruby        | `tree-sitter-ruby`                                 | `internal/parser/queries/ruby.scm`       | methods, singleton methods, classes, modules |
| PHP         | `tree-sitter-php`                                  | `internal/parser/queries/php.scm`        | functions, classes, methods, interfaces, traits, enums |
| Scala       | `tree-sitter-scala`                                | `internal/parser/queries/scala.scm`      | functions, classes, objects, traits, case classes |

Each `.scm` file defines tree-sitter query patterns that capture language-specific constructs. Query executor processes captures to build chunks with precise metadata extraction.

## Metadata Captured Per Chunk

- **Structural context**: chunk type, name, file path, start/end lines.
- **Language**: `chunk.Language` plus `Metadata["language"]` for all languages.
- **Doc comments**: extracted via query captures for all languages that support them.
- **Signatures**: function/method parameters and return types captured from query patterns.
- **Receivers**: method receiver types (Go, C++, PHP, etc.) captured when present.
- **Package/imports**: extracted before query execution via `extractFileMetadata`.
- **Fields**: struct fields and interface methods stored in metadata for reference types.
- **Heading context**: provided by the Markdown chunker so docs preserve navigation cues.
- **EmbeddingType**: drives whether code or text embeddings are generated.

## Query-Based Extraction Architecture

All 11 languages use unified query-based extraction:

- **Query files** (`internal/parser/queries/*.scm`) define patterns using tree-sitter query syntax
- **Query loader** (`query_loader.go`) embeds query files at compile time and caches compiled queries per language
- **Query executor** (`query_executor.go`) matches queries against AST and builds chunks from captures
- **Fallback** (`extractGenericFallback`) returns empty chunks if query loading/execution fails

### Language-Specific Query Patterns

- **Go** – Functions, methods (with receivers), structs (with fields), interfaces (with method signatures), constants, variables
- **Python** – Functions, async functions, classes, methods, decorated definitions, docstrings
- **JavaScript/TypeScript** – Functions, arrow functions, classes, methods, generators, async functions, interfaces, type aliases, enums
- **Java** – Classes, interfaces, methods, constructors, enums, records, annotations
- **Rust** – Functions, structs, enums, traits, impls, trait impls, modules, constants, statics, type aliases
- **C** – Functions, structs, unions, enums, typedefs
- **C++** – Functions, classes, methods, constructors, destructors, operators, namespaces, templates
- **Ruby** – Methods, singleton methods, classes, modules
- **PHP** – Functions, classes, methods, interfaces, traits, enums, namespaces
- **Scala** – Functions, classes, objects, traits, case classes

Query captures extract names, parameters, signatures, doc comments, receivers, and other language-specific metadata automatically.

## Testing Coverage

- `internal/parser/extractor_test.go` – spot checks per-language node extraction helpers.
- `internal/chunker/semantic_test.go` – validates Markdown + Go/Python behavior and metadata wiring.
- `internal/chunker/multilang_test.go` – integration tests for all 11 languages using real fixture directories. Each test verifies minimum chunk counts, chunk types, metadata presence, and ensures every chunk has a non-empty body.
- `internal/chunker/integration_test.go` – runs the semantic chunker across portions of this repository to catch regressions.

Running all parser + chunker tests:

```bash
go test ./internal/parser/... ./internal/chunker/...
```

These tests should pass before asserting that semantic chunking handles a new language or grammar update.
