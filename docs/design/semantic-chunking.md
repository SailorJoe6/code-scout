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
   - `Extractor.ExtractFunctions(ctx)` parses the file, caches package/import metadata, and walks the AST.
   - Go keeps specialized extractors (`extractFunction`, `extractMethod`, `extractTypes`) to preserve receivers, signatures, and field metadata.
   - All other languages share `extractGenericNode`, which maps tree-sitter node kinds to Code Scout chunk types through `mapNodeKindToChunkType`.
   - Doc comments, receivers, signatures, imports, packages, and docstrings are added to `Chunk.Metadata` so embeddings capture intent.

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

Each `.scm` file lists node patterns we care about per language. `extractGenericNode` keeps the implementation compact by translating these node types into Code Scout chunk types.

## Metadata Captured Per Chunk

- **Structural context**: chunk type, name, file path, start/end lines.
- **Language**: `chunk.Language` plus `Metadata["language"]` (currently always `"go"` for Go chunks, generic importer does not overwrite non-Go languages).
- **Doc comments**: attached for any node preceded by documentation comments (Go) or docstring nodes (Python, PHP, Ruby).
- **Signatures and receivers**: stored for Go functions/methods; generic extractor adds whatever identifier or name nodes exist.
- **Package/imports**: Go-specific metadata extracted before traversal via `extractFileMetadata`.
- **Heading context**: provided by the Markdown chunker so docs preserve navigation cues.
- **EmbeddingType**: drives whether `code-scout-code` or `code-scout-text` embeddings are generated.

## Language-specific Notes

- **Go** – still the richest metadata path. `extractFunction`, `extractMethod`, and `extractTypes` in `internal/parser/extractor.go` capture receivers, field lists, imports, and package names.
- **Python** – `function_definition`, `class_definition`, and `decorated_definition` nodes map to functions/classes. Docstrings remain part of the chunk body so embeddings can learn semantics.
- **JavaScript / TypeScript** – both share the same parser. Arrow functions, generator functions, and class components map to `function` or `method` chunk types while JSX/TSX syntax passes through untouched because tree-sitter scopes it.
- **Java** – classes, interfaces, enums, records, constructors, and methods are emitted; nested types become individual chunks so embeddings understand inner classes.
- **Rust** – `function_item`, `impl_item`, `trait_item`, `struct_item`, and `enum_item` help represent inherent impls and trait impls separately.
- **C / C++** – heuristics decide which parser to use. Structs, enums, classes, namespaces, and free functions become chunks even when located in headers.
- **Ruby** – class/module nesting plus singleton methods are preserved by `ruby.scm` so DSL-heavy code (e.g., Rails) still chunks around method boundaries.
- **PHP** – functions, methods, classes, interfaces, traits, and enums are supported; doc comments and attributes remain in `Chunk.Code`.
- **Scala** – `function_definition`, `class_definition`, `trait_definition`, and `object_definition` nodes emit chunks so both OO and functional constructs are indexed.

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
