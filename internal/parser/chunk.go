package parser

// ChunkType represents the type of code chunk
type ChunkType string

const (
	ChunkTypeFunction  ChunkType = "function"
	ChunkTypeMethod    ChunkType = "method"
	ChunkTypeStruct    ChunkType = "struct"
	ChunkTypeInterface ChunkType = "interface"
	ChunkTypeConst     ChunkType = "const"
	ChunkTypeVar       ChunkType = "var"
	ChunkTypeClass     ChunkType = "class"
	ChunkTypeEnum      ChunkType = "enum"
	ChunkTypeImpl      ChunkType = "impl"
	ChunkTypeModule    ChunkType = "module"
)

// Chunk represents a semantic code chunk extracted from source code
type Chunk struct {
	Type       ChunkType         // Type of chunk (function, method, struct, etc.)
	Name       string            // Name of the entity (function name, type name, etc.)
	Content    string            // Full source code of the chunk including doc comments
	DocComment string            // Documentation comment (if present)
	Signature  string            // Function/method signature (if applicable)
	Receiver   string            // Method receiver type (if applicable)
	StartLine  int               // Starting line number (1-indexed)
	EndLine    int               // Ending line number (1-indexed)
	StartByte  int               // Starting byte offset
	EndByte    int               // Ending byte offset
	Metadata   map[string]string // Additional language-specific metadata
}
