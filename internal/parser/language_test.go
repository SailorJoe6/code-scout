package parser

import "testing"

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		content  string
		want     Language
	}{
		// Unambiguous cases
		{
			name:     "Go file",
			filePath: "main.go",
			content:  "package main",
			want:     LanguageGo,
		},
		{
			name:     "Python file",
			filePath: "script.py",
			content:  "def main():",
			want:     LanguagePython,
		},
		{
			name:     "JavaScript file",
			filePath: "app.js",
			content:  "function main() {}",
			want:     LanguageJavaScript,
		},
		{
			name:     "TypeScript file",
			filePath: "app.ts",
			content:  "function main(): void {}",
			want:     LanguageTypeScript,
		},
		{
			name:     "Java file",
			filePath: "Main.java",
			content:  "public class Main {}",
			want:     LanguageJava,
		},
		{
			name:     "Rust file",
			filePath: "main.rs",
			content:  "fn main() {}",
			want:     LanguageRust,
		},
		{
			name:     "Ruby file",
			filePath: "script.rb",
			content:  "def main\nend",
			want:     LanguageRuby,
		},
		{
			name:     "PHP file",
			filePath: "index.php",
			content:  "<?php echo 'hello'; ?>",
			want:     LanguagePHP,
		},
		{
			name:     "Scala file",
			filePath: "Main.scala",
			content:  "object Main {}",
			want:     LanguageScala,
		},

		// C++ unambiguous
		{
			name:     "C++ .cpp file",
			filePath: "main.cpp",
			content:  "int main() {}",
			want:     LanguageCPP,
		},
		{
			name:     "C++ .hpp file",
			filePath: "header.hpp",
			content:  "#pragma once",
			want:     LanguageCPP,
		},

		// C/C++ ambiguous - .c files
		{
			name:     ".c file with C++ class",
			filePath: "legacy.c",
			content:  "class Foo { public: int x; };",
			want:     LanguageCPP,
		},
		{
			name:     ".c file with C++ namespace",
			filePath: "old.c",
			content:  "namespace foo { int bar; }",
			want:     LanguageCPP,
		},
		{
			name:     ".c file with C++ std::",
			filePath: "code.c",
			content:  "#include <iostream>\nstd::cout << \"hello\";",
			want:     LanguageCPP,
		},
		{
			name:     ".c file pure C",
			filePath: "util.c",
			content:  "#include <stdio.h>\nint add(int a, int b) { return a + b; }",
			want:     LanguageC,
		},

		// C/C++ ambiguous - .h files
		{
			name:     ".h file with C++ class",
			filePath: "header.h",
			content:  "class Widget { public: void draw(); };",
			want:     LanguageCPP,
		},
		{
			name:     ".h file with C++ template",
			filePath: "template.h",
			content:  "template<typename T> class Container {};",
			want:     LanguageCPP,
		},
		{
			name:     ".h file with namespace",
			filePath: "api.h",
			content:  "namespace api { void init(); }",
			want:     LanguageCPP,
		},
		{
			name:     ".h file pure C struct",
			filePath: "types.h",
			content:  "typedef struct { int x; int y; } Point;",
			want:     LanguageC,
		},
		{
			name:     ".h file pure C functions",
			filePath: "utils.h",
			content:  "#include <stdlib.h>\nvoid* malloc_wrapper(size_t size);",
			want:     LanguageC,
		},
		{
			name:     ".h file empty/unclear defaults to C++",
			filePath: "empty.h",
			content:  "// Just a comment",
			want:     LanguageCPP,
		},

		// Edge cases
		{
			name:     "unknown extension",
			filePath: "data.txt",
			content:  "some text",
			want:     LanguageUnknown,
		},
		{
			name:     "no extension",
			filePath: "Makefile",
			content:  "all: build",
			want:     LanguageUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.filePath, []byte(tt.content))
			if got != tt.want {
				t.Errorf("DetectLanguage() = %v (%s), want %v (%s)",
					got, got.String(), tt.want, tt.want.String())
			}
		})
	}
}

func TestLanguageString(t *testing.T) {
	tests := []struct {
		lang Language
		want string
	}{
		{LanguageGo, "go"},
		{LanguagePython, "python"},
		{LanguageJavaScript, "javascript"},
		{LanguageTypeScript, "typescript"},
		{LanguageJava, "java"},
		{LanguageRust, "rust"},
		{LanguageC, "c"},
		{LanguageCPP, "cpp"},
		{LanguageRuby, "ruby"},
		{LanguagePHP, "php"},
		{LanguageScala, "scala"},
		{LanguageUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.lang.String(); got != tt.want {
				t.Errorf("Language.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLanguageIsSupported(t *testing.T) {
	tests := []struct {
		lang Language
		want bool
	}{
		{LanguageGo, true},
		{LanguagePython, true},
		{LanguageJavaScript, true},
		{LanguageTypeScript, true},
		{LanguageJava, true},
		{LanguageRust, true},
		{LanguageC, true},
		{LanguageCPP, true},
		{LanguageRuby, true},
		{LanguagePHP, true},
		{LanguageScala, true},
		{LanguageUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.lang.String(), func(t *testing.T) {
			if got := tt.lang.IsSupported(); got != tt.want {
				t.Errorf("Language.IsSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLanguageFileExtensions(t *testing.T) {
	tests := []struct {
		lang Language
		want []string
	}{
		{LanguageGo, []string{".go"}},
		{LanguagePython, []string{".py"}},
		{LanguageJavaScript, []string{".js", ".jsx"}},
		{LanguageTypeScript, []string{".ts", ".tsx"}},
		{LanguageJava, []string{".java"}},
		{LanguageRust, []string{".rs"}},
		{LanguageC, []string{".c", ".h"}},
		{LanguageCPP, []string{".cpp", ".cc", ".cxx", ".hpp", ".hxx", ".h"}},
		{LanguageRuby, []string{".rb"}},
		{LanguagePHP, []string{".php"}},
		{LanguageScala, []string{".scala"}},
	}

	for _, tt := range tests {
		t.Run(tt.lang.String(), func(t *testing.T) {
			got := tt.lang.FileExtensions()
			if len(got) != len(tt.want) {
				t.Errorf("Language.FileExtensions() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Language.FileExtensions()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
