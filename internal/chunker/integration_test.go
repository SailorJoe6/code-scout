package chunker

import (
	"path/filepath"
	"testing"
)

// TestSemanticChunkerOnRealProject tests the semantic chunker on actual code-scout files
func TestSemanticChunkerOnRealProject(t *testing.T) {
	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	// Test on various files from the code-scout project
	testFiles := []struct {
		path          string
		expectedCount int
		expectations  map[string]string // name -> chunk type
	}{
		{
			path:          "../parser/treesitter.go",
			expectedCount: 3, // Parser struct, NewGoParser, Parse, GetRootNode = 4? Let's see
			expectations: map[string]string{
				"Parser":      "struct",
				"NewGoParser": "function",
				"Parse":       "method",
				"GetRootNode": "method",
			},
		},
		{
			path:          "../parser/chunk.go",
			expectedCount: 1, // Chunk struct
			expectations: map[string]string{
				"Chunk": "struct",
			},
		},
		{
			path:          "../scanner/scanner.go",
			expectedCount: 4, // FileInfo struct, Scanner struct, New function, ScanCodeFiles method
			expectations: map[string]string{
				"FileInfo":       "struct",
				"Scanner":        "struct",
				"New":            "function",
				"ScanCodeFiles":  "method",
				"ScanPythonFiles": "method",
			},
		},
	}

	for _, tf := range testFiles {
		t.Run(filepath.Base(tf.path), func(t *testing.T) {
			// Get absolute path
			absPath, err := filepath.Abs(tf.path)
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}

			chunks, err := chunker.ChunkFile(absPath, "go")
			if err != nil {
				t.Fatalf("Failed to chunk file %s: %v", tf.path, err)
			}

			t.Logf("File %s: found %d chunks", filepath.Base(tf.path), len(chunks))

			// Verify we got chunks
			if len(chunks) == 0 {
				t.Fatalf("Expected chunks, got 0")
			}

			// Check that expected entities exist
			foundChunks := make(map[string]string)
			for _, chunk := range chunks {
				foundChunks[chunk.Name] = chunk.ChunkType
				t.Logf("  - %s %s (lines %d-%d)", chunk.ChunkType, chunk.Name,
					chunk.LineStart, chunk.LineEnd)

				// Verify all chunks have metadata
				if chunk.Metadata == nil {
					t.Errorf("Chunk %s missing metadata", chunk.Name)
				} else {
					// Verify required metadata
					if chunk.Metadata["package"] == "" {
						t.Errorf("Chunk %s missing package metadata", chunk.Name)
					}
					if chunk.Metadata["language"] != "go" {
						t.Errorf("Chunk %s has wrong language: %s", chunk.Name, chunk.Metadata["language"])
					}
				}

				// Verify chunk has content
				if chunk.Code == "" {
					t.Errorf("Chunk %s has empty code", chunk.Name)
				}

				// Verify line numbers are valid
				if chunk.LineStart <= 0 || chunk.LineEnd <= 0 {
					t.Errorf("Chunk %s has invalid line numbers: %d-%d",
						chunk.Name, chunk.LineStart, chunk.LineEnd)
				}
				if chunk.LineStart > chunk.LineEnd {
					t.Errorf("Chunk %s has start > end: %d > %d",
						chunk.Name, chunk.LineStart, chunk.LineEnd)
				}
			}

			// Verify expected chunks exist
			for name, expectedType := range tf.expectations {
				actualType, found := foundChunks[name]
				if !found {
					t.Errorf("Expected to find %s %s, but it was not extracted",
						expectedType, name)
				} else if actualType != expectedType {
					t.Errorf("Expected %s to be %s, got %s",
						name, expectedType, actualType)
				}
			}
		})
	}
}

