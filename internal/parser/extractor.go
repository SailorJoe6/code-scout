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

// ExtractFunctions extracts all function and method declarations from Go source code
func (e *Extractor) ExtractFunctions(ctx context.Context) ([]*Chunk, error) {
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

	// Extract file-level metadata first
	e.extractFileMetadata(rootNode)

	var chunks []*Chunk

	// Walk the tree and find function and method declarations
	cursor := rootNode.Walk()
	defer cursor.Close()

	e.walkNode(rootNode, &chunks)

	// Enrich all chunks with file-level metadata
	e.enrichChunksWithMetadata(chunks)

	return chunks, nil
}

// walkNode recursively walks the AST and extracts function/method chunks
func (e *Extractor) walkNode(node *sitter.Node, chunks *[]*Chunk) {
	if node == nil {
		return
	}

	nodeKind := node.Kind()

	// Go-specific nodes
	if nodeKind == "function_declaration" {
		chunk := e.extractFunction(node)
		if chunk != nil {
			*chunks = append(*chunks, chunk)
		}
	}

	if nodeKind == "method_declaration" {
		chunk := e.extractMethod(node)
		if chunk != nil {
			*chunks = append(*chunks, chunk)
		}
	}

	if nodeKind == "type_declaration" {
		typeChunks := e.extractTypes(node)
		*chunks = append(*chunks, typeChunks...)
	}

	// Python-specific nodes
	if nodeKind == "function_definition" || nodeKind == "class_definition" {
		chunk := e.extractGenericNode(node, nodeKind)
		if chunk != nil {
			*chunks = append(*chunks, chunk)
		}
	}

	// JavaScript/TypeScript nodes
	if nodeKind == "function" || nodeKind == "arrow_function" ||
	   nodeKind == "class_declaration" || nodeKind == "method_definition" {
		chunk := e.extractGenericNode(node, nodeKind)
		if chunk != nil {
			*chunks = append(*chunks, chunk)
		}
	}

	// Java nodes
	if nodeKind == "class_declaration" || nodeKind == "interface_declaration" ||
	   nodeKind == "method_declaration" || nodeKind == "constructor_declaration" {
		// Only process if not already handled by Go
		if e.parser.Language() != LanguageGo {
			chunk := e.extractGenericNode(node, nodeKind)
			if chunk != nil {
				*chunks = append(*chunks, chunk)
			}
		}
	}

	// Rust nodes
	if nodeKind == "function_item" || nodeKind == "struct_item" ||
	   nodeKind == "enum_item" || nodeKind == "trait_item" || nodeKind == "impl_item" {
		chunk := e.extractGenericNode(node, nodeKind)
		if chunk != nil {
			*chunks = append(*chunks, chunk)
		}
	}

	// C/C++ nodes
	if nodeKind == "function_definition" || nodeKind == "class_specifier" ||
	   nodeKind == "struct_specifier" || nodeKind == "enum_specifier" {
		// Avoid duplicates with Python
		if e.parser.Language() == LanguageC || e.parser.Language() == LanguageCPP {
			chunk := e.extractGenericNode(node, nodeKind)
			if chunk != nil {
				*chunks = append(*chunks, chunk)
			}
		}
	}

	// Ruby nodes
	if nodeKind == "method" || nodeKind == "class" || nodeKind == "module" {
		chunk := e.extractGenericNode(node, nodeKind)
		if chunk != nil {
			*chunks = append(*chunks, chunk)
		}
	}

	// PHP nodes
	if nodeKind == "function_definition" || nodeKind == "class_declaration" ||
	   nodeKind == "interface_declaration" || nodeKind == "trait_declaration" {
		// Only process for PHP
		if e.parser.Language() == LanguagePHP {
			chunk := e.extractGenericNode(node, nodeKind)
			if chunk != nil {
				*chunks = append(*chunks, chunk)
			}
		}
	}

	// Scala nodes
	if nodeKind == "function_definition" || nodeKind == "class_definition" ||
	   nodeKind == "object_definition" || nodeKind == "trait_definition" {
		// Only process for Scala
		if e.parser.Language() == LanguageScala {
			chunk := e.extractGenericNode(node, nodeKind)
			if chunk != nil {
				*chunks = append(*chunks, chunk)
			}
		}
	}

	// Recursively walk children
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		e.walkNode(child, chunks)
	}
}

