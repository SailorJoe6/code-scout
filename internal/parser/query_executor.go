package parser

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// QueryExecutor executes tree-sitter queries and builds Chunks from matches
type QueryExecutor struct {
	parser     *Parser
	sourceCode []byte
	query      *sitter.Query
}

// NewQueryExecutor creates a new query executor
func NewQueryExecutor(parser *Parser, sourceCode []byte, query *sitter.Query) *QueryExecutor {
	return &QueryExecutor{
		parser:     parser,
		sourceCode: sourceCode,
		query:      query,
	}
}

// Execute runs the query against the AST and returns extracted chunks
func (qe *QueryExecutor) Execute(rootNode *sitter.Node) ([]*Chunk, error) {
	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	// Execute query
	matches := cursor.Matches(qe.query, rootNode, qe.sourceCode)

	var chunks []*Chunk
	processedNodes := make(map[uint32]bool) // Avoid duplicates based on start byte

	for match := matches.Next(); match != nil; match = matches.Next() {
		chunk := qe.processMatch(match)
		if chunk != nil {
			// Check for duplicates based on start position
			nodeID := uint32(chunk.StartByte)
			if !processedNodes[nodeID] {
				chunks = append(chunks, chunk)
				processedNodes[nodeID] = true
			}
		}
	}

	return chunks, nil
}

// processMatch converts a query match into a Chunk
func (qe *QueryExecutor) processMatch(match *sitter.QueryMatch) *Chunk {
	// Get all capture names
	captureNames := qe.query.CaptureNames()

	// Build map of captures by name
	captureMap := make(map[string]*sitter.Node)
	for _, capture := range match.Captures {
		// Get capture name from index
		if int(capture.Index) < len(captureNames) {
			captureName := captureNames[capture.Index]
			node := capture.Node
			captureMap[captureName] = &node
		}
	}

	// Determine chunk type and get definition node
	chunkType, definition := qe.determineChunkType(captureMap)
	if definition == nil {
		return nil
	}

	// Extract name
	name := qe.extractName(captureMap, chunkType)

	// Extract content
	content := definition.Utf8Text(qe.sourceCode)

	// Calculate positions
	startLine := int(definition.StartPosition().Row) + 1
	endLine := int(definition.EndPosition().Row) + 1
	startByte := int(definition.StartByte())
	endByte := int(definition.EndByte())

	// Extract metadata
	metadata := qe.extractMetadata(captureMap, chunkType)

	return &Chunk{
		Type:      chunkType,
		Name:      name,
		Content:   content,
		StartLine: startLine,
		EndLine:   endLine,
		StartByte: startByte,
		EndByte:   endByte,
		Metadata:  metadata,
	}
}

// determineChunkType determines the chunk type from captures and returns the definition node
func (qe *QueryExecutor) determineChunkType(captures map[string]*sitter.Node) (ChunkType, *sitter.Node) {
	// Check for each supported construct
	// Priority: more specific types first
	// Supports both dot notation (method.definition) and underscore notation (method_definition)

	// Methods (before functions, as methods are more specific)
	if def, ok := captures["method.definition"]; ok {
		return ChunkTypeMethod, def
	}
	if def, ok := captures["method_definition"]; ok {
		return ChunkTypeMethod, def
	}

	// Functions (various forms)
	if def, ok := captures["function.definition"]; ok {
		return ChunkTypeFunction, def
	}
	if def, ok := captures["function_definition"]; ok {
		return ChunkTypeFunction, def
	}
	if def, ok := captures["async_function.definition"]; ok {
		return ChunkTypeFunction, def
	}
	if def, ok := captures["async_function_definition"]; ok {
		return ChunkTypeFunction, def
	}
	if def, ok := captures["decorated_function.definition"]; ok {
		return ChunkTypeFunction, def
	}
	if def, ok := captures["decorated_function_definition"]; ok {
		return ChunkTypeFunction, def
	}

	// Classes
	if def, ok := captures["class.definition"]; ok {
		return ChunkTypeClass, def
	}
	if def, ok := captures["class_definition"]; ok {
		return ChunkTypeClass, def
	}
	if def, ok := captures["decorated_class.definition"]; ok {
		return ChunkTypeClass, def
	}
	if def, ok := captures["decorated_class_definition"]; ok {
		return ChunkTypeClass, def
	}

	// Interfaces
	if def, ok := captures["interface.definition"]; ok {
		return ChunkTypeInterface, def
	}
	if def, ok := captures["interface_definition"]; ok {
		return ChunkTypeInterface, def
	}

	// Structs
	if def, ok := captures["struct.definition"]; ok {
		return ChunkTypeStruct, def
	}
	if def, ok := captures["struct_definition"]; ok {
		return ChunkTypeStruct, def
	}

	// Enums
	if def, ok := captures["enum.definition"]; ok {
		return ChunkTypeEnum, def
	}
	if def, ok := captures["enum_definition"]; ok {
		return ChunkTypeEnum, def
	}

	// Traits (Rust, PHP, Scala)
	if def, ok := captures["trait.definition"]; ok {
		return ChunkTypeInterface, def
	}
	if def, ok := captures["trait_definition"]; ok {
		return ChunkTypeInterface, def
	}

	// Impls (Rust)
	if def, ok := captures["impl.definition"]; ok {
		return ChunkTypeImpl, def
	}
	if def, ok := captures["impl_definition"]; ok {
		return ChunkTypeImpl, def
	}
	if def, ok := captures["trait_impl.definition"]; ok {
		return ChunkTypeImpl, def
	}
	if def, ok := captures["trait_impl_definition"]; ok {
		return ChunkTypeImpl, def
	}

	// Modules
	if def, ok := captures["module.definition"]; ok {
		return ChunkTypeModule, def
	}
	if def, ok := captures["module_definition"]; ok {
		return ChunkTypeModule, def
	}

	// Namespaces (C++, PHP)
	if def, ok := captures["namespace.definition"]; ok {
		return ChunkTypeModule, def
	}
	if def, ok := captures["namespace_definition"]; ok {
		return ChunkTypeModule, def
	}

	// Type aliases
	if def, ok := captures["type_alias.definition"]; ok {
		return ChunkTypeType, def
	}
	if def, ok := captures["type_alias_definition"]; ok {
		return ChunkTypeType, def
	}

	// No recognized type found
	return ChunkTypeFunction, nil
}

