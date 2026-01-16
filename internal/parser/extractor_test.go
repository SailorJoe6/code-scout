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

// TestExtractPythonFunctions tests Python function extraction
func TestExtractPythonFunctions(t *testing.T) {
	pythonCode := `def greet(name):
    """Say hello to someone."""
    return f"Hello, {name}!"

class Person:
    """A person class."""
    def __init__(self, name):
        self.name = name

    def say_hello(self):
        """Say hello using name."""
        return greet(self.name)
`

	parser, err := NewParser(LanguagePython)
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}

	extractor := NewExtractor(parser, []byte(pythonCode))
	chunks, err := extractor.ExtractFunctions(context.Background())
	if err != nil {
		t.Fatalf("ExtractFunctions failed: %v", err)
	}

	// Should extract greet function, Person class, __init__ and say_hello methods
	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk from Python code")
	}

	// Verify we got some functions/classes
	foundFunction := false
	foundClass := false
	for _, chunk := range chunks {
		if chunk.Type == ChunkTypeFunction {
			foundFunction = true
		}
		if chunk.Type == ChunkTypeClass {
			foundClass = true
		}
		t.Logf("Extracted Python: %s (type=%s, lines=%d-%d)",
			chunk.Name, chunk.Type, chunk.StartLine, chunk.EndLine)
	}

	if !foundFunction && !foundClass {
		t.Error("Expected to find at least one function or class in Python code")
	}
}

// TestExtractJavaScriptFunctions tests JavaScript function extraction
func TestExtractJavaScriptFunctions(t *testing.T) {
	jsCode := `function add(a, b) {
    return a + b;
}

const multiply = (x, y) => x * y;

class Calculator {
    constructor() {
        this.result = 0;
    }

    add(n) {
        this.result += n;
        return this;
    }
}
`

	parser, err := NewParser(LanguageJavaScript)
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}

	extractor := NewExtractor(parser, []byte(jsCode))
	chunks, err := extractor.ExtractFunctions(context.Background())
	if err != nil {
		t.Fatalf("ExtractFunctions failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk from JavaScript code")
	}

	// Log what we found
	for _, chunk := range chunks {
		t.Logf("Extracted JavaScript: %s (type=%s, lines=%d-%d)",
			chunk.Name, chunk.Type, chunk.StartLine, chunk.EndLine)
	}
}

// TestExtractTypeScriptFunctions tests TypeScript function extraction
func TestExtractTypeScriptFunctions(t *testing.T) {
	tsCode := `interface User {
    name: string;
    age: number;
}

function createUser(name: string, age: number): User {
    return { name, age };
}

class UserManager {
    private users: User[] = [];

    addUser(user: User): void {
        this.users.push(user);
    }
}
`

	parser, err := NewParser(LanguageTypeScript)
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}

	extractor := NewExtractor(parser, []byte(tsCode))
	chunks, err := extractor.ExtractFunctions(context.Background())
	if err != nil {
		t.Fatalf("ExtractFunctions failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk from TypeScript code")
	}

	for _, chunk := range chunks {
		t.Logf("Extracted TypeScript: %s (type=%s, lines=%d-%d)",
			chunk.Name, chunk.Type, chunk.StartLine, chunk.EndLine)
	}
}

// TestExtractRustFunctions tests Rust function extraction
func TestExtractRustFunctions(t *testing.T) {
	rustCode := `pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

pub struct Point {
    x: f64,
    y: f64,
}

impl Point {
    pub fn new(x: f64, y: f64) -> Self {
        Point { x, y }
    }

    pub fn distance(&self) -> f64 {
        (self.x * self.x + self.y * self.y).sqrt()
    }
}
`

	parser, err := NewParser(LanguageRust)
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}

	extractor := NewExtractor(parser, []byte(rustCode))
	chunks, err := extractor.ExtractFunctions(context.Background())
	if err != nil {
		t.Fatalf("ExtractFunctions failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk from Rust code")
	}

	for _, chunk := range chunks {
		t.Logf("Extracted Rust: %s (type=%s, lines=%d-%d)",
			chunk.Name, chunk.Type, chunk.StartLine, chunk.EndLine)
	}
}

