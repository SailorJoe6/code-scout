package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	ignore "github.com/sabhiram/go-gitignore"
)

// FileInfo represents a discovered file
type FileInfo struct {
	Path     string
	Language string
	ModTime  time.Time
}

// Scanner scans directories for code files
type Scanner struct {
	rootDir       string
	ignoreMatcher *ignore.GitIgnore
}

// New creates a new Scanner
func New(rootDir string) *Scanner {
	scanner := &Scanner{rootDir: rootDir}
	scanner.loadIgnorePatterns()
	return scanner
}

// loadIgnorePatterns reads .gitignore and .code-scout-ignore files
func (s *Scanner) loadIgnorePatterns() {
	var allPatterns []string

	// Read .gitignore if it exists
	gitignorePath := filepath.Join(s.rootDir, ".gitignore")
	if patterns := readIgnoreFile(gitignorePath); len(patterns) > 0 {
		allPatterns = append(allPatterns, patterns...)
	}

	// Read .code-scout-ignore if it exists (takes precedence)
	codeScoutIgnorePath := filepath.Join(s.rootDir, ".code-scout-ignore")
	if patterns := readIgnoreFile(codeScoutIgnorePath); len(patterns) > 0 {
		allPatterns = append(allPatterns, patterns...)
	}

	// Always ignore .code-scout directory (hardcoded)
	allPatterns = append(allPatterns, ".code-scout/")

	// Create the ignore matcher if we have any patterns
	if len(allPatterns) > 0 {
		s.ignoreMatcher = ignore.CompileIgnoreLines(allPatterns...)
	}
}

// readIgnoreFile reads an ignore file and returns non-comment, non-empty lines
func readIgnoreFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist or can't be read - that's okay
		return nil
	}

	lines := strings.Split(string(data), "\n")
	var patterns []string

	for _, line := range lines {
		// Trim whitespace
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		patterns = append(patterns, line)
	}

	return patterns
}

// shouldIgnore checks if a path should be ignored based on ignore patterns
func (s *Scanner) shouldIgnore(path string, isDir bool) bool {
	if s.ignoreMatcher == nil {
		return false
	}

	// Convert to relative path from root directory
	relPath, err := filepath.Rel(s.rootDir, path)
	if err != nil {
		// If we can't get relative path, don't ignore
		return false
	}

	// For directories, append "/" to match gitignore behavior
	if isDir && !strings.HasSuffix(relPath, "/") {
		relPath = relPath + "/"
	}

	return s.ignoreMatcher.MatchesPath(relPath)
}

// languageExtensions maps file extensions to language names
// Note: For .h files, we assign "c" here, but DetectLanguage() in the parser
// will use heuristics to determine if it's actually C or C++ based on content
var languageExtensions = map[string]string{
	// Go
	".go": "go",

	// Python
	".py": "python",

	// JavaScript
	".js":  "javascript",
	".jsx": "javascript",
	".mjs": "javascript",
	".cjs": "javascript",

	// TypeScript
	".ts":  "typescript",
	".tsx": "typescript",

	// Java
	".java": "java",

	// Rust
	".rs": "rust",

	// C (note: .h files assigned to "c" but will be detected as C or C++ by parser)
	".c": "c",
	".h": "c",

	// C++
	".cpp": "cpp",
	".cc":  "cpp",
	".cxx": "cpp",
	".hpp": "cpp",
	".hxx": "cpp",

	// Ruby
	".rb": "ruby",

	// PHP
	".php": "php",

	// Scala
	".scala": "scala",

	// Documentation files
	".md":  "markdown",
	".txt": "text",
	".rst": "rst",
}

// ScanCodeFiles recursively scans for code and documentation files
func (s *Scanner) ScanCodeFiles() ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(s.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if path should be ignored
		if s.shouldIgnore(path, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden directories (as fallback if not in ignore patterns)
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Skip hidden files (as fallback if not in ignore patterns)
		if !info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Check for supported code and documentation files
		if !info.IsDir() {
			ext := filepath.Ext(info.Name())
			if lang, ok := languageExtensions[ext]; ok {
				files = append(files, FileInfo{
					Path:     path,
					Language: lang,
					ModTime:  info.ModTime(),
				})
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// ScanPythonFiles recursively scans for Python files (deprecated: use ScanCodeFiles)
func (s *Scanner) ScanPythonFiles() ([]FileInfo, error) {
	return s.ScanCodeFiles()
}
