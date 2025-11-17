package parser

import (
	"bytes"
	"path/filepath"
	"strings"
)

// Language represents a programming language
type Language int

const (
	LanguageUnknown Language = iota
	LanguageGo
	LanguagePython
	LanguageJavaScript
	LanguageTypeScript
	LanguageJava
	LanguageRust
	LanguageC
	LanguageCPP
	LanguageRuby
	LanguagePHP
	LanguageScala
)

// String returns the string representation of a language
func (l Language) String() string {
	switch l {
	case LanguageGo:
		return "go"
	case LanguagePython:
		return "python"
	case LanguageJavaScript:
		return "javascript"
	case LanguageTypeScript:
		return "typescript"
	case LanguageJava:
		return "java"
	case LanguageRust:
		return "rust"
	case LanguageC:
		return "c"
	case LanguageCPP:
		return "cpp"
	case LanguageRuby:
		return "ruby"
	case LanguagePHP:
		return "php"
	case LanguageScala:
		return "scala"
	default:
		return "unknown"
	}
}

// DetectLanguage determines the programming language from file path and content
func DetectLanguage(filePath string, content []byte) Language {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Unambiguous extensions
	switch ext {
	case ".go":
		return LanguageGo
	case ".py":
		return LanguagePython
	case ".js", ".jsx":
		return LanguageJavaScript
	case ".ts", ".tsx":
		return LanguageTypeScript
	case ".java":
		return LanguageJava
	case ".rs":
		return LanguageRust
	case ".rb":
		return LanguageRuby
	case ".php":
		return LanguagePHP
	case ".scala":
		return LanguageScala
	case ".cpp", ".cc", ".cxx":
		return LanguageCPP
	case ".hpp", ".hxx":
		return LanguageCPP
	}

	// Ambiguous extensions requiring heuristics
	if ext == ".c" {
		if containsCPlusPlusMarkers(content) {
			return LanguageCPP
		}
		return LanguageC
	}

	if ext == ".h" {
		// Default to C++ for .h files (most common in modern codebases)
		// Only use C if we detect it's clearly a C-only header
		if containsCPlusPlusMarkers(content) {
			return LanguageCPP
		}
		// If no C++ markers and has clear C-only patterns, use C
		if containsOnlyCMarkers(content) {
			return LanguageC
		}
		// Default to C++ when unclear
		return LanguageCPP
	}

	return LanguageUnknown
}

// containsCPlusPlusMarkers checks if content contains C++-specific constructs
func containsCPlusPlusMarkers(content []byte) bool {
	cppMarkers := [][]byte{
		[]byte("class "),
		[]byte("namespace "),
		[]byte("template<"),
		[]byte("::"),
		[]byte("std::"),
		[]byte("public:"),
		[]byte("private:"),
		[]byte("protected:"),
		[]byte("typename "),
		[]byte("constexpr "),
		[]byte("nullptr"),
		[]byte("virtual "),
		[]byte("override"),
		[]byte("final"),
		[]byte("delete"),
		[]byte(" new "),
	}

	for _, marker := range cppMarkers {
		if bytes.Contains(content, marker) {
			return true
		}
	}
	return false
}

// containsOnlyCMarkers checks if content appears to be C-only code
func containsOnlyCMarkers(content []byte) bool {
	// Look for typical C patterns
	cPatterns := [][]byte{
		[]byte("struct "),
		[]byte("typedef "),
		[]byte("void "),
		[]byte("int "),
		[]byte("char "),
		[]byte("#include <"),
	}

	// Must have at least some C patterns
	hasCPatterns := false
	for _, pattern := range cPatterns {
		if bytes.Contains(content, pattern) {
			hasCPatterns = true
			break
		}
	}

	// If no C patterns at all, it's not clearly C
	if !hasCPatterns {
		return false
	}

	// If it has C++ markers, it's not C-only
	if containsCPlusPlusMarkers(content) {
		return false
	}

	return true
}

// IsSupported returns true if the language is supported for parsing
func (l Language) IsSupported() bool {
	switch l {
	case LanguageGo:
		return true
	case LanguagePython,
		LanguageJavaScript,
		LanguageTypeScript,
		LanguageJava,
		LanguageRust,
		LanguageC,
		LanguageCPP,
		LanguageRuby,
		LanguagePHP,
		LanguageScala:
		return true // Will be implemented
	default:
		return false
	}
}

// FileExtensions returns the common file extensions for a language
func (l Language) FileExtensions() []string {
	switch l {
	case LanguageGo:
		return []string{".go"}
	case LanguagePython:
		return []string{".py"}
	case LanguageJavaScript:
		return []string{".js", ".jsx"}
	case LanguageTypeScript:
		return []string{".ts", ".tsx"}
	case LanguageJava:
		return []string{".java"}
	case LanguageRust:
		return []string{".rs"}
	case LanguageC:
		return []string{".c", ".h"}
	case LanguageCPP:
		return []string{".cpp", ".cc", ".cxx", ".hpp", ".hxx", ".h"}
	case LanguageRuby:
		return []string{".rb"}
	case LanguagePHP:
		return []string{".php"}
	case LanguageScala:
		return []string{".scala"}
	default:
		return []string{}
	}
}
