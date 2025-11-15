package parser

import (
	"context"
	"testing"
)

func TestNewGoParser(t *testing.T) {
	parser, err := NewGoParser()
	if err != nil {
		t.Fatalf("Failed to create Go parser: %v", err)
	}
	if parser == nil {
		t.Fatal("Parser is nil")
	}
}

func TestParseGoCode(t *testing.T) {
	parser, err := NewGoParser()
	if err != nil {
		t.Fatalf("Failed to create Go parser: %v", err)
	}

	testCases := []struct {
		name       string
		sourceCode string
		wantError  bool
	}{
		{
			name: "simple function",
			sourceCode: `package main

func hello() string {
	return "Hello, World!"
}`,
			wantError: false,
		},
		{
			name: "function with doc comment",
			sourceCode: `package main

// Greet returns a greeting message
func Greet(name string) string {
	return "Hello, " + name
}`,
			wantError: false,
		},
		{
			name: "struct with methods",
			sourceCode: `package main

// User represents a user in the system
type User struct {
	Name string
	Age  int
}

// GetName returns the user's name
func (u *User) GetName() string {
	return u.Name
}`,
			wantError: false,
		},
		{
			name: "interface definition",
			sourceCode: `package main

// Reader defines a reader interface
type Reader interface {
	Read(p []byte) (n int, err error)
}`,
			wantError: false,
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tree, err := parser.Parse(ctx, []byte(tc.sourceCode))
			if tc.wantError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tc.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tree == nil {
				t.Error("Tree is nil")
				return
			}

			rootNode := parser.GetRootNode(tree)
			if rootNode == nil {
				t.Error("Root node is nil")
				return
			}

			// Verify we got a valid parse tree
			nodeType := rootNode.Kind()
			if nodeType != "source_file" {
				t.Errorf("Expected root node type 'source_file', got '%s'", nodeType)
			}

			// Verify there are child nodes
			childCount := int(rootNode.ChildCount())
			if childCount == 0 {
				t.Error("Expected child nodes, got 0")
			}

			t.Logf("Parsed successfully: %d child nodes, type=%s", childCount, nodeType)
		})
	}
}

func TestParseRealGoFile(t *testing.T) {
	parser, err := NewGoParser()
	if err != nil {
		t.Fatalf("Failed to create Go parser: %v", err)
	}

	// Use the parser's own source code as test input
	sourceCode := `package parser

import (
	"context"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

// Parser wraps Tree-sitter functionality
type Parser struct {
	parser *sitter.Parser
}

// NewGoParser creates a new parser
func NewGoParser() (*Parser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(tree_sitter_go.Language())

	if err := parser.SetLanguage(lang); err != nil {
		return nil, err
	}

	return &Parser{parser: parser}, nil
}
`

	ctx := context.Background()
	tree, err := parser.Parse(ctx, []byte(sourceCode))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if tree == nil {
		t.Fatal("Tree is nil")
	}

	rootNode := parser.GetRootNode(tree)
	if rootNode == nil {
		t.Fatal("Root node is nil")
	}

	t.Logf("Successfully parsed real Go code: %d child nodes", rootNode.ChildCount())
}