// extractFunction extracts a function declaration chunk
func (e *Extractor) extractFunction(node *sitter.Node) *Chunk {
	if node == nil {
		return nil
	}

	// Get function name
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := nameNode.Utf8Text(e.sourceCode)

	// Get the full function text including doc comments
	startByte := node.StartByte()
	endByte := node.EndByte()

	// Try to find preceding comment
	docComment := e.findDocComment(node)
	if docComment != "" {
		// Adjust start byte to include doc comment
		// We'll refine this later
	}

	content := string(e.sourceCode[startByte:endByte])

	// Get signature (parameters and return type)
	signature := e.extractFunctionSignature(node)

	// Calculate line numbers (1-indexed)
	startLine := int(node.StartPosition().Row) + 1
	endLine := int(node.EndPosition().Row) + 1

	return &Chunk{
		Type:       ChunkTypeFunction,
		Name:       name,
		Content:    content,
		DocComment: docComment,
		Signature:  signature,
		StartLine:  startLine,
		EndLine:    endLine,
		StartByte:  int(startByte),
		EndByte:    int(endByte),
		Metadata:   make(map[string]string),
	}
}

// extractMethod extracts a method declaration chunk
func (e *Extractor) extractMethod(node *sitter.Node) *Chunk {
	if node == nil {
		return nil
	}

	// Get method name
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := nameNode.Utf8Text(e.sourceCode)

	// Get receiver type
	receiverNode := node.ChildByFieldName("receiver")
	receiver := ""
	if receiverNode != nil {
		receiver = e.extractReceiver(receiverNode)
	}

	// Get the full method text
	startByte := node.StartByte()
	endByte := node.EndByte()

	// Try to find preceding comment
	docComment := e.findDocComment(node)

	content := string(e.sourceCode[startByte:endByte])

	// Get signature
	signature := e.extractFunctionSignature(node)

	// Calculate line numbers (1-indexed)
	startLine := int(node.StartPosition().Row) + 1
	endLine := int(node.EndPosition().Row) + 1

	return &Chunk{
		Type:       ChunkTypeMethod,
		Name:       name,
		Content:    content,
		DocComment: docComment,
		Signature:  signature,
		Receiver:   receiver,
		StartLine:  startLine,
		EndLine:    endLine,
		StartByte:  int(startByte),
		EndByte:    int(endByte),
		Metadata:   make(map[string]string),
	}
}

// extractFunctionSignature extracts the function/method signature
func (e *Extractor) extractFunctionSignature(node *sitter.Node) string {
	if node == nil {
		return ""
	}

	// Get parameters
	paramsNode := node.ChildByFieldName("parameters")
	params := ""
	if paramsNode != nil {
		params = paramsNode.Utf8Text(e.sourceCode)
	}

	// Get result (return type)
	resultNode := node.ChildByFieldName("result")
	result := ""
	if resultNode != nil {
		result = " " + resultNode.Utf8Text(e.sourceCode)
	}

	return params + result
}

// extractReceiver extracts the receiver type from a method
func (e *Extractor) extractReceiver(receiverNode *sitter.Node) string {
	if receiverNode == nil {
		return ""
	}

	// The receiver is a parameter_list containing a parameter_declaration
	// Example: (r *Receiver) or (r Receiver)
	text := receiverNode.Utf8Text(e.sourceCode)

	// Clean up the text - remove parentheses and extract just the type
	text = strings.TrimPrefix(text, "(")
	text = strings.TrimSuffix(text, ")")
	text = strings.TrimSpace(text)

	// Split on space to get "name type" or "name *type"
	parts := strings.Fields(text)
	if len(parts) >= 2 {
		return parts[1] // Return the type part
	}
	if len(parts) == 1 {
		return parts[0]
	}

	return text
}

