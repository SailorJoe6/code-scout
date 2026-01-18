package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanCodeFiles_DetectsCodeFiles(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"main.go":      "package main",
		"utils.py":     "def hello(): pass",
		"README.md":    "# README",
		"docs.txt":     "Documentation",
		"guide.rst":    "Guide",
		".hidden.go":   "should be skipped",
		"helper.java":  "public class Helper {}",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create .code-scout directory (should be skipped)
	codeScoutDir := filepath.Join(tmpDir, ".code-scout")
	if err := os.Mkdir(codeScoutDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codeScoutDir, "test.go"), []byte("skip"), 0644); err != nil {
		t.Fatal(err)
	}

	// Scan directory
	scanner := New(tmpDir)
	results, err := scanner.ScanCodeFiles()
	if err != nil {
		t.Fatalf("ScanCodeFiles failed: %v", err)
	}

	// Verify results
	expected := map[string]string{
		"main.go":     "go",
		"utils.py":    "python",
		"README.md":   "markdown",
		"docs.txt":    "text",
		"guide.rst":   "rst",
		"helper.java": "java",
	}

	if len(results) != len(expected) {
		t.Errorf("Expected %d files, got %d", len(expected), len(results))
		for _, f := range results {
			t.Logf("Found: %s (%s)", filepath.Base(f.Path), f.Language)
		}
	}

	// Check each expected file
	found := make(map[string]string)
	for _, result := range results {
		basename := filepath.Base(result.Path)
		found[basename] = result.Language
	}

	for filename, expectedLang := range expected {
		if actualLang, ok := found[filename]; !ok {
			t.Errorf("File %s not found in results", filename)
		} else if actualLang != expectedLang {
			t.Errorf("File %s: expected language %s, got %s", filename, expectedLang, actualLang)
		}
	}

	// Verify .code-scout files are skipped
	for _, result := range results {
		if filepath.Dir(result.Path) == codeScoutDir {
			t.Errorf("Found file in .code-scout directory: %s", result.Path)
		}
	}
}

func TestScanCodeFiles_RespectsHiddenDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hidden directory with files
	hiddenDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(hiddenDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "config.go"), []byte("skip"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create visible file
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	// Scan directory
	scanner := New(tmpDir)
	results, err := scanner.ScanCodeFiles()
	if err != nil {
		t.Fatalf("ScanCodeFiles failed: %v", err)
	}

	// Should only find main.go, not config.go in .git
	if len(results) != 1 {
		t.Errorf("Expected 1 file, got %d", len(results))
	}

	if len(results) > 0 && filepath.Base(results[0].Path) != "main.go" {
		t.Errorf("Expected main.go, got %s", filepath.Base(results[0].Path))
	}
}

func TestLanguageExtensions(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
		exists   bool
	}{
		// Go
		{".go", "go", true},
		// Python
		{".py", "python", true},
		// JavaScript
		{".js", "javascript", true},
		{".jsx", "javascript", true},
		{".mjs", "javascript", true},
		{".cjs", "javascript", true},
		// TypeScript
		{".ts", "typescript", true},
		{".tsx", "typescript", true},
		// Java
		{".java", "java", true},
		// Rust
		{".rs", "rust", true},
		// C
		{".c", "c", true},
		{".h", "c", true},
		// C++
		{".cpp", "cpp", true},
		{".cc", "cpp", true},
		{".cxx", "cpp", true},
		{".hpp", "cpp", true},
		{".hxx", "cpp", true},
		// Ruby
		{".rb", "ruby", true},
		// PHP
		{".php", "php", true},
		// Scala
		{".scala", "scala", true},
		// Documentation
		{".md", "markdown", true},
		{".txt", "text", true},
		{".rst", "rst", true},
		// Unsupported
		{".xml", "", false},
		{".json", "", false},
		{".yaml", "", false},
	}

	for _, tt := range tests {
		lang, ok := languageExtensions[tt.ext]
		if ok != tt.exists {
			t.Errorf("Extension %s: expected exists=%v, got %v", tt.ext, tt.exists, ok)
		}
		if ok && lang != tt.expected {
			t.Errorf("Extension %s: expected language %s, got %s", tt.ext, tt.expected, lang)
		}
	}
}