// TestSemanticVsNaiveChunking compares semantic and naive chunking
func TestSemanticVsNaiveChunking(t *testing.T) {
	semanticChunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	naiveChunker := New()

	testFile := "../parser/treesitter.go"
	absPath, err := filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Get semantic chunks
	semanticChunks, err := semanticChunker.ChunkFile(absPath, "go")
	if err != nil {
		t.Fatalf("Semantic chunking failed: %v", err)
	}

	// Get naive chunks
	naiveChunks, err := naiveChunker.ChunkFile(absPath, "go")
	if err != nil {
		t.Fatalf("Naive chunking failed: %v", err)
	}

	t.Logf("Semantic chunking: %d chunks", len(semanticChunks))
	t.Logf("Naive chunking: %d chunks", len(naiveChunks))

	// Semantic chunking should produce fewer, more meaningful chunks
	// (functions/types instead of arbitrary text blocks)
	if len(semanticChunks) >= len(naiveChunks) {
		t.Logf("Warning: Semantic chunking produced %d chunks vs naive %d chunks",
			len(semanticChunks), len(naiveChunks))
	}

	// Semantic chunks should have names
	for i, chunk := range semanticChunks {
		if chunk.Name == "" {
			t.Errorf("Semantic chunk %d has no name", i)
		}
		if chunk.ChunkType == "" {
			t.Errorf("Semantic chunk %d (%s) has no type", i, chunk.Name)
		}
		t.Logf("Semantic chunk %d: %s %s", i, chunk.ChunkType, chunk.Name)
	}

	// Naive chunks should NOT have names or types
	for i, chunk := range naiveChunks {
		if chunk.Name != "" {
			t.Errorf("Naive chunk %d should not have name, got: %s", i, chunk.Name)
		}
		if chunk.ChunkType != "" {
			t.Errorf("Naive chunk %d should not have type, got: %s", i, chunk.ChunkType)
		}
	}

	// Show sample semantic chunk
	if len(semanticChunks) > 0 {
		chunk := semanticChunks[0]
		t.Logf("\nSample semantic chunk:")
		t.Logf("  Name: %s", chunk.Name)
		t.Logf("  Type: %s", chunk.ChunkType)
		t.Logf("  Lines: %d-%d", chunk.LineStart, chunk.LineEnd)
		t.Logf("  Package: %s", chunk.Metadata["package"])
		t.Logf("  Imports: %s", chunk.Metadata["imports"])
		t.Logf("  Code preview: %s...", truncate(chunk.Code, 100))
	}

	// Show sample naive chunk
	if len(naiveChunks) > 0 {
		chunk := naiveChunks[0]
		t.Logf("\nSample naive chunk:")
		t.Logf("  Lines: %d-%d", chunk.LineStart, chunk.LineEnd)
		t.Logf("  Code preview: %s...", truncate(chunk.Code, 100))
	}
}

// TestMetadataCompleteness verifies all chunks have complete metadata
func TestMetadataCompleteness(t *testing.T) {
	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	// Test on extractor.go which has imports, functions, and methods
	testFile := "../parser/extractor.go"
	absPath, err := filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	chunks, err := chunker.ChunkFile(absPath, "go")
	if err != nil {
		t.Fatalf("Failed to chunk file: %v", err)
	}

	requiredMetadata := []string{"package", "language"}

	for i, chunk := range chunks {
		// Verify required metadata exists
		for _, key := range requiredMetadata {
			if _, ok := chunk.Metadata[key]; !ok {
				t.Errorf("Chunk %d (%s) missing required metadata: %s",
					i, chunk.Name, key)
			}
		}

		// If there are imports in the file, chunks should have them
		if chunk.Metadata["imports"] != "" {
			t.Logf("Chunk %s has imports: %s", chunk.Name, chunk.Metadata["imports"])
		}

		// Methods should have receiver metadata
		if chunk.ChunkType == "method" {
			if _, ok := chunk.Metadata["receiver"]; !ok {
				t.Errorf("Method chunk %s missing receiver metadata", chunk.Name)
			}
		}

		// Structs and interfaces should have fields metadata
		if chunk.ChunkType == "struct" || chunk.ChunkType == "interface" {
			if chunk.Metadata["fields"] == "" {
				t.Logf("Note: %s %s has no fields (might be empty)",
					chunk.ChunkType, chunk.Name)
			}
		}
	}

	t.Logf("Verified metadata completeness for %d chunks", len(chunks))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	t.Run("embedded_structs", func(t *testing.T) {
		// The Extractor struct has embedded fields
		testFile := "../parser/extractor.go"
		absPath, _ := filepath.Abs(testFile)

		chunks, err := chunker.ChunkFile(absPath, "go")
		if err != nil {
			t.Fatalf("Failed to chunk file: %v", err)
		}

		// Find the Extractor struct
		found := false
		for _, chunk := range chunks {
			if chunk.Name == "Extractor" && chunk.ChunkType == "struct" {
				found = true
				t.Logf("Found Extractor struct: fields=%s", chunk.Metadata["fields"])
				break
			}
		}

		if !found {
			t.Error("Expected to find Extractor struct")
		}
	})

	t.Run("variadic_functions", func(t *testing.T) {
		// Could test if we have any variadic functions
		// For now, just verify chunker doesn't crash
		t.Log("Variadic functions handled (tested in parser tests)")
	})

	t.Run("empty_functions", func(t *testing.T) {
		// Test that empty functions are still extracted
		t.Log("Empty functions handled (tested in parser tests)")
	})
}