// extractName extracts the name from captures
func (qe *QueryExecutor) extractName(captures map[string]*sitter.Node, chunkType ChunkType) string {
	// Try type-specific name patterns first (both dot and underscore notation)
	typePrefix := chunkTypeToPrefix(chunkType)
	if typePrefix != "" {
		if nameNode, ok := captures[typePrefix+".name"]; ok {
			return nameNode.Utf8Text(qe.sourceCode)
		}
		if nameNode, ok := captures[typePrefix+"_name"]; ok {
			return nameNode.Utf8Text(qe.sourceCode)
		}
	}

	// Try common name patterns (both notations)
	namePatterns := []string{
		"function.name", "function_name",
		"async_function.name", "async_function_name",
		"decorated_function.name", "decorated_function_name",
		"method.name", "method_name",
		"class.name", "class_name",
		"decorated_class.name", "decorated_class_name",
		"interface.name", "interface_name",
		"struct.name", "struct_name",
		"enum.name", "enum_name",
		"trait.name", "trait_name",
		"impl.name", "impl_name",
		"module.name", "module_name",
		"namespace.name", "namespace_name",
		"type_alias.name", "type_alias_name",
	}

	for _, pattern := range namePatterns {
		if nameNode, ok := captures[pattern]; ok {
			return nameNode.Utf8Text(qe.sourceCode)
		}
	}

	return ""
}

// extractMetadata extracts metadata from captures
func (qe *QueryExecutor) extractMetadata(captures map[string]*sitter.Node, chunkType ChunkType) map[string]string {
	metadata := make(map[string]string)

	// Extract parameters (both notations)
	paramPatterns := []string{
		"function.parameters", "function_parameters",
		"async_function.parameters", "async_function_parameters",
		"decorated_function.parameters", "decorated_function_parameters",
		"method.parameters", "method_parameters",
	}
	for _, pattern := range paramPatterns {
		if params, ok := captures[pattern]; ok {
			metadata["parameters"] = params.Utf8Text(qe.sourceCode)
			break
		}
	}

	// Extract decorators (Python, Java)
	if decorator, ok := captures["decorator"]; ok {
		metadata["decorator"] = decorator.Utf8Text(qe.sourceCode)
	}

	// Extract async marker
	if _, ok := captures["function.async"]; ok {
		metadata["async"] = "true"
	}

	// Extract visibility modifiers (Java, C++, PHP)
	if visibility, ok := captures["visibility"]; ok {
		metadata["visibility"] = visibility.Utf8Text(qe.sourceCode)
	}

	// Extract receiver (Go methods)
	if receiver, ok := captures["method.receiver"]; ok {
		metadata["receiver"] = receiver.Utf8Text(qe.sourceCode)
	}

	// Extract return type (if available)
	if result, ok := captures["function.result"]; ok {
		metadata["return_type"] = result.Utf8Text(qe.sourceCode)
	} else if result, ok := captures["method.result"]; ok {
		metadata["return_type"] = result.Utf8Text(qe.sourceCode)
	}

	return metadata
}

// chunkTypeToPrefix converts a ChunkType to its capture name prefix
func chunkTypeToPrefix(ct ChunkType) string {
	switch ct {
	case ChunkTypeFunction:
		return "function"
	case ChunkTypeMethod:
		return "method"
	case ChunkTypeClass:
		return "class"
	case ChunkTypeInterface:
		return "interface"
	case ChunkTypeStruct:
		return "struct"
	case ChunkTypeEnum:
		return "enum"
	case ChunkTypeImpl:
		return "impl"
	case ChunkTypeModule:
		return "module"
	case ChunkTypeType:
		return "type_alias"
	default:
		return ""
	}
}

// SplitCapturesByDefinition groups captures by their definition nodes
// This is useful for handling nested constructs (e.g., multiple methods in a class)
func (qe *QueryExecutor) SplitCapturesByDefinition(captures map[string]*sitter.Node) []map[string]*sitter.Node {
	// This is a placeholder for future enhancement to handle nested constructs
	// For now, we return a single group
	return []map[string]*sitter.Node{captures}
}