// extractTypes extracts type declarations (struct, interface, etc.)
func (e *Extractor) extractTypes(node *sitter.Node) []*Chunk {
	if node == nil {
		return nil
	}

	var chunks []*Chunk

	// A type_declaration can contain multiple type specs
	// type (
	//   Foo struct { ... }
	//   Bar interface { ... }
	// )
	// OR a single type spec:
	// type Foo struct { ... }

	// Find type_spec children
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child.Kind() == "type_spec" {
			chunk := e.extractTypeSpec(child, node)
			if chunk != nil {
				chunks = append(chunks, chunk)
			}
		}
	}

	return chunks
}

// extractTypeSpec extracts a single type specification
func (e *Extractor) extractTypeSpec(typeSpecNode, typeDeclarationNode *sitter.Node) *Chunk {
	if typeSpecNode == nil {
		return nil
	}

	// Get type name
	nameNode := typeSpecNode.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := nameNode.Utf8Text(e.sourceCode)

	// Get type definition (struct_type, interface_type, etc.)
	typeNode := typeSpecNode.ChildByFieldName("type")
	if typeNode == nil {
		return nil
	}

	typeKind := typeNode.Kind()

	// Only extract struct and interface types for now
	if typeKind != "struct_type" && typeKind != "interface_type" {
		return nil
	}

	// Determine chunk type
	var chunkType ChunkType
	if typeKind == "struct_type" {
		chunkType = ChunkTypeStruct
	} else if typeKind == "interface_type" {
		chunkType = ChunkTypeInterface
	}

	// Get the full type declaration text
	// Use the type_declaration node to include doc comments
	startByte := typeDeclarationNode.StartByte()
	endByte := typeDeclarationNode.EndByte()

	// Try to find preceding comment
	docComment := e.findDocComment(typeDeclarationNode)

	content := string(e.sourceCode[startByte:endByte])

	// Calculate line numbers (1-indexed)
	startLine := int(typeDeclarationNode.StartPosition().Row) + 1
	endLine := int(typeDeclarationNode.EndPosition().Row) + 1

	// Extract fields or methods
	fields := e.extractFields(typeNode)

	chunk := &Chunk{
		Type:       chunkType,
		Name:       name,
		Content:    content,
		DocComment: docComment,
		StartLine:  startLine,
		EndLine:    endLine,
		StartByte:  int(startByte),
		EndByte:    int(endByte),
		Metadata:   make(map[string]string),
	}

	// Store fields in metadata
	if len(fields) > 0 {
		chunk.Metadata["fields"] = strings.Join(fields, ", ")
	}

	return chunk
}

// extractFields extracts field names from a struct or method signatures from an interface
func (e *Extractor) extractFields(typeNode *sitter.Node) []string {
	if typeNode == nil {
		return nil
	}

	var fields []string

	if typeNode.Kind() == "struct_type" {
		// Extract struct fields - look for field_declaration_list child
		childCount := typeNode.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := typeNode.Child(i)
			if child.Kind() == "field_declaration_list" {
				// Now iterate through field_declaration nodes
				fieldCount := child.ChildCount()
				for j := uint(0); j < fieldCount; j++ {
					fieldNode := child.Child(j)
					if fieldNode.Kind() == "field_declaration" {
						fieldName := e.extractFieldName(fieldNode)
						if fieldName != "" {
							fields = append(fields, fieldName)
						}
					}
				}
				break
			}
		}
	} else if typeNode.Kind() == "interface_type" {
		// Extract interface methods - iterate directly through children
		// Interface methods are represented as method_elem nodes
		childCount := typeNode.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := typeNode.Child(i)
			if child.Kind() == "method_elem" {
				methodName := e.extractMethodSpecName(child)
				if methodName != "" {
					fields = append(fields, methodName)
				}
			}
		}
	}

	return fields
}