func TestScanPythonFiles(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"script.py":  "def main(): pass",
		"utils.py":   "# utility functions",
		"README.md":  "# README",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test ScanPythonFiles (deprecated, but should work like ScanCodeFiles)
	scanner := New(tmpDir)
	results, err := scanner.ScanPythonFiles()
	if err != nil {
		t.Fatalf("ScanPythonFiles failed: %v", err)
	}

	// Should find all files (both .py and .md)
	if len(results) != 3 {
		t.Errorf("Expected 3 files, got %d", len(results))
	}

	// Verify it finds Python files
	foundPython := false
	for _, result := range results {
		if result.Language == "python" {
			foundPython = true
			break
		}
	}
	if !foundPython {
		t.Error("ScanPythonFiles did not find any Python files")
	}
}

func TestScanCodeFiles_NonexistentDirectory(t *testing.T) {
	// Test scanning a non-existent directory
	scanner := New("/nonexistent/path/that/does/not/exist")
	results, err := scanner.ScanCodeFiles()

	// Should return an error
	if err == nil {
		t.Error("Expected error when scanning non-existent directory, got nil")
	}

	// Results should be nil or empty
	if results != nil && len(results) > 0 {
		t.Errorf("Expected no results for non-existent directory, got %d", len(results))
	}
}

func TestIgnorePatterns_GitignoreOnly(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitignore
	gitignore := `# Test .gitignore
*.log
build/
temp.py
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignore), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test files
	files := map[string]bool{
		"main.go":        false, // should be found
		"test.log":       true,  // should be ignored (*.log)
		"app.log":        true,  // should be ignored (*.log)
		"utils.py":       false, // should be found
		"temp.py":        true,  // should be ignored (exact match)
		"build/out.go":   true,  // should be ignored (in build/)
		"src/helper.go":  false, // should be found
	}

	for name := range files {
		path := filepath.Join(tmpDir, name)
		dir := filepath.Dir(path)
		if dir != tmpDir {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Scan directory
	scanner := New(tmpDir)
	results, err := scanner.ScanCodeFiles()
	if err != nil {
		t.Fatalf("ScanCodeFiles failed: %v", err)
	}

	// Build map of found files
	found := make(map[string]bool)
	for _, result := range results {
		relPath, _ := filepath.Rel(tmpDir, result.Path)
		found[relPath] = true
	}

	// Verify expected files
	for filename, shouldIgnore := range files {
		if shouldIgnore {
			if found[filename] {
				t.Errorf("File %s should have been ignored but was found", filename)
			}
		} else {
			if !found[filename] {
				t.Errorf("File %s should have been found but was ignored", filename)
			}
		}
	}
}

func TestIgnorePatterns_CodeScoutIgnoreOnly(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .code-scout-ignore
	codeScoutIgnore := `# Test .code-scout-ignore
examples/
test_data/
*.tmp
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".code-scout-ignore"), []byte(codeScoutIgnore), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test files
	files := map[string]bool{
		"main.go":                false, // should be found
		"examples/demo.go":       true,  // should be ignored (examples/)
		"test_data/fixture.py":   true,  // should be ignored (test_data/)
		"cache.tmp":              true,  // should be ignored (*.tmp)
		"src/utils.go":           false, // should be found
	}

	for name := range files {
		path := filepath.Join(tmpDir, name)
		dir := filepath.Dir(path)
		if dir != tmpDir {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Scan directory
	scanner := New(tmpDir)
	results, err := scanner.ScanCodeFiles()
	if err != nil {
		t.Fatalf("ScanCodeFiles failed: %v", err)
	}

	// Build map of found files
	found := make(map[string]bool)
	for _, result := range results {
		relPath, _ := filepath.Rel(tmpDir, result.Path)
		found[relPath] = true
	}

	// Verify expected files
	for filename, shouldIgnore := range files {
		if shouldIgnore {
			if found[filename] {
				t.Errorf("File %s should have been ignored but was found", filename)
			}
		} else {
			if !found[filename] {
				t.Errorf("File %s should have been found but was ignored", filename)
			}
		}
	}
}

func TestIgnorePatterns_BothIgnoreFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitignore
	gitignore := `*.log
dist/
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignore), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .code-scout-ignore
	codeScoutIgnore := `node_modules/
*.tmp
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".code-scout-ignore"), []byte(codeScoutIgnore), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test files
	files := map[string]bool{
		"main.go":               false, // should be found
		"app.log":               true,  // ignored by .gitignore
		"cache.tmp":             true,  // ignored by .code-scout-ignore
		"dist/bundle.go":        true,  // ignored by .gitignore
		"node_modules/lib.go":   true,  // ignored by .code-scout-ignore
		"src/utils.go":          false, // should be found
	}

	for name := range files {
		path := filepath.Join(tmpDir, name)
		dir := filepath.Dir(path)
		if dir != tmpDir {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Scan directory
	scanner := New(tmpDir)
	results, err := scanner.ScanCodeFiles()
	if err != nil {
		t.Fatalf("ScanCodeFiles failed: %v", err)
	}

	// Build map of found files
	found := make(map[string]bool)
	for _, result := range results {
		relPath, _ := filepath.Rel(tmpDir, result.Path)
		found[relPath] = true
	}

	// Verify expected files
	for filename, shouldIgnore := range files {
		if shouldIgnore {
			if found[filename] {
				t.Errorf("File %s should have been ignored but was found", filename)
			}
		} else {
			if !found[filename] {
				t.Errorf("File %s should have been found but was ignored", filename)
			}
		}
	}
}

func TestIgnorePatterns_NoIgnoreFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files (no .gitignore or .code-scout-ignore)
	files := []string{
		"main.go",
		"utils.py",
		"README.md",
	}

	for _, name := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Scan directory
	scanner := New(tmpDir)
	results, err := scanner.ScanCodeFiles()
	if err != nil {
		t.Fatalf("ScanCodeFiles failed: %v", err)
	}

	// Should find all files (except hidden ones)
	if len(results) != len(files) {
		t.Errorf("Expected %d files, got %d", len(files), len(results))
	}
}