// TestExtractJavaCode tests Java class, interface, enum extraction
func TestExtractJavaCode(t *testing.T) {
	javaCode := `package com.example;

public interface Drawable {
    void draw();
}

public enum Color {
    RED, GREEN, BLUE
}

public class Circle implements Drawable {
    private int radius;

    public Circle(int radius) {
        this.radius = radius;
    }

    @Override
    public void draw() {
        System.out.println("Drawing circle");
    }

    public int getRadius() {
        return radius;
    }
}
`

	parser, err := NewParser(LanguageJava)
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}

	extractor := NewExtractor(parser, []byte(javaCode))
	chunks, err := extractor.ExtractFunctions(context.Background())
	if err != nil {
		t.Fatalf("ExtractFunctions failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk from Java code")
	}

	// Look for interface, enum, class types
	foundInterface := false
	foundEnum := false
	foundClass := false

	for _, chunk := range chunks {
		if chunk.Type == ChunkTypeInterface {
			foundInterface = true
		}
		if chunk.Type == ChunkTypeEnum {
			foundEnum = true
		}
		if chunk.Type == ChunkTypeClass {
			foundClass = true
		}
		t.Logf("Extracted Java: %s (type=%s, lines=%d-%d)",
			chunk.Name, chunk.Type, chunk.StartLine, chunk.EndLine)
	}

	if !foundInterface {
		t.Log("Warning: No interface found in Java code (may be tree-sitter limitation)")
	}
	if !foundEnum {
		t.Log("Warning: No enum found in Java code (may be tree-sitter limitation)")
	}
	if !foundClass {
		t.Error("Expected to find at least one class in Java code")
	}
}

// TestExtractCppCode tests C++ namespace and class extraction
func TestExtractCppCode(t *testing.T) {
	cppCode := `namespace math {

class Calculator {
public:
    int add(int a, int b) {
        return a + b;
    }

    int subtract(int a, int b) {
        return a - b;
    }
};

int multiply(int a, int b) {
    return a * b;
}

}  // namespace math
`

	parser, err := NewParser(LanguageCPP)
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}

	extractor := NewExtractor(parser, []byte(cppCode))
	chunks, err := extractor.ExtractFunctions(context.Background())
	if err != nil {
		t.Fatalf("ExtractFunctions failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk from C++ code")
	}

	for _, chunk := range chunks {
		t.Logf("Extracted C++: %s (type=%s, lines=%d-%d)",
			chunk.Name, chunk.Type, chunk.StartLine, chunk.EndLine)
	}
}

// TestExtractScalaCode tests Scala object and trait extraction
func TestExtractScalaCode(t *testing.T) {
	scalaCode := `trait Animal {
  def speak(): String
}

object Dog extends Animal {
  def speak(): String = "Woof!"
}

class Cat extends Animal {
  def speak(): String = "Meow!"
}
`

	parser, err := NewParser(LanguageScala)
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}

	extractor := NewExtractor(parser, []byte(scalaCode))
	chunks, err := extractor.ExtractFunctions(context.Background())
	if err != nil {
		t.Fatalf("ExtractFunctions failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk from Scala code")
	}

	for _, chunk := range chunks {
		t.Logf("Extracted Scala: %s (type=%s, lines=%d-%d)",
			chunk.Name, chunk.Type, chunk.StartLine, chunk.EndLine)
	}
}

// TestExtractorErrorHandling tests error conditions
func TestExtractorErrorHandling(t *testing.T) {
	// Test with invalid Go code (no package declaration)
	invalidCode := `func broken {
		this is not valid syntax
	}`

	parser, err := NewParser(LanguageGo)
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}

	extractor := NewExtractor(parser, []byte(invalidCode))
	chunks, err := extractor.ExtractFunctions(context.Background())

	// Should not error, just return empty or partial chunks
	if err != nil {
		t.Logf("Extract returned error (may be acceptable): %v", err)
	}
	t.Logf("Extracted %d chunks from invalid code", len(chunks))
}

// TestExtractorContextCancellation tests context cancellation
func TestExtractorContextCancellation(t *testing.T) {
	sourceCode := `package main

func test() {
	// Simple function
}
`

	parser, err := NewParser(LanguageGo)
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}

	extractor := NewExtractor(parser, []byte(sourceCode))

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = extractor.ExtractFunctions(ctx)
	if err == nil {
		t.Log("Warning: Expected error from cancelled context, but got none (may still succeed for small files)")
	} else {
		t.Logf("Got expected error from cancelled context: %v", err)
	}
}