// extractFieldName extracts the field name from a field_declaration
func (e *Extractor) extractFieldName(fieldNode *sitter.Node) string {
	if fieldNode == nil {
		return ""
	}

	nameNode := fieldNode.ChildByFieldName("name")
	if nameNode != nil {
		return nameNode.Utf8Text(e.sourceCode)
	}

	return ""
}

// extractMethodSpecName extracts the method name from a method_spec (interface method)
func (e *Extractor) extractMethodSpecName(methodNode *sitter.Node) string {
	if methodNode == nil {
		return ""
	}

	nameNode := methodNode.ChildByFieldName("name")
	if nameNode != nil {
		return nameNode.Utf8Text(e.sourceCode)
	}

	return ""
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

// findDocComment finds the documentation comment preceding a node
func (e *Extractor) findDocComment(node *sitter.Node) string {
	if node == nil {
		return ""
	}

	parent := node.Parent()
	if parent == nil {
		return ""
	}

	// Look for a comment node immediately before this node
	// In Go, doc comments are typically comment nodes that precede the declaration
	prevSibling := node.PrevSibling()
	if prevSibling != nil && prevSibling.Kind() == "comment" {
		comment := prevSibling.Utf8Text(e.sourceCode)
		// Remove leading // or /* */ markers
		comment = strings.TrimPrefix(comment, "//")
		comment = strings.TrimPrefix(comment, "/*")
		comment = strings.TrimSuffix(comment, "*/")
		return strings.TrimSpace(comment)
	}

	return ""
}

// extractGenericNode extracts a generic node for non-Go languages
// This is a simplified extractor that works across different languages
func (e *Extractor) extractGenericNode(node *sitter.Node, nodeKind string) *Chunk {
	if node == nil {
		return nil
	}

	// Try to extract name from common field names
	var name string
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		name = nameNode.Utf8Text(e.sourceCode)
	}

	// If no name field, try to find identifier in first few children
	if name == "" {
		for i := uint(0); i < node.ChildCount() && i < 5; i++ {
			child := node.Child(i)
			if child != nil && child.Kind() == "identifier" {
				name = child.Utf8Text(e.sourceCode)
				break
			}
		}
	}

	// Get the full node text
	startByte := node.StartByte()
	endByte := node.EndByte()
	content := string(e.sourceCode[startByte:endByte])

	// Calculate line numbers (1-indexed)
	startLine := int(node.StartPosition().Row) + 1
	endLine := int(node.EndPosition().Row) + 1

	// Map node kind to chunk type
	chunkType := e.mapNodeKindToChunkType(nodeKind)

	return &Chunk{
		Type:       chunkType,
		Name:       name,
		Content:    content,
		StartLine:  startLine,
		EndLine:    endLine,
		StartByte:  int(startByte),
		EndByte:    int(endByte),
		Metadata:   make(map[string]string),
	}
}

// mapNodeKindToChunkType maps Tree-sitter node kinds to chunk types
func (e *Extractor) mapNodeKindToChunkType(nodeKind string) ChunkType {
	switch nodeKind {
	case "function_definition", "function_declaration", "function_item", "function":
		return ChunkTypeFunction
	case "method_declaration", "method_definition", "method":
		return ChunkTypeMethod
	case "class_definition", "class_declaration", "class_specifier", "class":
		return ChunkTypeClass
	case "struct_item", "struct_specifier":
		return ChunkTypeStruct
	case "enum_item", "enum_specifier", "enum_declaration":
		return ChunkTypeEnum
	case "interface_declaration", "trait_item", "trait_declaration":
		return ChunkTypeInterface
	case "impl_item":
		return ChunkTypeImpl
	case "module", "namespace_definition":
		return ChunkTypeModule
	case "arrow_function":
		return ChunkTypeFunction
	case "object_definition":
		return ChunkTypeClass // Scala objects are similar to classes
	default:
		return ChunkTypeFunction // Default fallback
	}
}