func TestIgnorePatterns_EmptyIgnoreFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty .gitignore
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test files
	files := []string{"main.go", "utils.py"}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Scan directory
	scanner := New(tmpDir)
	results, err := scanner.ScanCodeFiles()
	if err != nil {
		t.Fatalf("ScanCodeFiles failed: %v", err)
	}

	// Should find all files
	if len(results) != len(files) {
		t.Errorf("Expected %d files, got %d", len(files), len(results))
	}
}

func TestIgnorePatterns_CommentsAndEmptyLines(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitignore with comments and empty lines
	gitignore := `# This is a comment
*.log

# Another comment

*.tmp

# Trailing whitespace above
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignore), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test files
	files := map[string]bool{
		"main.go":   false,
		"test.log":  true,
		"cache.tmp": true,
	}

	for name := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Scan directory
	scanner := New(tmpDir)
	results, err := scanner.ScanCodeFiles()
	if err != nil {
		t.Fatalf("ScanCodeFiles failed: %v", err)
	}

	// Build map of found files
	found := make(map[string]bool)
	for _, result := range results {
		relPath, _ := filepath.Rel(tmpDir, result.Path)
		found[relPath] = true
	}

	// Verify expected files
	for filename, shouldIgnore := range files {
		if shouldIgnore {
			if found[filename] {
				t.Errorf("File %s should have been ignored but was found", filename)
			}
		} else {
			if !found[filename] {
				t.Errorf("File %s should have been found but was ignored", filename)
			}
		}
	}
}

func TestIgnorePatterns_CodeScoutDirectoryAlwaysIgnored(t *testing.T) {
	tmpDir := t.TempDir()

	// Don't create any .gitignore or .code-scout-ignore
	// .code-scout should still be ignored (hardcoded)

	// Create .code-scout directory
	codeScoutDir := filepath.Join(tmpDir, ".code-scout")
	if err := os.Mkdir(codeScoutDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codeScoutDir, "data.go"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create regular file
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Scan directory
	scanner := New(tmpDir)
	results, err := scanner.ScanCodeFiles()
	if err != nil {
		t.Fatalf("ScanCodeFiles failed: %v", err)
	}

	// Should only find main.go, not data.go in .code-scout
	if len(results) != 1 {
		t.Errorf("Expected 1 file, got %d", len(results))
	}

	if len(results) > 0 && filepath.Base(results[0].Path) != "main.go" {
		t.Errorf("Expected main.go, got %s", filepath.Base(results[0].Path))
	}
}

func TestReadIgnoreFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with valid file
	validPath := filepath.Join(tmpDir, "valid.ignore")
	content := `# Comment
*.log

*.tmp
  # Another comment
`
	if err := os.WriteFile(validPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	patterns := readIgnoreFile(validPath)
	expected := []string{"*.log", "*.tmp"}

	if len(patterns) != len(expected) {
		t.Errorf("Expected %d patterns, got %d", len(expected), len(patterns))
	}

	for i, pattern := range expected {
		if i >= len(patterns) || patterns[i] != pattern {
			t.Errorf("Expected pattern[%d] = %s, got %s", i, pattern, patterns[i])
		}
	}

	// Test with non-existent file
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.ignore")
	patterns = readIgnoreFile(nonExistentPath)
	if patterns != nil {
		t.Errorf("Expected nil for non-existent file, got %d patterns", len(patterns))
	}
}
