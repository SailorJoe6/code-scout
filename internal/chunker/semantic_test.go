package chunker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSemanticChunker(t *testing.T) {
	// Create a temporary Go file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	sourceCode := `package main

import "fmt"

// HelloWorld prints a greeting
func HelloWorld() {
	fmt.Println("Hello, World!")
}

// User represents a user
type User struct {
	Name string
	Age  int
}

// GetName returns the user's name
func (u *User) GetName() string {
	return u.Name
}
`

	err := os.WriteFile(testFile, []byte(sourceCode), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create semantic chunker
	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	// Chunk the file
	chunks, err := chunker.ChunkFile(testFile, "go")
	if err != nil {
		t.Fatalf("Failed to chunk file: %v", err)
	}

	// Should have 3 chunks: HelloWorld function, User struct, GetName method
	if len(chunks) != 3 {
		t.Fatalf("Expected 3 chunks, got %d", len(chunks))
	}

	// Verify first chunk (HelloWorld function)
	if chunks[0].ChunkType != "function" {
		t.Errorf("Chunk 0: expected type 'function', got '%s'", chunks[0].ChunkType)
	}
	if chunks[0].Name != "HelloWorld" {
		t.Errorf("Chunk 0: expected name 'HelloWorld', got '%s'", chunks[0].Name)
	}
	if chunks[0].LineStart != 6 {
		t.Errorf("Chunk 0: expected start line 6, got %d", chunks[0].LineStart)
	}
	if chunks[0].FilePath != testFile {
		t.Errorf("Chunk 0: file path mismatch")
	}
	if chunks[0].Language != "go" {
		t.Errorf("Chunk 0: expected language 'go', got '%s'", chunks[0].Language)
	}

	// Verify metadata
	if chunks[0].Metadata["package"] != "main" {
		t.Errorf("Chunk 0: expected package 'main', got '%s'", chunks[0].Metadata["package"])
	}
	if chunks[0].Metadata["imports"] != "fmt" {
		t.Errorf("Chunk 0: expected imports 'fmt', got '%s'", chunks[0].Metadata["imports"])
	}
	if chunks[0].Metadata["language"] != "go" {
		t.Errorf("Chunk 0: expected metadata language 'go', got '%s'", chunks[0].Metadata["language"])
	}

	// Verify second chunk (User struct)
	if chunks[1].ChunkType != "struct" {
		t.Errorf("Chunk 1: expected type 'struct', got '%s'", chunks[1].ChunkType)
	}
	if chunks[1].Name != "User" {
		t.Errorf("Chunk 1: expected name 'User', got '%s'", chunks[1].Name)
	}

	// Verify third chunk (GetName method)
	if chunks[2].ChunkType != "method" {
		t.Errorf("Chunk 2: expected type 'method', got '%s'", chunks[2].ChunkType)
	}
	if chunks[2].Name != "GetName" {
		t.Errorf("Chunk 2: expected name 'GetName', got '%s'", chunks[2].Name)
	}
	if chunks[2].Metadata["receiver"] != "*User" {
		t.Errorf("Chunk 2: expected receiver '*User', got '%s'", chunks[2].Metadata["receiver"])
	}

	// Log all chunks for debugging
	for i, chunk := range chunks {
		t.Logf("Chunk %d: %s %s (lines %d-%d)", i, chunk.ChunkType, chunk.Name,
			chunk.LineStart, chunk.LineEnd)
		t.Logf("  Metadata: package=%s, imports=%s", chunk.Metadata["package"], chunk.Metadata["imports"])
	}
}

func TestSemanticChunkerWithMultipleImports(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	sourceCode := `package parser

import (
	"context"
	"fmt"
	"strings"
)

func Parse(ctx context.Context, input string) error {
	parts := strings.Split(input, ",")
	fmt.Println(parts)
	return nil
}
`

	err := os.WriteFile(testFile, []byte(sourceCode), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	chunks, err := chunker.ChunkFile(testFile, "go")
	if err != nil {
		t.Fatalf("Failed to chunk file: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(chunks))
	}

	chunk := chunks[0]

	// Verify imports
	imports := chunk.Metadata["imports"]
	expectedImports := []string{"context", "fmt", "strings"}
	for _, exp := range expectedImports {
		if !contains(imports, exp) {
			t.Errorf("Expected imports to contain '%s', got '%s'", exp, imports)
		}
	}

	// Verify package
	if chunk.Metadata["package"] != "parser" {
		t.Errorf("Expected package 'parser', got '%s'", chunk.Metadata["package"])
	}
}

func TestSemanticChunkerPythonFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	pythonCode := `def greet(name):
    """Greet someone by name."""
    return f"Hello, {name}!"

class Person:
    """A person class."""
    def __init__(self, name):
        self.name = name

    def say_hello(self):
        return greet(self.name)
`

	err := os.WriteFile(testFile, []byte(pythonCode), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	chunks, err := chunker.ChunkFile(testFile, "python")
	if err != nil {
		t.Fatalf("Expected Python support, got error: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk for Python file")
	}

	// Verify chunks have correct language
	for _, chunk := range chunks {
		if chunk.Language != "python" {
			t.Errorf("Expected language 'python', got '%s'", chunk.Language)
		}
		if chunk.EmbeddingType != "code" {
			t.Errorf("Expected embedding_type 'code', got '%s'", chunk.EmbeddingType)
		}
	}
}

func contains(s, substr string) bool {
	if s == "" || substr == "" {
		return false
	}
	// Simple substring check
	return s == substr || hasWord(s, substr)
}

func hasWord(s, word string) bool {
	// Check if word appears in comma-separated list
	parts := split(s, ',')
	for _, part := range parts {
		if trimSpace(part) == word {
			return true
		}
	}
	return false
}

func split(s string, sep rune) []string {
	var result []string
	var current string
	for _, c := range s {
		if c == sep {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func trimSpace(s string) string {
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

// TestSemanticChunkerMarkdown tests markdown file chunking
func TestSemanticChunkerMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	markdownContent := `# Main Title

This is the introduction.

## Section 1

Some content in section 1.

## Section 2

Some content in section 2.
`

	err := os.WriteFile(testFile, []byte(markdownContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	chunks, err := chunker.ChunkFile(testFile, "markdown")
	if err != nil {
		t.Fatalf("Failed to chunk markdown file: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk for markdown file")
	}

	// Verify all chunks have embedding_type = "docs"
	for _, chunk := range chunks {
		if chunk.EmbeddingType != "docs" {
			t.Errorf("Expected embedding_type 'docs', got '%s'", chunk.EmbeddingType)
		}
		if chunk.Language != "markdown" {
			t.Errorf("Expected language 'markdown', got '%s'", chunk.Language)
		}
	}
}

// TestSemanticChunkerPlainText tests plain text file chunking
func TestSemanticChunkerPlainText(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	textContent := `This is a plain text file.
It contains multiple lines.
But no special formatting.`

	err := os.WriteFile(testFile, []byte(textContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	chunks, err := chunker.ChunkFile(testFile, "text")
	if err != nil {
		t.Fatalf("Failed to chunk text file: %v", err)
	}

	// Plain text should be one chunk
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for plain text, got %d", len(chunks))
	}

	if len(chunks) > 0 {
		chunk := chunks[0]
		if chunk.EmbeddingType != "docs" {
			t.Errorf("Expected embedding_type 'docs', got '%s'", chunk.EmbeddingType)
		}
		if chunk.Language != "text" {
			t.Errorf("Expected language 'text', got '%s'", chunk.Language)
		}
		if chunk.ChunkType != "document" {
			t.Errorf("Expected chunk_type 'document', got '%s'", chunk.ChunkType)
		}
		if chunk.Code != textContent {
			t.Error("Expected chunk content to match file content")
		}
	}
}

// TestSemanticChunkerRST tests reStructuredText file chunking
func TestSemanticChunkerRST(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.rst")

	rstContent := `Documentation Title
===================

This is a reStructuredText file.

Subsection
----------

More content here.`

	err := os.WriteFile(testFile, []byte(rstContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	chunks, err := chunker.ChunkFile(testFile, "rst")
	if err != nil {
		t.Fatalf("Failed to chunk RST file: %v", err)
	}

	// RST should be one chunk (treated like plain text)
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for RST, got %d", len(chunks))
	}

	if len(chunks) > 0 {
		chunk := chunks[0]
		if chunk.EmbeddingType != "docs" {
			t.Errorf("Expected embedding_type 'docs', got '%s'", chunk.EmbeddingType)
		}
		if chunk.Language != "rst" {
			t.Errorf("Expected language 'rst', got '%s'", chunk.Language)
		}
		if chunk.ChunkType != "document" {
			t.Errorf("Expected chunk_type 'document', got '%s'", chunk.ChunkType)
		}
	}
}

// TestSemanticChunkerUnsupportedLanguage tests error handling for unsupported languages
func TestSemanticChunkerUnsupportedLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.xyz")

	err := os.WriteFile(testFile, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	_, err = chunker.ChunkFile(testFile, "unsupported")
	if err == nil {
		t.Error("Expected error for unsupported language")
	}
}

// TestSemanticChunkerNonExistentFile tests error handling for missing files
func TestSemanticChunkerNonExistentFile(t *testing.T) {
	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	_, err = chunker.ChunkFile("/nonexistent/file.txt", "text")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestSemanticChunkerCodeEmbeddingType tests that code files get "code" embedding type
func TestSemanticChunkerCodeEmbeddingType(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	sourceCode := `package main

func main() {
	println("test")
}
`

	err := os.WriteFile(testFile, []byte(sourceCode), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	chunks, err := chunker.ChunkFile(testFile, "go")
	if err != nil {
		t.Fatalf("Failed to chunk file: %v", err)
	}

	// Verify all code chunks have embedding_type = "code"
	for _, chunk := range chunks {
		if chunk.EmbeddingType != "code" {
			t.Errorf("Expected embedding_type 'code' for Go file, got '%s'", chunk.EmbeddingType)
		}
	}
}

// TestSemanticChunkerNonExistentCodeFile tests error handling for missing code files
func TestSemanticChunkerNonExistentCodeFile(t *testing.T) {
	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	_, err = chunker.ChunkFile("/nonexistent/file.go", "go")
	if err == nil {
		t.Error("Expected error for non-existent code file")
	}
}

// TestSemanticChunkerMetadataPopulation tests that all metadata is properly set
func TestSemanticChunkerMetadataPopulation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	sourceCode := `package mypackage

// MyFunc does something with parameters
func MyFunc(a int, b string) bool {
	return true
}

type MyType struct {
	Field int
}

// Method has a receiver and a signature
func (m *MyType) Method(x int) string {
	return "result"
}
`

	err := os.WriteFile(testFile, []byte(sourceCode), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	chunks, err := chunker.ChunkFile(testFile, "go")
	if err != nil {
		t.Fatalf("Failed to chunk file: %v", err)
	}

	// Find the method chunk and verify receiver is set
	foundMethod := false
	for _, chunk := range chunks {
		if chunk.ChunkType == "method" {
			foundMethod = true
			if chunk.Metadata["receiver"] == "" {
				t.Error("Expected receiver metadata for method")
			}
			if chunk.Metadata["signature"] == "" {
				t.Error("Expected signature metadata for method")
			}
		}
		// All code chunks should have metadata
		if chunk.Metadata == nil {
			t.Errorf("Chunk %s has nil metadata", chunk.Name)
		}
	}

	if !foundMethod {
		t.Error("Expected to find at least one method chunk")
	}
}
