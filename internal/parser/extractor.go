package parser

import (
	"context"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Extractor extracts semantic chunks from parsed source code
type Extractor struct {
	parser      *Parser
	sourceCode  []byte
	imports     []string // Cached imports for the file
	packageName string   // Cached package name
}

// NewExtractor creates a new extractor for the given parser and source code
func NewExtractor(parser *Parser, sourceCode []byte) *Extractor {
	return &Extractor{
		parser:     parser,
		sourceCode: sourceCode,
	}
}

// ExtractFunctions extracts all function and method declarations from source code
// Uses query-based extraction for all supported languages, with fallback to generic extraction
func (e *Extractor) ExtractFunctions(ctx context.Context) ([]*Chunk, error) {
	// Try query-based extraction first (preferred method)
	return e.extractWithQuery(ctx)
}

// walkNode recursively walks the AST
// This is only used as a fallback when query-based extraction fails
// It performs minimal extraction to ensure we return something rather than nothing
func (e *Extractor) walkNode(node *sitter.Node, chunks *[]*Chunk) {
	if node == nil {
		return
	}

	// Recursively walk children
	// In fallback mode, we don't extract anything - query-based extraction is required
	// This function exists only to maintain compatibility with fallback code path
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		e.walkNode(child, chunks)
	}
}


// extractFileMetadata extracts file-level metadata like package name and imports
func (e *Extractor) extractFileMetadata(rootNode *sitter.Node) {
	if rootNode == nil {
		return
	}

	childCount := rootNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := rootNode.Child(i)
		if child == nil {
			continue
		}

		// Extract package name
		if child.Kind() == "package_clause" {
			e.packageName = e.extractPackageName(child)
		}

		// Extract imports
		if child.Kind() == "import_declaration" {
			imports := e.extractImports(child)
			e.imports = append(e.imports, imports...)
		}
	}
}

// extractPackageName extracts the package name from a package_clause node
func (e *Extractor) extractPackageName(packageNode *sitter.Node) string {
	if packageNode == nil {
		return ""
	}

	// Look for package_identifier child
	childCount := packageNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := packageNode.Child(i)
		if child.Kind() == "package_identifier" {
			return child.Utf8Text(e.sourceCode)
		}
	}

	return ""
}

// extractImports extracts import paths from an import_declaration node
func (e *Extractor) extractImports(importNode *sitter.Node) []string {
	if importNode == nil {
		return nil
	}

	var imports []string

	// Import declarations can be:
	// import "fmt"
	// import (
	//   "fmt"
	//   "strings"
	// )

	childCount := importNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := importNode.Child(i)
		if child == nil {
			continue
		}

		// Single import spec
		if child.Kind() == "import_spec" {
			importPath := e.extractImportPath(child)
			if importPath != "" {
				imports = append(imports, importPath)
			}
		}

		// Import spec list (multiple imports)
		if child.Kind() == "import_spec_list" {
			specCount := child.ChildCount()
			for j := uint(0); j < specCount; j++ {
				spec := child.Child(j)
				if spec != nil && spec.Kind() == "import_spec" {
					importPath := e.extractImportPath(spec)
					if importPath != "" {
						imports = append(imports, importPath)
					}
				}
			}
		}
	}

	return imports
}

// extractImportPath extracts the import path from an import_spec node
func (e *Extractor) extractImportPath(importSpec *sitter.Node) string {
	if importSpec == nil {
		return ""
	}

	pathNode := importSpec.ChildByFieldName("path")
	if pathNode != nil {
		path := pathNode.Utf8Text(e.sourceCode)
		// Remove quotes
		path = strings.Trim(path, "\"")
		return path
	}

	return ""
}

// enrichChunksWithMetadata adds file-level metadata to all chunks
func (e *Extractor) enrichChunksWithMetadata(chunks []*Chunk) {
	for _, chunk := range chunks {
		if chunk.Metadata == nil {
			chunk.Metadata = make(map[string]string)
		}

		// Add package name
		if e.packageName != "" {
			chunk.Metadata["package"] = e.packageName
		}

		// Add imports
		if len(e.imports) > 0 {
			chunk.Metadata["imports"] = strings.Join(e.imports, ", ")
		}

		// Add language
		chunk.Metadata["language"] = "go"
	}
}



// extractWithQuery uses tree-sitter queries for semantic extraction
// This is the preferred extraction method for all languages
func (e *Extractor) extractWithQuery(ctx context.Context) ([]*Chunk, error) {
	tree, err := e.parser.Parse(ctx, e.sourceCode)
	if err != nil {
		return nil, err
	}
	if tree == nil {
		return nil, nil
	}

	rootNode := e.parser.GetRootNode(tree)
	if rootNode == nil {
		return nil, nil
	}

	// Load query for this language
	query, err := globalQueryCache.LoadQuery(e.parser.Language(), e.parser.TSLanguage())
	if err != nil {
		// Query loading failed - fall back to generic extraction
		// This can happen if:
		// - Query file is missing
		// - Query has syntax errors
		// - Language not supported
		return e.extractGenericFallback(ctx, rootNode)
	}

	// Execute query
	executor := NewQueryExecutor(e.parser, e.sourceCode, query)
	chunks, err := executor.Execute(rootNode)
	if err != nil {
		// Query execution failed - fall back to generic extraction
		return e.extractGenericFallback(ctx, rootNode)
	}

	// If no chunks found, try fallback
	if len(chunks) == 0 {
		return e.extractGenericFallback(ctx, rootNode)
	}

	// Extract file-level metadata first
	e.extractFileMetadata(rootNode)

	// Enrich chunks with file metadata
	e.enrichChunksWithMetadata(chunks)

	return chunks, nil
}

// extractGenericFallback falls back to generic extraction when queries fail
// This uses the existing walkNode approach
func (e *Extractor) extractGenericFallback(ctx context.Context, rootNode *sitter.Node) ([]*Chunk, error) {
	var chunks []*Chunk

	// Extract file-level metadata first
	e.extractFileMetadata(rootNode)

	// Walk the tree
	e.walkNode(rootNode, &chunks)

	// Enrich chunks with file metadata
	e.enrichChunksWithMetadata(chunks)

	return chunks, nil
}
