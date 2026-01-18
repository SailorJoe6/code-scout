# Multi-Language Support Implementation Specification

## Overview

Code Scout claims to support 11 programming languages with semantic code chunking, but currently only indexes Python and Go files. This specification defines the work required to deliver full multi-language support as documented in the README.

## Problem Statement

Two critical gaps prevent Code Scout from indexing 9 of its 11 "supported" languages:

1. **Scanner Discovery Gap**: The file scanner hardcodes only Python (`.py`) and Go (`.go`) extensions, filtering out all other language files before indexing
2. **Extraction Implementation Gap**: Tree-sitter query files exist for detailed semantic extraction but are never loaded or used; extraction falls back to a generic AST walker with limited capabilities

These gaps affect: JavaScript, TypeScript, Java, Rust, C, C++, Ruby, PHP, and Scala.

## Current State Analysis

### What Works
- ✅ Tree-sitter parser bindings installed for all 11 languages ([go.mod:11-21](../../go.mod))
- ✅ Language detection supports all file extensions ([language.go:57-109](../../internal/parser/language.go))
- ✅ Parser initialization handles all languages ([treesitter.go:26-67](../../internal/parser/treesitter.go))
- ✅ Chunker routes to tree-sitter for all code languages ([semantic.go:34](../../internal/chunker/semantic.go))
- ✅ Comprehensive test data files exist for all languages ([testdata/](../../internal/chunker/testdata/))
- ✅ Detailed tree-sitter query files written for all languages (709 total lines in [queries/](../../internal/parser/queries/))

### What's Broken

#### 1. Scanner Discovery
**File**: [internal/scanner/scanner.go:104-113](../../internal/scanner/scanner.go)

```go
var languageExtensions = map[string]string{
    ".py": "python",
    ".go": "go",
    ".md": "markdown",
    ".txt": "text",
    ".rst": "rst",
}
```

**Impact**: Only files with these 5 extensions are discovered. All other files are silently ignored during `code-scout index`.

**Missing Extensions**:
- JavaScript: `.js`, `.jsx`, `.mjs`, `.cjs`
- TypeScript: `.ts`, `.tsx`
- Java: `.java`
- Rust: `.rs`
- C: `.c`, `.h`
- C++: `.cpp`, `.cc`, `.cxx`, `.hpp`, `.hxx`, `.h`
- Ruby: `.rb`
- PHP: `.php`
- Scala: `.scala`

#### 2. Semantic Extraction
**File**: [internal/parser/extractor.go:657-704](../../internal/parser/extractor.go)

**Current behavior**: `extractGenericNode()` performs basic AST walking:
- Extracts node name from common field names or first identifier child
- Captures entire node content as single chunk
- Maps node kind to chunk type generically
- No language-specific logic

**Query files not used**: Despite 10 detailed `.scm` files defining precise extraction patterns, none are loaded or executed. No code references `embed`, `ReadFile`, or tree-sitter Query APIs.

**Extraction limitations**:

| Language | Current Extraction | Query Defines | Gap |
|----------|-------------------|---------------|-----|
| **Python** | Functions, classes (generic) | Functions, classes, methods, async functions, decorated functions/classes | No method/decorator distinction |
| **JavaScript** | Functions, classes (generic) | Functions, arrow functions, classes, methods, generators, exports | No arrow function/method/generator distinction |
| **TypeScript** | Same as JavaScript | Same as JavaScript plus interfaces, type aliases, enums | No TypeScript-specific constructs |
| **Java** | Classes, methods (generic) | Classes, interfaces, methods, constructors, enums, records, annotations | No constructor/interface/record distinction |
| **Rust** | Functions, structs, enums, impls (generic) | Functions, structs, enums, traits, impls, trait impls, modules, consts, statics, type aliases | No trait/module/const extraction |
| **C** | Functions, structs (generic) | Functions, structs, unions, enums, typedefs | No union/typedef extraction |
| **C++** | Functions, classes, structs, enums (generic) | Functions, classes, structs, enums, namespaces, templates, constructors, destructors, operators, type aliases | No namespace/template/constructor/destructor/operator extraction, extracts entire class instead of individual methods |
| **Ruby** | Methods, classes, modules (generic) | Methods, classes, modules, singleton methods, blocks | No singleton method distinction |
| **PHP** | Functions, classes (generic) | Functions, classes, methods, traits, interfaces, enums, namespaces | No trait/interface/namespace extraction |
| **Scala** | Functions, classes (generic) | Functions, classes, objects, traits, case classes, method definitions | No object/trait/case class distinction |

**Go extraction**: Currently uses language-specific logic in `extractFunction()`, `extractMethod()`, `extractTypes()`. Works correctly but doesn't use the [go.scm](../../internal/parser/queries/go.scm) query file.

