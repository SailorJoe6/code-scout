package parser

import (
	"context"
	"testing"
)

func TestExtractFunctions(t *testing.T) {
	testCases := []struct {
		name          string
		sourceCode    string
		expectedCount int
		checks        []func(*testing.T, *Chunk)
	}{
		{
			name: "simple function",
			sourceCode: `package main

func hello() string {
	return "Hello, World!"
}`,
			expectedCount: 1,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					if c.Type != ChunkTypeFunction {
						t.Errorf("Expected type %s, got %s", ChunkTypeFunction, c.Type)
					}
					if c.Name != "hello" {
						t.Errorf("Expected name 'hello', got '%s'", c.Name)
					}
					if c.Signature != "() string" {
						t.Errorf("Expected signature '() string', got '%s'", c.Signature)
					}
					if c.StartLine != 3 {
						t.Errorf("Expected start line 3, got %d", c.StartLine)
					}
					if c.EndLine != 5 {
						t.Errorf("Expected end line 5, got %d", c.EndLine)
					}
				},
			},
		},
		{
			name: "function with doc comment",
			sourceCode: `package main

// Greet returns a greeting message for the given name
func Greet(name string) string {
	return "Hello, " + name
}`,
			expectedCount: 1,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					if c.Name != "Greet" {
						t.Errorf("Expected name 'Greet', got '%s'", c.Name)
					}
					if c.Signature != "(name string) string" {
						t.Errorf("Expected signature '(name string) string', got '%s'", c.Signature)
					}
					// Doc comment should be extracted
					if c.DocComment == "" {
						t.Error("Expected doc comment, got empty string")
					}
				},
			},
		},
		{
			name: "multiple functions",
			sourceCode: `package main

func add(a, b int) int {
	return a + b
}

func subtract(a, b int) int {
	return a - b
}

func multiply(a, b int) int {
	return a * b
}`,
			expectedCount: 3,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					if c.Name != "add" {
						t.Errorf("Expected first function 'add', got '%s'", c.Name)
					}
				},
				func(t *testing.T, c *Chunk) {
					if c.Name != "subtract" {
						t.Errorf("Expected second function 'subtract', got '%s'", c.Name)
					}
				},
				func(t *testing.T, c *Chunk) {
					if c.Name != "multiply" {
						t.Errorf("Expected third function 'multiply', got '%s'", c.Name)
					}
				},
			},
		},
		{
			name: "method with receiver",
			sourceCode: `package main

type User struct {
	Name string
}

// GetName returns the user's name
func (u *User) GetName() string {
	return u.Name
}

// SetName sets the user's name
func (u *User) SetName(name string) {
	u.Name = name
}`,
			expectedCount: 3, // User struct + 2 methods
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					if c.Type != ChunkTypeStruct {
						t.Errorf("Expected type %s, got %s", ChunkTypeStruct, c.Type)
					}
					if c.Name != "User" {
						t.Errorf("Expected name 'User', got '%s'", c.Name)
					}
				},
				func(t *testing.T, c *Chunk) {
					if c.Type != ChunkTypeMethod {
						t.Errorf("Expected type %s, got %s", ChunkTypeMethod, c.Type)
					}
					if c.Name != "GetName" {
						t.Errorf("Expected name 'GetName', got '%s'", c.Name)
					}
					if c.Receiver != "*User" {
						t.Errorf("Expected receiver '*User', got '%s'", c.Receiver)
					}
					if c.Signature != "() string" {
						t.Errorf("Expected signature '() string', got '%s'", c.Signature)
					}
				},
				func(t *testing.T, c *Chunk) {
					if c.Name != "SetName" {
						t.Errorf("Expected name 'SetName', got '%s'", c.Name)
					}
					if c.Receiver != "*User" {
						t.Errorf("Expected receiver '*User', got '%s'", c.Receiver)
					}
				},
			},
		},
		{
			name: "function with multiple return values",
			sourceCode: `package main

func divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}`,
			expectedCount: 1,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					if c.Name != "divide" {
						t.Errorf("Expected name 'divide', got '%s'", c.Name)
					}
					if c.Signature != "(a, b int) (int, error)" {
						t.Errorf("Expected signature '(a, b int) (int, error)', got '%s'", c.Signature)
					}
				},
			},
		},
		{
			name: "variadic function",
			sourceCode: `package main

func sum(numbers ...int) int {
	total := 0
	for _, n := range numbers {
		total += n
	}
	return total
}`,
			expectedCount: 1,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					if c.Name != "sum" {
						t.Errorf("Expected name 'sum', got '%s'", c.Name)
					}
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser, err := NewGoParser()
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			extractor := NewExtractor(parser, []byte(tc.sourceCode))
			chunks, err := extractor.ExtractFunctions(context.Background())
			if err != nil {
				t.Fatalf("ExtractFunctions failed: %v", err)
			}

			if len(chunks) != tc.expectedCount {
				t.Fatalf("Expected %d chunks, got %d", tc.expectedCount, len(chunks))
			}

			// Run custom checks for each chunk
			for i, check := range tc.checks {
				if i >= len(chunks) {
					t.Errorf("Check %d: Not enough chunks (have %d)", i, len(chunks))
					continue
				}
				check(t, chunks[i])
			}

			// Log chunk details for debugging
			for i, chunk := range chunks {
				t.Logf("Chunk %d: type=%s, name=%s, signature=%s, receiver=%s, lines=%d-%d",
					i, chunk.Type, chunk.Name, chunk.Signature, chunk.Receiver,
					chunk.StartLine, chunk.EndLine)
			}
		})
	}
}

