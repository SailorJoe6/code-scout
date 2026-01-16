package parser

import (
	"context"
	"strings"
	"testing"
)

func TestNewParser(t *testing.T) {
	tests := []struct {
		name     string
		lang     Language
		wantErr  bool
		testCode string
	}{
		{
			name:     "Go parser",
			lang:     LanguageGo,
			wantErr:  false,
			testCode: "package main\n\nfunc main() {}",
		},
		{
			name:     "Python parser",
			lang:     LanguagePython,
			wantErr:  false,
			testCode: "def main():\n    pass",
		},
		{
			name:     "JavaScript parser",
			lang:     LanguageJavaScript,
			wantErr:  false,
			testCode: "function main() {}",
		},
		{
			name:     "TypeScript parser",
			lang:     LanguageTypeScript,
			wantErr:  false,
			testCode: "function main(): void {}",
		},
		{
			name:     "Java parser",
			lang:     LanguageJava,
			wantErr:  false,
			testCode: "public class Main { public static void main(String[] args) {} }",
		},
		{
			name:     "Rust parser",
			lang:     LanguageRust,
			wantErr:  false,
			testCode: "fn main() {}",
		},
		{
			name:     "C parser",
			lang:     LanguageC,
			wantErr:  false,
			testCode: "int main() { return 0; }",
		},
		{
			name:     "C++ parser",
			lang:     LanguageCPP,
			wantErr:  false,
			testCode: "int main() { return 0; }",
		},
		{
			name:     "Ruby parser",
			lang:     LanguageRuby,
			wantErr:  false,
			testCode: "def main\nend",
		},
		{
			name:     "PHP parser",
			lang:     LanguagePHP,
			wantErr:  false,
			testCode: "<?php function main() {} ?>",
		},
		{
			name:     "Scala parser",
			lang:     LanguageScala,
			wantErr:  false,
			testCode: "object Main { def main(args: Array[String]): Unit = {} }",
		},
		{
			name:    "Unknown language",
			lang:    LanguageUnknown,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewParser(tt.lang)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if p == nil {
				t.Errorf("NewParser() returned nil parser")
				return
			}

			if p.Language() != tt.lang {
				t.Errorf("Parser.Language() = %v, want %v", p.Language(), tt.lang)
			}

			// Try parsing some test code
			if tt.testCode != "" {
				ctx := context.Background()
				tree, err := p.Parse(ctx, []byte(tt.testCode))
				if err != nil {
					t.Errorf("Parse() error = %v", err)
					return
				}
				if tree == nil {
					t.Errorf("Parse() returned nil tree")
					return
				}

				root := p.GetRootNode(tree)
				if root == nil {
					t.Errorf("GetRootNode() returned nil")
				}
			}
		})
	}
}

func TestNewGoParser_BackwardCompatibility(t *testing.T) {
	p, err := NewGoParser()
	if err != nil {
		t.Fatalf("NewGoParser() error = %v", err)
	}

	if p.Language() != LanguageGo {
		t.Errorf("NewGoParser().Language() = %v, want %v", p.Language(), LanguageGo)
	}

	ctx := context.Background()
	tree, err := p.Parse(ctx, []byte("package main\n\nfunc main() {}"))
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}
	if tree == nil {
		t.Errorf("Parse() returned nil tree")
	}
}

func TestParser_ParseRealCode(t *testing.T) {
	tests := []struct {
		name     string
		lang     Language
		code     string
		wantFunc bool // Should contain at least one function
	}{
		{
			name: "Python function",
			lang: LanguagePython,
			code: `def hello(name):
    print(f"Hello, {name}!")
    return True`,
			wantFunc: true,
		},
		{
			name: "JavaScript function",
			lang: LanguageJavaScript,
			code: `function hello(name) {
    console.log("Hello, " + name + "!");
    return true;
}`,
			wantFunc: true,
		},
		{
			name: "Java class with method",
			lang: LanguageJava,
			code: `public class Greeter {
    public void hello(String name) {
        System.out.println("Hello, " + name + "!");
    }
}`,
			wantFunc: true,
		},
		{
			name: "Rust function",
			lang: LanguageRust,
			code: `fn hello(name: &str) {
    println!("Hello, {}!", name);
}`,
			wantFunc: true,
		},
		{
			name: "C++ class",
			lang: LanguageCPP,
			code: `class Greeter {
public:
    void hello(const std::string& name) {
        std::cout << "Hello, " << name << "!" << std::endl;
    }
};`,
			wantFunc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewParser(tt.lang)
			if err != nil {
				t.Fatalf("NewParser() error = %v", err)
			}

			ctx := context.Background()
			tree, err := p.Parse(ctx, []byte(tt.code))
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}
			if tree == nil {
				t.Errorf("Parse() returned nil tree")
				return
			}

			root := p.GetRootNode(tree)
			if root == nil {
				t.Errorf("GetRootNode() returned nil")
				return
			}

			// Basic sanity check - the tree should have child nodes
			if root.ChildCount() == 0 {
				t.Errorf("Parse() produced empty tree for valid code")
			}
		})
	}
}

