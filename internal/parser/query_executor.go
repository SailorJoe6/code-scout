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

	// Extract signature (for functions and methods)
	signature := qe.extractSignature(captureMap, chunkType)

	// Extract receiver (for methods)
	receiver := qe.extractReceiver(captureMap, chunkType)

	// Extract doc comment
	docComment := qe.extractDocComment(definition)

	return &Chunk{
		Type:       chunkType,
		Name:       name,
		Content:    content,
		DocComment: docComment,
		Signature:  signature,
		Receiver:   receiver,
		StartLine:  startLine,
		EndLine:    endLine,
		StartByte:  startByte,
		EndByte:    endByte,
		Metadata:   metadata,
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

	// Extract receiver (Go methods) - store full text in metadata too
	if receiver, ok := captures["method.receiver"]; ok {
		metadata["receiver"] = receiver.Utf8Text(qe.sourceCode)
	}

	// Extract return type (if available)
	if result, ok := captures["function.result"]; ok {
		metadata["return_type"] = result.Utf8Text(qe.sourceCode)
	} else if result, ok := captures["method.result"]; ok {
		metadata["return_type"] = result.Utf8Text(qe.sourceCode)
	}

	// Extract struct fields (Go)
	if fields, ok := captures["struct.fields"]; ok {
		fieldsText := fields.Utf8Text(qe.sourceCode)
		metadata["fields"] = qe.extractFieldNames(fieldsText)
	} else if fields, ok := captures["struct_fields"]; ok {
		fieldsText := fields.Utf8Text(qe.sourceCode)
		metadata["fields"] = qe.extractFieldNames(fieldsText)
	}

	// Extract interface methods (Go)
	if body, ok := captures["interface.body"]; ok {
		bodyText := body.Utf8Text(qe.sourceCode)
		metadata["fields"] = qe.extractMethodNames(bodyText)
	} else if body, ok := captures["interface_body"]; ok {
		bodyText := body.Utf8Text(qe.sourceCode)
		metadata["fields"] = qe.extractMethodNames(bodyText)
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

// extractSignature builds a signature string from parameters and result captures
func (qe *QueryExecutor) extractSignature(captures map[string]*sitter.Node, chunkType ChunkType) string {
	if chunkType != ChunkTypeFunction && chunkType != ChunkTypeMethod {
		return ""
	}

	var params, result string

	// Extract parameters (try both notations)
	paramPatterns := []string{
		"function.parameters", "function_parameters",
		"async_function.parameters", "async_function_parameters",
		"decorated_function.parameters", "decorated_function_parameters",
		"method.parameters", "method_parameters",
	}
	for _, pattern := range paramPatterns {
		if node, ok := captures[pattern]; ok {
			params = node.Utf8Text(qe.sourceCode)
			break
		}
	}

	// Extract result/return type (try both notations)
	resultPatterns := []string{
		"function.result", "function_result",
		"async_function.result", "async_function_result",
		"decorated_function.result", "decorated_function_result",
		"method.result", "method_result",
	}
	for _, pattern := range resultPatterns {
		if node, ok := captures[pattern]; ok {
			result = node.Utf8Text(qe.sourceCode)
			break
		}
	}

	// Build signature
	if result != "" {
		return params + " " + result
	}
	return params
}

// extractReceiver extracts the receiver from method captures
func (qe *QueryExecutor) extractReceiver(captures map[string]*sitter.Node, chunkType ChunkType) string {
	if chunkType != ChunkTypeMethod {
		return ""
	}

	// Try both notations
	receiverPatterns := []string{"method.receiver", "method_receiver"}
	for _, pattern := range receiverPatterns {
		if node, ok := captures[pattern]; ok {
			receiverText := node.Utf8Text(qe.sourceCode)
			// Clean up receiver: convert "(r *Receiver)" to "*Receiver"
			// Remove parentheses and variable name
			return cleanReceiver(receiverText)
		}
	}

	return ""
}

// cleanReceiver cleans up receiver text from "(r *Receiver)" to "*Receiver"
func cleanReceiver(receiver string) string {
	// Remove leading/trailing whitespace
	receiver = trimSpace(receiver)

	// Remove outer parentheses if present
	if len(receiver) > 0 && receiver[0] == '(' && receiver[len(receiver)-1] == ')' {
		receiver = receiver[1 : len(receiver)-1]
		receiver = trimSpace(receiver)
	}

	// Find the first space and take everything after it
	// This removes the variable name (e.g., "r *Receiver" -> "*Receiver")
	for i, ch := range receiver {
		if ch == ' ' || ch == '\t' {
			return trimSpace(receiver[i+1:])
		}
	}

	// No space found, return as-is
	return receiver
}

// trimSpace removes leading and trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)

	// Trim leading whitespace
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// Trim trailing whitespace
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

// extractDocComment extracts doc comments that appear before the definition node
func (qe *QueryExecutor) extractDocComment(definition *sitter.Node) string {
	// Look for comment nodes that appear immediately before this definition
	// This is language-specific, so we'll use a simple heuristic:
	// Search backwards from the definition's start position to find comments

	defStartByte := int(definition.StartByte())
	if defStartByte == 0 {
		return ""
	}

	// Search backwards up to 1000 bytes for comment blocks
	searchStart := defStartByte - 1000
	if searchStart < 0 {
		searchStart = 0
	}

	// Extract the text before the definition
	precedingText := string(qe.sourceCode[searchStart:defStartByte])

	// Find doc comments using simple pattern matching
	// This is a simplified approach - ideally we'd use the AST comment nodes
	return extractDocCommentFromText(precedingText)
}

// extractDocCommentFromText extracts doc comments from preceding text
func extractDocCommentFromText(text string) string {
	// Look for Go-style doc comments (//) or block comments (/* */)
	// This is a simplified implementation

	// Reverse search for comment patterns
	lines := splitLines(text)
	var commentLines []string

	// Search backwards from the end
	for i := len(lines) - 1; i >= 0; i-- {
		line := trimSpace(lines[i])

		// Skip empty lines
		if line == "" {
			if len(commentLines) > 0 {
				// Empty line after comments - stop
				break
			}
			continue
		}

		// Check for // comments
		if len(line) >= 2 && line[0] == '/' && line[1] == '/' {
			// Remove // prefix
			comment := trimSpace(line[2:])
			commentLines = append([]string{comment}, commentLines...)
			continue
		}

		// Check for /* */ block comments
		if len(line) >= 2 && line[0] == '/' && line[1] == '*' {
			// Extract content between /* and */
			endIdx := len(line)
			if endIdx >= 2 && line[endIdx-2] == '*' && line[endIdx-1] == '/' {
				comment := trimSpace(line[2 : endIdx-2])
				commentLines = append([]string{comment}, commentLines...)
			}
			continue
		}

		// Non-comment line found - stop
		if len(commentLines) > 0 {
			break
		}
	}

	// Join comment lines
	if len(commentLines) == 0 {
		return ""
	}

	result := ""
	for _, line := range commentLines {
		if result != "" {
			result += "\n"
		}
		result += line
	}
	return result
}

// splitLines splits text into lines
func splitLines(text string) []string {
	var lines []string
	start := 0

	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}

	// Add the last line if there's any remaining text
	if start < len(text) {
		lines = append(lines, text[start:])
	}

	return lines
}

// extractFieldNames extracts field names from struct field declaration list
func (qe *QueryExecutor) extractFieldNames(fieldsText string) string {
	// Parse field names from Go struct field declarations
	// Format: "{\n\tName string\n\tAge int\n}" -> "Name, Age"

	var names []string
	lines := splitLines(fieldsText)

	for _, line := range lines {
		line = trimSpace(line)
		if line == "" || line == "{" || line == "}" {
			continue
		}

		// Find the first identifier (field name)
		// Field format: "Name string" or "Name, Age int"
		parts := splitOnWhitespace(line)
		if len(parts) > 0 {
			// First part is the field name
			fieldName := parts[0]
			// Check if it contains comma (multiple fields)
			if containsChar(fieldName, ',') {
				// Split by comma
				multiNames := splitOnComma(fieldName)
				for _, name := range multiNames {
					name = trimSpace(name)
					if name != "" {
						names = append(names, name)
					}
				}
			} else {
				names = append(names, fieldName)
			}
		}
	}

	// Join names with ", "
	result := ""
	for i, name := range names {
		if i > 0 {
			result += ", "
		}
		result += name
	}
	return result
}

// extractMethodNames extracts method names from interface body
func (qe *QueryExecutor) extractMethodNames(bodyText string) string {
	// Parse method names from Go interface
	// Format: "{\n\tRead() error\n\tClose() error\n}" -> "Read, Close"

	var names []string
	lines := splitLines(bodyText)

	for _, line := range lines {
		line = trimSpace(line)
		if line == "" || line == "{" || line == "}" {
			continue
		}

		// Find method name (identifier before parentheses)
		// Method format: "Read() error" or "Read(p []byte) (int, error)"
		parenIdx := -1
		for i, ch := range line {
			if ch == '(' {
				parenIdx = i
				break
			}
		}

		if parenIdx > 0 {
			methodName := trimSpace(line[:parenIdx])
			if methodName != "" {
				names = append(names, methodName)
			}
		}
	}

	// Join names with ", "
	result := ""
	for i, name := range names {
		if i > 0 {
			result += ", "
		}
		result += name
	}
	return result
}

// splitOnWhitespace splits a string on whitespace characters
func splitOnWhitespace(s string) []string {
	var parts []string
	start := 0
	inWord := false

	for i, ch := range s {
		isSpace := ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'

		if !isSpace && !inWord {
			start = i
			inWord = true
		} else if isSpace && inWord {
			parts = append(parts, s[start:i])
			inWord = false
		}
	}

	if inWord {
		parts = append(parts, s[start:])
	}

	return parts
}

// splitOnComma splits a string on comma characters
func splitOnComma(s string) []string {
	var parts []string
	start := 0

	for i, ch := range s {
		if ch == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}

	if start < len(s) {
		parts = append(parts, s[start:])
	}

	return parts
}

// containsChar checks if a string contains a character
func containsChar(s string, ch rune) bool {
	for _, c := range s {
		if c == ch {
			return true
		}
	}
	return false
}