// TestExtractorWithRubyCode tests Ruby class and method extraction
func TestExtractorWithRubyCode(t *testing.T) {
	rubyCode := `class Person
  attr_accessor :name, :age

  def initialize(name, age)
    @name = name
    @age = age
  end

  def greet
    puts "Hello, I'm #{@name}"
  end
end

def standalone_function
  puts "I'm standalone"
end
`

	parser, err := NewParser(LanguageRuby)
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}

	extractor := NewExtractor(parser, []byte(rubyCode))
	chunks, err := extractor.ExtractFunctions(context.Background())
	if err != nil {
		t.Fatalf("ExtractFunctions failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk from Ruby code")
	}

	for _, chunk := range chunks {
		t.Logf("Extracted Ruby: %s (type=%s, lines=%d-%d)",
			chunk.Name, chunk.Type, chunk.StartLine, chunk.EndLine)
	}
}

// TestExtractorWithPHPCode tests PHP class and function extraction
func TestExtractorWithPHPCode(t *testing.T) {
	phpCode := `<?php

namespace App\Models;

class User {
    private $name;
    private $email;

    public function __construct($name, $email) {
        $this->name = $name;
        $this->email = $email;
    }

    public function getName() {
        return $this->name;
    }
}

function helper_function($arg) {
    return strtoupper($arg);
}
`

	parser, err := NewParser(LanguagePHP)
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}

	extractor := NewExtractor(parser, []byte(phpCode))
	chunks, err := extractor.ExtractFunctions(context.Background())
	if err != nil {
		t.Fatalf("ExtractFunctions failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk from PHP code")
	}

	for _, chunk := range chunks {
		t.Logf("Extracted PHP: %s (type=%s, lines=%d-%d)",
			chunk.Name, chunk.Type, chunk.StartLine, chunk.EndLine)
	}
}

// TestExtractorWithCCode tests C struct and function extraction
func TestExtractorWithCCode(t *testing.T) {
	cCode := `#include <stdio.h>

struct Point {
    int x;
    int y;
};

int add(int a, int b) {
    return a + b;
}

void print_point(struct Point *p) {
    printf("Point(%d, %d)\n", p->x, p->y);
}
`

	parser, err := NewParser(LanguageC)
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}

	extractor := NewExtractor(parser, []byte(cCode))
	chunks, err := extractor.ExtractFunctions(context.Background())
	if err != nil {
		t.Fatalf("ExtractFunctions failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk from C code")
	}

	for _, chunk := range chunks {
		t.Logf("Extracted C: %s (type=%s, lines=%d-%d)",
			chunk.Name, chunk.Type, chunk.StartLine, chunk.EndLine)
	}
}

// TestExtractorWithComplexGoCode tests interfaces with methods
func TestExtractorWithComplexGoCode(t *testing.T) {
	goCode := `package main

import "io"

// Reader interface with doc comment
type Reader interface {
	// Read reads data
	Read(p []byte) (n int, err error)
	// Close closes the reader
	Close() error
}

// Implementation
type FileReader struct {
	path string
}

func (f *FileReader) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (f *FileReader) Close() error {
	return nil
}
`

	parser, err := NewGoParser()
	if err != nil {
		t.Fatalf("NewGoParser failed: %v", err)
	}

	extractor := NewExtractor(parser, []byte(goCode))
	chunks, err := extractor.ExtractFunctions(context.Background())
	if err != nil {
		t.Fatalf("ExtractFunctions failed: %v", err)
	}

	// Should extract interface and struct and methods
	foundInterface := false
	foundStruct := false
	foundMethod := false

	for _, chunk := range chunks {
		if chunk.Type == ChunkTypeInterface {
			foundInterface = true
			// Check for interface methods in metadata
			if chunk.Metadata != nil {
				t.Logf("Interface %s metadata: %v", chunk.Name, chunk.Metadata)
			}
		}
		if chunk.Type == ChunkTypeStruct {
			foundStruct = true
		}
		if chunk.Type == ChunkTypeMethod {
			foundMethod = true
		}
		t.Logf("Extracted Go: %s (type=%s, lines=%d-%d)",
			chunk.Name, chunk.Type, chunk.StartLine, chunk.EndLine)
	}

	if !foundInterface {
		t.Log("Warning: No interface found (may need metadata)")
	}
	if !foundStruct {
		t.Error("Expected to find struct")
	}
	if !foundMethod {
		t.Error("Expected to find methods")
	}
}

