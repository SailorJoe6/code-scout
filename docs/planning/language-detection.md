# Language Detection Strategy

## Overview

Code Scout uses file extensions and heuristic content analysis to determine the programming language of source files for semantic parsing with Tree-sitter.

## Supported Languages

### Fully Supported (Slice 4+)
- Python (`.py`)
- JavaScript (`.js`, `.jsx`)
- TypeScript (`.ts`, `.tsx`)
- Java (`.java`)
- Rust (`.rs`)
- C (`.c`, some `.h`)
- C++ (`.cpp`, `.cc`, `.cxx`, `.hpp`, `.hxx`, most `.h`)
- Ruby (`.rb`)
- PHP (`.php`)
- Scala (`.scala`)
- Go (`.go`) - Implemented in Slice 2

## File Extension Mapping

### Unambiguous Extensions

| Extension | Language | Notes |
|-----------|----------|-------|
| `.go` | Go | |
| `.py` | Python | |
| `.js`, `.jsx` | JavaScript | JSX for React |
| `.ts`, `.tsx` | TypeScript | TSX for React |
| `.java` | Java | |
| `.rs` | Rust | |
| `.rb` | Ruby | |
| `.php` | PHP | |
| `.scala` | Scala | |
| `.cpp`, `.cc`, `.cxx` | C++ | Source files |
| `.hpp`, `.hxx` | C++ | Header files |

### Ambiguous Extensions

#### `.c` Files
**Default**: C
**Exception**: If file contains C++ markers, parse as C++

**C++ Markers Checked:**
- `class ` declarations
- `namespace ` declarations
- `template<` usage
- `::` scope resolution
- `std::` standard library
- Access specifiers: `public:`, `private:`, `protected:`
- C++ keywords: `typename`, `constexpr`, `nullptr`

**Rationale**: Some legacy C++ codebases use `.c` extensions. The heuristic catches most cases.

#### `.h` Files
**Default**: C++
**Exception**: If file contains ONLY C constructs and no C++ markers, parse as C

**Why C++ is the default:**
- Modern C++ codebases commonly use `.h` for headers
- Header-only libraries (Boost, Eigen) use `.h`
- C projects typically use `.h` but can be distinguished by lack of C++ markers
- Lower false-positive rate when defaulting to C++

**C-only indicators:**
- No C++ keywords present
- Uses `struct` without methods
- Pure function declarations
- `typedef` instead of `using`

## Detection Algorithm

```
function detectLanguage(filepath, content):
    extension = getFileExtension(filepath)

    // Unambiguous cases
    if extension in UNAMBIGUOUS_EXTENSIONS:
        return LANGUAGE_MAP[extension]

    // Ambiguous: .c files
    if extension == ".c":
        if containsCPlusPlusMarkers(content):
            return CPP
        return C

    // Ambiguous: .h files
    if extension == ".h":
        if containsCPlusPlusMarkers(content):
            return CPP  // Default assumption
        if containsOnlyCMarkers(content):
            return C
        return CPP  // Default to C++ when unclear

    return UNKNOWN
```

## Heuristic Implementation

### C++ Marker Detection

The system scans file content for these patterns:

```go
cppMarkers := [][]byte{
    []byte("class "),
    []byte("namespace "),
    []byte("template<"),
    []byte("::"),
    []byte("std::"),
    []byte("public:"),
    []byte("private:"),
    []byte("protected:"),
    []byte("typename "),
    []byte("constexpr "),
    []byte("nullptr"),
}
```

**Performance**: Marker detection uses simple byte scanning (O(n)) and is fast enough for typical source files.

### Graceful Degradation

If Tree-sitter parsing fails:
1. Try the alternate parser (C ↔ C++)
2. Log which language was successful
3. Remember the choice for future indexing

## Configuration Overrides (Future)

Users will be able to override language detection in `.code-scout.json`:

```json
{
  "language_overrides": {
    "legacy/*.h": "c",
    "vendor/cpp-lib/*.h": "cpp",
    "*.c": "cpp"
  }
}
```

**Status**: Not yet implemented. Planned for Slice 6.

## Known Limitations

### Header Files Without Code
Pure declaration headers (`.h`) with only:
- Function prototypes
- Struct definitions
- Macro definitions

May not contain enough markers for accurate detection. Default to C++ is usually safe.

### Mixed C/C++ Projects
Projects that compile some files as C and others as C++:
- Detection is per-file, not project-wide
- May require configuration overrides (future)

### Objective-C
- `.h` files for Objective-C may be misdetected as C++
- Objective-C support not planned for Slice 4
- Future: Add Objective-C markers (`@interface`, `@implementation`)

## Testing Strategy

Each language implementation includes tests for:
1. **Unambiguous files**: Standard extension files
2. **Ambiguous files**: `.h` and `.c` files with mixed content
3. **Edge cases**: Empty files, comment-only files
4. **Real-world samples**: Code from popular open-source projects

## Future Enhancements

1. **Smarter heuristics**: Machine learning model for language detection
2. **Project-level context**: Use build files (CMakeLists.txt, Makefile) to inform detection
3. **User feedback**: Allow users to report misdetections
4. **Language statistics**: Show detection confidence scores
5. **Objective-C support**: Add markers for `.h` disambiguation

## References

- [Tree-sitter Language Support](https://tree-sitter.github.io/tree-sitter/#available-parsers)
- [GitHub Linguist](https://github.com/github-linguist/linguist) - Similar heuristic approach
- [File extension conventions](https://en.wikipedia.org/wiki/C%2B%2B#File_extensions)