func TestExtractFromRealGoFile(t *testing.T) {
	// Use a real Go file from this project
	sourceCode := `package parser

import (
	"context"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

// Parser wraps Tree-sitter functionality for parsing source code
type Parser struct {
	parser *sitter.Parser
}

// NewGoParser creates a new parser configured for Go source code
func NewGoParser() (*Parser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(tree_sitter_go.Language())

	if err := parser.SetLanguage(lang); err != nil {
		return nil, err
	}

	return &Parser{parser: parser}, nil
}

// Parse parses Go source code and returns the syntax tree
func (p *Parser) Parse(ctx context.Context, sourceCode []byte) (*sitter.Tree, error) {
	tree := p.parser.Parse(sourceCode, nil)
	if tree == nil {
		return nil, nil
	}
	return tree, nil
}

// GetRootNode returns the root node of a parsed tree
func (p *Parser) GetRootNode(tree *sitter.Tree) *sitter.Node {
	return tree.RootNode()
}
`

	parser, err := NewGoParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	extractor := NewExtractor(parser, []byte(sourceCode))
	chunks, err := extractor.ExtractFunctions(context.Background())
	if err != nil {
		t.Fatalf("ExtractFunctions failed: %v", err)
	}

	// Should extract: Parser struct, NewGoParser (function), Parse (method), GetRootNode (method)
	expectedChunks := []struct {
		name      string
		chunkType ChunkType
	}{
		{"Parser", ChunkTypeStruct},
		{"NewGoParser", ChunkTypeFunction},
		{"Parse", ChunkTypeMethod},
		{"GetRootNode", ChunkTypeMethod},
	}

	if len(chunks) != len(expectedChunks) {
		t.Fatalf("Expected %d chunks, got %d", len(expectedChunks), len(chunks))
	}

	for i, expected := range expectedChunks {
		if chunks[i].Name != expected.name {
			t.Errorf("Expected chunk %d to be '%s', got '%s'", i, expected.name, chunks[i].Name)
		}
		if chunks[i].Type != expected.chunkType {
			t.Errorf("Expected chunk %d type to be '%s', got '%s'", i, expected.chunkType, chunks[i].Type)
		}
		t.Logf("Extracted: %s (type=%s, lines=%d-%d)",
			chunks[i].Name, chunks[i].Type, chunks[i].StartLine, chunks[i].EndLine)
	}
}