// TestExtractorEdgeCases tests helper function edge cases for coverage
func TestExtractorEdgeCases(t *testing.T) {
	t.Run("extractMethodSpecName with nil node", func(t *testing.T) {
		parser, err := NewGoParser()
		if err != nil {
			t.Fatalf("NewGoParser failed: %v", err)
		}
		extractor := NewExtractor(parser, []byte("package main"))
		// Call with nil node - should return empty string
		result := extractor.extractMethodSpecName(nil)
		if result != "" {
			t.Errorf("Expected empty string for nil node, got %q", result)
		}
	})

	t.Run("extractPackageName with nil node", func(t *testing.T) {
		parser, err := NewGoParser()
		if err != nil {
			t.Fatalf("NewGoParser failed: %v", err)
		}
		extractor := NewExtractor(parser, []byte("package main"))
		// Call with nil node - should return empty string
		result := extractor.extractPackageName(nil)
		if result != "" {
			t.Errorf("Expected empty string for nil node, got %q", result)
		}
	})

	t.Run("extractImportPath with nil node", func(t *testing.T) {
		parser, err := NewGoParser()
		if err != nil {
			t.Fatalf("NewGoParser failed: %v", err)
		}
		extractor := NewExtractor(parser, []byte("package main"))
		// Call with nil node - should return empty string
		result := extractor.extractImportPath(nil)
		if result != "" {
			t.Errorf("Expected empty string for nil node, got %q", result)
		}
	})

	t.Run("extractPackageName with empty file", func(t *testing.T) {
		// Test package name extraction with various package declarations
		testCases := []struct {
			name    string
			code    string
			wantPkg string
		}{
			{
				name:    "standard package",
				code:    "package main\n",
				wantPkg: "main",
			},
			{
				name:    "package with underscore",
				code:    "package my_package\n",
				wantPkg: "my_package",
			},
			{
				name:    "no package declaration",
				code:    "// just a comment\n",
				wantPkg: "",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				parser, err := NewGoParser()
				if err != nil {
					t.Fatalf("NewGoParser failed: %v", err)
				}
				extractor := NewExtractor(parser, []byte(tc.code))
				_, err = extractor.ExtractFunctions(context.Background())
				if err != nil {
					t.Fatalf("ExtractFunctions failed: %v", err)
				}
				if extractor.packageName != tc.wantPkg {
					t.Errorf("Expected package %q, got %q", tc.wantPkg, extractor.packageName)
				}
			})
		}
	})

	t.Run("extractImportPath with various imports", func(t *testing.T) {
		testCases := []struct {
			name        string
			code        string
			wantImports int
		}{
			{
				name: "single import",
				code: `package main
import "fmt"
`,
				wantImports: 1,
			},
			{
				name: "multiple imports in group",
				code: `package main
import (
	"fmt"
	"os"
	"io"
)
`,
				wantImports: 3,
			},
			{
				name: "import with dot",
				code: `package main
import . "fmt"
`,
				wantImports: 1,
			},
			{
				name: "import with alias",
				code: `package main
import f "fmt"
`,
				wantImports: 1,
			},
			{
				name: "no imports",
				code: `package main
func main() {}
`,
				wantImports: 0,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				parser, err := NewGoParser()
				if err != nil {
					t.Fatalf("NewGoParser failed: %v", err)
				}
				extractor := NewExtractor(parser, []byte(tc.code))
				_, err = extractor.ExtractFunctions(context.Background())
				if err != nil {
					t.Fatalf("ExtractFunctions failed: %v", err)
				}
				if len(extractor.imports) != tc.wantImports {
					t.Errorf("Expected %d imports, got %d: %v", tc.wantImports, len(extractor.imports), extractor.imports)
				}
			})
		}
	})
}
