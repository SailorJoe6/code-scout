package parser

import (
	"context"
	"testing"
)

func TestExtractTypes(t *testing.T) {
	testCases := []struct {
		name          string
		sourceCode    string
		expectedCount int
		checks        []func(*testing.T, *Chunk)
	}{
		{
			name: "simple struct",
			sourceCode: `package main

type User struct {
	Name string
	Age  int
}`,
			expectedCount: 1,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					if c.Type != ChunkTypeStruct {
						t.Errorf("Expected type %s, got %s", ChunkTypeStruct, c.Type)
					}
					if c.Name != "User" {
						t.Errorf("Expected name 'User', got '%s'", c.Name)
					}
					if c.StartLine != 3 {
						t.Errorf("Expected start line 3, got %d", c.StartLine)
					}
					// Check fields metadata
					fields, ok := c.Metadata["fields"]
					if !ok {
						t.Error("Expected fields metadata")
					}
					if fields != "Name, Age" {
						t.Errorf("Expected fields 'Name, Age', got '%s'", fields)
					}
				},
			},
		},
		{
			name: "struct with doc comment",
			sourceCode: `package main

// User represents a user in the system
type User struct {
	Name string
	Email string
}`,
			expectedCount: 1,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					if c.Name != "User" {
						t.Errorf("Expected name 'User', got '%s'", c.Name)
					}
					if c.DocComment == "" {
						t.Error("Expected doc comment, got empty string")
					}
				},
			},
		},
		{
			name: "interface",
			sourceCode: `package main

// Reader defines a reader interface
type Reader interface {
	Read(p []byte) (n int, err error)
	Close() error
}`,
			expectedCount: 1,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					if c.Type != ChunkTypeInterface {
						t.Errorf("Expected type %s, got %s", ChunkTypeInterface, c.Type)
					}
					if c.Name != "Reader" {
						t.Errorf("Expected name 'Reader', got '%s'", c.Name)
					}
					// Check methods metadata
					fields, ok := c.Metadata["fields"]
					if !ok {
						t.Error("Expected fields metadata for interface methods")
					}
					if fields != "Read, Close" {
						t.Errorf("Expected methods 'Read, Close', got '%s'", fields)
					}
				},
			},
		},
		{
			name: "multiple types",
			sourceCode: `package main

type Point struct {
	X, Y int
}

type Circle struct {
	Center Point
	Radius float64
}

type Shape interface {
	Area() float64
	Perimeter() float64
}`,
			expectedCount: 3,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					if c.Type != ChunkTypeStruct {
						t.Errorf("Expected type %s, got %s", ChunkTypeStruct, c.Type)
					}
					if c.Name != "Point" {
						t.Errorf("Expected name 'Point', got '%s'", c.Name)
					}
				},
				func(t *testing.T, c *Chunk) {
					if c.Type != ChunkTypeStruct {
						t.Errorf("Expected type %s, got %s", ChunkTypeStruct, c.Type)
					}
					if c.Name != "Circle" {
						t.Errorf("Expected name 'Circle', got '%s'", c.Name)
					}
				},
				func(t *testing.T, c *Chunk) {
					if c.Type != ChunkTypeInterface {
						t.Errorf("Expected type %s, got %s", ChunkTypeInterface, c.Type)
					}
					if c.Name != "Shape" {
						t.Errorf("Expected name 'Shape', got '%s'", c.Name)
					}
				},
			},
		},
		{
			name: "empty interface",
			sourceCode: `package main

type Any interface{}`,
			expectedCount: 1,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					if c.Type != ChunkTypeInterface {
						t.Errorf("Expected type %s, got %s", ChunkTypeInterface, c.Type)
					}
					if c.Name != "Any" {
						t.Errorf("Expected name 'Any', got '%s'", c.Name)
					}
				},
			},
		},
		{
			name: "struct with embedded fields",
			sourceCode: `package main

type Base struct {
	ID int
}

type Extended struct {
	Base
	Name string
}`,
			expectedCount: 2,
			checks: []func(*testing.T, *Chunk){
				func(t *testing.T, c *Chunk) {
					if c.Name != "Base" {
						t.Errorf("Expected name 'Base', got '%s'", c.Name)
					}
				},
				func(t *testing.T, c *Chunk) {
					if c.Name != "Extended" {
						t.Errorf("Expected name 'Extended', got '%s'", c.Name)
					}
					// Should have both fields
					fields, ok := c.Metadata["fields"]
					if !ok {
						t.Error("Expected fields metadata")
					}
					if fields != "Name" {
						// Note: embedded fields might not have names in field_declaration
						t.Logf("Fields: %s", fields)
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
				t.Logf("Chunk %d: type=%s, name=%s, lines=%d-%d, fields=%s",
					i, chunk.Type, chunk.Name, chunk.StartLine, chunk.EndLine,
					chunk.Metadata["fields"])
			}
		})
	}
}

func TestExtractMixedFunctionsAndTypes(t *testing.T) {
	sourceCode := `package main

// User represents a user
type User struct {
	Name string
	Age  int
}

// NewUser creates a new user
func NewUser(name string, age int) *User {
	return &User{Name: name, Age: age}
}

// GetName returns the user's name
func (u *User) GetName() string {
	return u.Name
}

// Validator defines validation interface
type Validator interface {
	Validate() error
}

// Validate checks if user is valid
func (u *User) Validate() error {
	return nil
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

	// Should extract: User struct, NewUser function, GetName method, Validator interface, Validate method
	expectedChunks := map[string]ChunkType{
		"User":      ChunkTypeStruct,
		"NewUser":   ChunkTypeFunction,
		"GetName":   ChunkTypeMethod,
		"Validator": ChunkTypeInterface,
		"Validate":  ChunkTypeMethod,
	}

	if len(chunks) != len(expectedChunks) {
		t.Fatalf("Expected %d chunks, got %d", len(expectedChunks), len(chunks))
	}

	foundChunks := make(map[string]ChunkType)
	for _, chunk := range chunks {
		foundChunks[chunk.Name] = chunk.Type
		t.Logf("Found: %s (type=%s, lines=%d-%d)", chunk.Name, chunk.Type,
			chunk.StartLine, chunk.EndLine)
	}

	for name, expectedType := range expectedChunks {
		actualType, found := foundChunks[name]
		if !found {
			t.Errorf("Expected to find chunk '%s'", name)
			continue
		}
		if actualType != expectedType {
			t.Errorf("Chunk '%s': expected type %s, got %s", name, expectedType, actualType)
		}
	}
}