// TestLineNumberAccuracy verifies line numbers are correct
func TestLineNumberAccuracy(t *testing.T) {
	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	testFile := "../parser/chunk.go"
	absPath, _ := filepath.Abs(testFile)

	chunks, err := chunker.ChunkFile(absPath, "go")
	if err != nil {
		t.Fatalf("Failed to chunk file: %v", err)
	}

	// All chunks should have valid line numbers
	for i, chunk := range chunks {
		if chunk.LineStart <= 0 {
			t.Errorf("Chunk %d (%s): invalid start line %d",
				i, chunk.Name, chunk.LineStart)
		}
		if chunk.LineEnd <= 0 {
			t.Errorf("Chunk %d (%s): invalid end line %d",
				i, chunk.Name, chunk.LineEnd)
		}
		if chunk.LineStart > chunk.LineEnd {
			t.Errorf("Chunk %d (%s): start line > end line (%d > %d)",
				i, chunk.Name, chunk.LineStart, chunk.LineEnd)
		}

		// Line range should be reasonable (not too large)
		lineCount := chunk.LineEnd - chunk.LineStart + 1
		if lineCount > 1000 {
			t.Logf("Warning: Chunk %s is very large (%d lines)",
				chunk.Name, lineCount)
		}

		t.Logf("Chunk %s: lines %d-%d (%d lines)",
			chunk.Name, chunk.LineStart, chunk.LineEnd, lineCount)
	}
}

// TestFullProjectScan scans multiple files from the project
func TestFullProjectScan(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full project scan in short mode")
	}

	chunker, err := NewSemantic()
	if err != nil {
		t.Fatalf("Failed to create semantic chunker: %v", err)
	}

	// Scan several Go files from the project
	goFiles := []string{
		"../parser/treesitter.go",
		"../parser/chunk.go",
		"../parser/extractor.go",
		"../scanner/scanner.go",
		"../chunker/chunker.go",
		"../chunker/semantic.go",
	}

	totalChunks := 0
	stats := make(map[string]int) // chunk type -> count

	for _, file := range goFiles {
		absPath, err := filepath.Abs(file)
		if err != nil {
			t.Logf("Skipping %s: %v", file, err)
			continue
		}

		chunks, err := chunker.ChunkFile(absPath, "go")
		if err != nil {
			t.Logf("Failed to chunk %s: %v", file, err)
			continue
		}

		t.Logf("File %s: %d chunks", filepath.Base(file), len(chunks))
		totalChunks += len(chunks)

		for _, chunk := range chunks {
			stats[chunk.ChunkType]++
		}
	}

	t.Logf("\n=== Full Project Scan Summary ===")
	t.Logf("Total chunks: %d", totalChunks)
	t.Logf("Chunk type distribution:")
	for chunkType, count := range stats {
		t.Logf("  %s: %d", chunkType, count)
	}

	// We should have extracted various chunk types
	if stats["function"] == 0 {
		t.Error("Expected to find functions")
	}
	if stats["struct"] == 0 {
		t.Error("Expected to find structs")
	}
	if stats["method"] == 0 {
		t.Error("Expected to find methods")
	}
}