## Goals

1. **File Discovery**: Index all supported language files during `code-scout index`
2. **Semantic Extraction**: Extract language-specific constructs as defined in query files
3. **Consistency**: Use tree-sitter query-based extraction uniformly across all languages
4. **Test Coverage**: Verify extraction produces expected chunks for all test files

## Scope

### Supported Languages

All 11 languages must have full support:

| Language | Extensions | Key Constructs |
|----------|-----------|----------------|
| Go | `.go` | Functions, methods, structs, interfaces, constants, variables |
| Python | `.py` | Functions, classes, methods, async functions, decorators |
| JavaScript | `.js`, `.jsx`, `.mjs`, `.cjs` | Functions, classes, methods, arrow functions, generators |
| TypeScript | `.ts`, `.tsx` | Functions, classes, methods, interfaces, type aliases, enums |
| Java | `.java` | Classes, interfaces, methods, constructors, enums, records |
| Rust | `.rs` | Functions, structs, enums, traits, impls, modules |
| C | `.c`, `.h` | Functions, structs, unions, enums, typedefs |
| C++ | `.cpp`, `.cc`, `.cxx`, `.hpp`, `.hxx`, `.h` | Functions, classes, namespaces, templates, methods, constructors, destructors, operators |
| Ruby | `.rb` | Methods, classes, modules, singleton methods |
| PHP | `.php` | Functions, classes, methods, traits, interfaces, enums |
| Scala | `.scala` | Functions, classes, objects, traits, case classes |

### Out of Scope

- Adding new languages beyond the 11 listed
- Modifying embedding model selection
- Changing vector database schema
- Performance optimization (unless blocking)
- Query file modifications (use existing queries as-is)

## Technical Requirements

### 1. Scanner Enhancement

**File**: `internal/scanner/scanner.go`

**Required changes**:
- Replace hardcoded `languageExtensions` map with comprehensive extension mapping
- Support all extensions listed in [language.go:196-224](../../internal/parser/language.go)
- Maintain existing ignore pattern behavior (`.gitignore`, `.code-scout-ignore`)
- Preserve hidden file/directory filtering

**Extension mapping** (41 total extensions):
```
Python: .py
Go: .go
JavaScript: .js, .jsx, .mjs, .cjs
TypeScript: .ts, .tsx
Java: .java
Rust: .rs
C: .c, .h
C++: .cpp, .cc, .cxx, .hpp, .hxx, .h
Ruby: .rb
PHP: .php
Scala: .scala
Markdown: .md
Text: .txt
RST: .rst
```

**Edge cases**:
- `.h` files: Can be C or C++. Scanner assigns "c" or "cpp" based on heuristics in `DetectLanguage()` ([language.go:95-107](../../internal/parser/language.go))
- Multiple extensions per language: Scanner must map all to correct language name

### 2. Query-Based Extraction

**Files**: `internal/parser/extractor.go`, new query loader

**Architecture**:

1. **Query Loading**:
   - Embed all `.scm` files from `internal/parser/queries/` using Go `embed` directive
   - Load appropriate query file based on parser language
   - Create tree-sitter Query object for pattern matching
   - Cache queries to avoid recompilation

2. **Query Execution**:
   - Parse source code into AST
   - Execute language-specific query against AST
   - Process query matches into Chunk objects
   - Extract metadata (name, parameters, signatures, doc comments)

3. **Fallback Behavior**:
   - If query loading fails: Fall back to generic extraction with warning
   - If query execution fails: Fall back to generic extraction with warning
   - Log errors for debugging

**Query match processing**:
- Each query defines captures like `@function.name`, `@function.parameters`, `@function.body`
- Extract these captures from query matches
- Combine captures into Chunk with appropriate fields
- Handle nested constructs (methods within classes, etc.)

**Metadata extraction**:
- Function/method signatures (parameters + return types)
- Doc comments (language-specific comment styles)
- Decorators/attributes (Python, Java)
- Visibility modifiers (public/private/protected)
- Generic type parameters (C++, Java, Scala, Rust)

### 3. Unified Extraction Path

**Current**: Go uses specialized extractors, other languages use generic walker

**Target**: All languages use query-based extraction

**Migration**:
- Create query-based extractor using [go.scm](../../internal/parser/queries/go.scm)
- Deprecate `extractFunction()`, `extractMethod()`, `extractTypes()` for Go
- Route all languages through unified `extractWithQuery()` function
- Remove language-specific conditionals in `walkNode()`

### 4. Test Coverage

**Existing tests**: [multilang_test.go](../../internal/chunker/multilang_test.go) expects 10+ chunks per language

