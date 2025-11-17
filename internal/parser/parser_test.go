package parser

import (
	"context"
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
