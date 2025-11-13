package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// FileInfo represents a discovered file
type FileInfo struct {
	Path     string
	Language string
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
	".py": "python",
	".go": "go",
}

// ScanCodeFiles recursively scans for code files (Python, Go, etc.)
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

		// Check for supported code files
		if !info.IsDir() {
			ext := filepath.Ext(info.Name())
			if lang, ok := languageExtensions[ext]; ok {
				files = append(files, FileInfo{
					Path:     path,
					Language: lang,
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