**Required**:
- All existing tests pass with query-based extraction
- Verify expected chunk types found for each language
- Verify expected names found for each language
- Add tests for:
  - Method extraction within classes (C++, Java, JavaScript, TypeScript)
  - Namespace extraction (C++, PHP, Scala)
  - Template extraction (C++, Rust generics)
  - Decorator/annotation extraction (Python, Java)
  - Async function extraction (Python, JavaScript, TypeScript)
  - Trait extraction (Rust, PHP, Scala)

**Test data**: Comprehensive samples already exist in [testdata/](../../internal/chunker/testdata/)

## Implementation Requirements

### Phase 1: Scanner Fix
1. Update `languageExtensions` map to include all 41 extensions
2. Verify scanner discovers files for all languages
3. Test ignore pattern behavior remains unchanged
4. Update scanner tests

### Phase 2: Query Infrastructure
1. Add `embed` directive for query files
2. Implement query loader with caching
3. Implement query executor with error handling
4. Create Chunk builder from query matches
5. Add unit tests for query loading/execution

### Phase 3: Language-Specific Extraction
1. Implement query-based extraction for each language:
   - Python: Functions, classes, methods, async, decorators
   - JavaScript: Functions, classes, methods, arrows, generators
   - TypeScript: All JavaScript + interfaces, type aliases, enums
   - Java: Classes, interfaces, methods, constructors, enums, records
   - Rust: Functions, structs, enums, traits, impls, modules
   - C: Functions, structs, unions, enums, typedefs
   - C++: Functions, classes, namespaces, templates, constructors, destructors, operators
   - Ruby: Methods, classes, modules, singleton methods
   - PHP: Functions, classes, methods, traits, interfaces, namespaces
   - Scala: Functions, classes, objects, traits, case classes
   - Go: Migrate to query-based from current specialized extractors
2. Verify each language extracts expected constructs from test data
3. Update integration tests

### Phase 4: Deprecation & Cleanup
1. Remove `extractGenericNode()` and related generic extraction code
2. Remove Go-specific `extractFunction()`, `extractMethod()`, `extractTypes()`
3. Remove language-specific conditionals in `walkNode()`
4. Simplify `Extractor` to use only query-based path
5. Update documentation

## Success Criteria

### Functional
- ✅ `code-scout index` discovers and indexes files for all 11 languages
- ✅ Search results include chunks from all language files
- ✅ Each language extracts constructs listed in README table
- ✅ Query files are loaded and executed for all languages
- ✅ All existing tests pass
- ✅ New tests verify language-specific extraction

### Quality
- ✅ Extraction granularity matches query definitions (not entire classes)
- ✅ Chunk metadata includes names, signatures, doc comments
- ✅ Error handling degrades gracefully to generic extraction
- ✅ No silent failures during indexing

### Documentation
- ✅ README accurately reflects implemented support
- ✅ Code comments explain query-based extraction
- ✅ Test data covers all supported constructs

## Constraints

### Compatibility
- Maintain existing LanceDB schema
- Preserve existing config file format
- No breaking changes to CLI interface
- Existing indexes should remain valid

### Dependencies
- Use existing tree-sitter bindings (no version changes)
- Use existing query files as-is (no query modifications)
- No new external dependencies

### Performance
- Query compilation cached per language
- No significant slowdown vs current generic extraction
- Memory usage remains reasonable for large codebases

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Query syntax errors in existing `.scm` files | Extraction fails for affected language | Validate queries during tests, fallback to generic extraction |
| Tree-sitter API differences across languages | Inconsistent extraction quality | Test each language independently, document quirks |
| Large files cause query timeouts | Some files not indexed | Add query timeout, fallback to generic extraction |
| Ambiguous file extensions (`.h`) | Wrong language detection | Use existing heuristics in `DetectLanguage()` |

## Open Questions

1. **C++ method extraction**: Should methods be extracted individually or kept within class chunks? Query defines both patterns.
2. **Nested constructs**: How deep should nesting go (e.g., methods within classes within namespaces)?
3. **Performance tuning**: What's acceptable indexing time for large codebases?
4. **Query errors**: Log only or surface to user?

## References

- README: [README.md](../../README.md) - Language support claims
- Scanner: [internal/scanner/scanner.go](../../internal/scanner/scanner.go) - File discovery
- Language detection: [internal/parser/language.go](../../internal/parser/language.go) - Extension mapping
- Parser: [internal/parser/treesitter.go](../../internal/parser/treesitter.go) - Tree-sitter setup
- Extractor: [internal/parser/extractor.go](../../internal/parser/extractor.go) - Current extraction logic
- Queries: [internal/parser/queries/](../../internal/parser/queries/) - Tree-sitter query definitions
- Tests: [internal/chunker/multilang_test.go](../../internal/chunker/multilang_test.go) - Multi-language test suite
- Test data: [internal/chunker/testdata/](../../internal/chunker/testdata/) - Sample files
