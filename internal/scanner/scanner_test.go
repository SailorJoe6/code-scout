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
		"main.go":       "package main",
		"utils.py":      "def hello(): pass",
		"README.md":     "# README",
		"docs.txt":      "Documentation",
		"guide.rst":     "Guide",
		".hidden.go":    "should be skipped",
		"ignored.java":  "should be ignored (not supported)",
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
		"main.go":   "go",
		"utils.py":  "python",
		"README.md": "markdown",
		"docs.txt":  "text",
		"guide.rst": "rst",
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
		{".go", "go", true},
		{".py", "python", true},
		{".md", "markdown", true},
		{".txt", "text", true},
		{".rst", "rst", true},
		{".java", "", false},
		{".rs", "", false},
		{".js", "", false},
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
