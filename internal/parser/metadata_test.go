package parser

import (
	"context"
	"testing"
)

func TestMetadataExtraction(t *testing.T) {
	testCases := []struct {
		name   string
		source string
		checks []func(*testing.T, *Chunk)
	}{
		{
			name: "package and imports metadata",
			source: `package main

import (
	"fmt"
	"strings"
	"context"
)

func hello() string {
	return "Hello, World!"
}`,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					// Check package metadata
					pkg, ok := c.Metadata["package"]
					if !ok {
						t.Error("Expected package metadata")
					}
					if pkg != "main" {
						t.Errorf("Expected package 'main', got '%s'", pkg)
					}

					// Check imports metadata
					imports, ok := c.Metadata["imports"]
					if !ok {
						t.Error("Expected imports metadata")
					}
					if imports != "fmt, strings, context" {
						t.Errorf("Expected imports 'fmt, strings, context', got '%s'", imports)
					}

					// Check language metadata
					lang, ok := c.Metadata["language"]
					if !ok {
						t.Error("Expected language metadata")
					}
					if lang != "go" {
						t.Errorf("Expected language 'go', got '%s'", lang)
					}
				},
			},
		},
		{
			name: "single import",
			source: `package parser

import "fmt"

func Print(s string) {
	fmt.Println(s)
}`,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					pkg := c.Metadata["package"]
					if pkg != "parser" {
						t.Errorf("Expected package 'parser', got '%s'", pkg)
					}

					imports := c.Metadata["imports"]
					if imports != "fmt" {
						t.Errorf("Expected imports 'fmt', got '%s'", imports)
					}
				},
			},
		},
		{
			name: "no imports",
			source: `package simple

func add(a, b int) int {
	return a + b
}`,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					pkg := c.Metadata["package"]
					if pkg != "simple" {
						t.Errorf("Expected package 'simple', got '%s'", pkg)
					}

					// Should not have imports metadata
					imports, ok := c.Metadata["imports"]
					if ok && imports != "" {
						t.Errorf("Expected no imports, got '%s'", imports)
					}

					// Should still have language
					lang := c.Metadata["language"]
					if lang != "go" {
						t.Errorf("Expected language 'go', got '%s'", lang)
					}
				},
			},
		},
		{
			name: "struct with fields metadata",
			source: `package types

type User struct {
	Name  string
	Email string
	Age   int
}`,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					if c.Type != ChunkTypeStruct {
						t.Errorf("Expected struct, got %s", c.Type)
					}

					// Check fields metadata (from type extraction)
					fields := c.Metadata["fields"]
					if fields != "Name, Email, Age" {
						t.Errorf("Expected fields 'Name, Email, Age', got '%s'", fields)
					}

					// Check package metadata
					pkg := c.Metadata["package"]
					if pkg != "types" {
						t.Errorf("Expected package 'types', got '%s'", pkg)
					}
				},
			},
		},
		{
			name: "method with receiver metadata",
			source: `package models

type Database struct {
	conn string
}

func (db *Database) Connect() error {
	return nil
}`,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					// First chunk is the struct
					if c.Type != ChunkTypeStruct {
						return
					}
				},
				func(t *testing.T, c *Chunk) {
					// Second chunk is the method
					if c.Type != ChunkTypeMethod {
						t.Errorf("Expected method, got %s", c.Type)
					}

					// Check receiver
					if c.Receiver != "*Database" {
						t.Errorf("Expected receiver '*Database', got '%s'", c.Receiver)
					}

					// Check package metadata
					pkg := c.Metadata["package"]
					if pkg != "models" {
						t.Errorf("Expected package 'models', got '%s'", pkg)
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

			extractor := NewExtractor(parser, []byte(tc.source))
			chunks, err := extractor.ExtractFunctions(context.Background())
			if err != nil {
				t.Fatalf("ExtractFunctions failed: %v", err)
			}

			if len(chunks) == 0 {
				t.Fatal("No chunks extracted")
			}

			// Run checks
			for i, check := range tc.checks {
				if i >= len(chunks) {
					t.Errorf("Check %d: Not enough chunks (have %d)", i, len(chunks))
					continue
				}
				check(t, chunks[i])
			}

			// Log all metadata for debugging
			for i, chunk := range chunks {
				t.Logf("Chunk %d (%s %s) metadata:", i, chunk.Type, chunk.Name)
				for k, v := range chunk.Metadata {
					t.Logf("  %s: %s", k, v)
				}
			}
		})
	}
}

func TestComplexFileMetadata(t *testing.T) {
	source := `package main

import (
	"context"
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

// Parser wraps Tree-sitter functionality
type Parser struct {
	parser *sitter.Parser
}

// NewParser creates a new parser
func NewParser() (*Parser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(tree_sitter_go.Language())

	if err := parser.SetLanguage(lang); err != nil {
		return nil, err
	}

	return &Parser{parser: parser}, nil
}

// Parse parses source code
func (p *Parser) Parse(ctx context.Context, source []byte) error {
	return nil
}
`

	parser, err := NewGoParser()
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	extractor := NewExtractor(parser, []byte(source))
	chunks, err := extractor.ExtractFunctions(context.Background())
	if err != nil {
		t.Fatalf("ExtractFunctions failed: %v", err)
	}

	// Should have: Parser struct, NewParser function, Parse method
	if len(chunks) != 3 {
		t.Fatalf("Expected 3 chunks, got %d", len(chunks))
	}

	// All chunks should have the same file-level metadata
	for i, chunk := range chunks {
		t.Logf("Chunk %d: %s %s", i, chunk.Type, chunk.Name)

		// Check package
		pkg := chunk.Metadata["package"]
		if pkg != "main" {
			t.Errorf("Chunk %d: expected package 'main', got '%s'", i, pkg)
		}

		// Check imports (should have all 4)
		imports := chunk.Metadata["imports"]
		expectedImports := []string{"context", "fmt", "github.com/tree-sitter/go-tree-sitter", "github.com/tree-sitter/tree-sitter-go/bindings/go"}
		for _, exp := range expectedImports {
			if !contains(imports, exp) {
				t.Errorf("Chunk %d: imports missing '%s', got '%s'", i, exp, imports)
			}
		}

		// Check language
		lang := chunk.Metadata["language"]
		if lang != "go" {
			t.Errorf("Chunk %d: expected language 'go', got '%s'", i, lang)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || containsWord(s, substr))
}

func containsWord(s, word string) bool {
	// Simple check if word appears in comma-separated list
	parts := splitByComma(s)
	for _, part := range parts {
		if part == word {
			return true
		}
	}
	return false
}

func splitByComma(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else if c != ' ' {
			current += string(c)
		} else if current != "" {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, trim(current))
	}
	return result
}

func trim(s string) string {
	start := 0
	end := len(s)

	for start < end && s[start] == ' ' {
		start++
	}
	for end > start && s[end-1] == ' ' {
		end--
	}

	return s[start:end]
}
