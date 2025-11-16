package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileInfo represents a discovered file
type FileInfo struct {
	Path     string
	Language string
	ModTime  time.Time
}

// Scanner scans directories for code files
type Scanner struct {
	rootDir string
}

// New creates a new Scanner
func New(rootDir string) *Scanner {
	return &Scanner{rootDir: rootDir}
}

// languageExtensions maps file extensions to language names
var languageExtensions = map[string]string{
	// Code files
	".py": "python",
	".go": "go",
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

		// Skip .code-scout directory
		if info.IsDir() && info.Name() == ".code-scout" {
			return filepath.SkipDir
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Skip hidden files
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