func TestExtractor_ExtractMethodSpecName(t *testing.T) {
	// Test extractMethodSpecName edge cases for improved coverage
	testCode := `package main

type Reader interface {
	Read(p []byte) (n int, err error)
	Close() error
}

type EmptyInterface interface {
}
`

	p, err := NewParser(LanguageGo)
	if err != nil {
		t.Fatalf("NewParser() error = %v", err)
	}

	ctx := context.Background()
	e := NewExtractor(p, []byte(testCode))
	chunks, err := e.ExtractFunctions(ctx)
	if err != nil {
		t.Fatalf("ExtractFunctions() error = %v", err)
	}

	// Should extract two interfaces
	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks, got %d", len(chunks))
	}

	// Verify that interface methods are extracted
	hasReaderInterface := false
	for _, chunk := range chunks {
		if chunk.Name == "Reader" && chunk.Type == ChunkTypeInterface {
			hasReaderInterface = true
			// Check that methods were extracted into metadata
			if fields, ok := chunk.Metadata["fields"]; ok {
				if !containsSubstring(fields, "Read") || !containsSubstring(fields, "Close") {
					t.Errorf("Reader interface should contain Read and Close methods, got: %s", fields)
				}
			}
		}
	}

	if !hasReaderInterface {
		t.Errorf("Should have extracted Reader interface")
	}
}

func TestExtractor_ExtractPackageName(t *testing.T) {
	// Test extractPackageName edge cases for improved coverage
	tests := []struct {
		name        string
		code        string
		wantPackage string
	}{
		{
			name:        "Standard package",
			code:        "package main\n\nfunc main() {}",
			wantPackage: "main",
		},
		{
			name:        "Different package name",
			code:        "package utils\n\nfunc Helper() {}",
			wantPackage: "utils",
		},
		{
			name:        "Package with underscore",
			code:        "package test_utils\n\nfunc Test() {}",
			wantPackage: "test_utils",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewParser(LanguageGo)
			if err != nil {
				t.Fatalf("NewParser() error = %v", err)
			}

			ctx := context.Background()
			e := NewExtractor(p, []byte(tt.code))
			chunks, err := e.ExtractFunctions(ctx)
			if err != nil {
				t.Fatalf("ExtractFunctions() error = %v", err)
			}

			// Check that package name was extracted
			if len(chunks) > 0 {
				if pkg, ok := chunks[0].Metadata["package"]; !ok || pkg != tt.wantPackage {
					t.Errorf("Expected package %q, got %q", tt.wantPackage, pkg)
				}
			}
		})
	}
}

func TestExtractor_ExtractImportPath(t *testing.T) {
	// Test extractImportPath edge cases for improved coverage
	tests := []struct {
		name        string
		code        string
		wantImports []string
	}{
		{
			name: "Single import",
			code: `package main

import "fmt"

func main() {}`,
			wantImports: []string{"fmt"},
		},
		{
			name: "Multiple imports",
			code: `package main

import (
	"fmt"
	"strings"
	"os"
)

func main() {}`,
			wantImports: []string{"fmt", "strings", "os"},
		},
		{
			name: "Import with alias",
			code: `package main

import (
	f "fmt"
	. "strings"
)

func main() {}`,
			wantImports: []string{"fmt", "strings"},
		},
		{
			name: "No imports",
			code: `package main

func main() {}`,
			wantImports: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewParser(LanguageGo)
			if err != nil {
				t.Fatalf("NewParser() error = %v", err)
			}

			ctx := context.Background()
			e := NewExtractor(p, []byte(tt.code))
			chunks, err := e.ExtractFunctions(ctx)
			if err != nil {
				t.Fatalf("ExtractFunctions() error = %v", err)
			}

			// Check that imports were extracted
			if len(chunks) > 0 {
				if importsStr, ok := chunks[0].Metadata["imports"]; ok {
					for _, wantImport := range tt.wantImports {
						if !containsSubstring(importsStr, wantImport) {
							t.Errorf("Expected import %q to be in %q", wantImport, importsStr)
						}
					}
				} else if len(tt.wantImports) > 0 {
					t.Errorf("Expected imports %v, but no imports found", tt.wantImports)
				}
			}
		})
	}
}

func TestDetectLanguage_ContainsOnlyCMarkers(t *testing.T) {
	// Test containsOnlyCMarkers edge cases for improved coverage
	tests := []struct {
		name     string
		content  string
		filePath string
		want     Language
	}{
		{
			name: "Pure C header",
			content: `#ifndef MYHEADER_H
#define MYHEADER_H

struct Point {
	int x;
	int y;
};

typedef struct {
	char name[100];
	int age;
} Person;

void initialize(void);

#endif`,
			filePath: "myheader.h",
			want:     LanguageC,
		},
		{
			name: "C++ header with class",
			content: `#ifndef MYHEADER_H
#define MYHEADER_H

class Point {
public:
	int x;
	int y;
};

#endif`,
			filePath: "myheader.h",
			want:     LanguageCPP,
		},
		{
			name: "Empty .h file defaults to C++",
			content: `// Empty header
`,
			filePath: "empty.h",
			want:     LanguageCPP,
		},
		{
			name: "Header with C patterns and C++ markers",
			content: `#include <stdio.h>

class MyClass {
	void doSomething();
};

struct MyStruct {
	int value;
};
`,
			filePath: "mixed.h",
			want:     LanguageCPP,
		},
		{
			name: ".c file with C++ markers",
			content: `#include <iostream>

class MyClass {
	void test();
};
`,
			filePath: "test.c",
			want:     LanguageCPP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.filePath, []byte(tt.content))
			if got != tt.want {
				t.Errorf("DetectLanguage() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || strings.Contains(s, substr))
}
