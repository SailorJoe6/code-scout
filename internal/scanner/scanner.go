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

// ScanPythonFiles recursively scans for Python files
func (s *Scanner) ScanPythonFiles() ([]FileInfo, error) {
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

		// Check for Python files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".py") {
			files = append(files, FileInfo{
				Path:     path,
				Language: "python",
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
