package chunker

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiLanguageChunking tests semantic chunking for all supported languages
func TestMultiLanguageChunking(t *testing.T) {
	tests := []struct {
		name           string
		file           string
		language       string
		minChunks      int // Minimum expected chunks
		expectTypes    []string
		expectNames    []string
	}{
		{
			name:        "Python",
			file:        "testdata/sample.py",
			language:    "python",
			minChunks:   10,
			expectTypes: []string{"function", "class"},
			expectNames: []string{"simple_function", "BaseClass", "DerivedClass"},
		},
		{
			name:        "JavaScript",
			file:        "testdata/sample.js",
			language:    "javascript",
			minChunks:   10,
			expectTypes: []string{"function", "class"},
			expectNames: []string{"simpleFunction", "BaseClass", "DerivedClass"},
		},
		{
			name:        "TypeScript",
			file:        "testdata/sample.ts",
			language:    "typescript",
			minChunks:   10,
			expectTypes: []string{"function", "class"},
			expectNames: []string{"greet", "Dog"},
		},
		{
			name:        "Java",
			file:        "testdata/Sample.java",
			language:    "java",
			minChunks:   10,
			expectTypes: []string{"class", "method"},
			expectNames: []string{"User", "UserRepository"},
		},
		{
			name:        "Rust",
			file:        "testdata/sample.rs",
			language:    "rust",
			minChunks:   10,
			expectTypes: []string{"function", "struct", "enum", "impl"},
			expectNames: []string{"greet", "Point", "Status"},
		},
		{
			name:        "C",
			file:        "testdata/sample.c",
			language:    "c",
			minChunks:   10,
			expectTypes: []string{"function", "struct"},
			expectNames: []string{}, // Names may vary based on parsing
		},
		{
			name:        "C++",
			file:        "testdata/sample.cpp",
			language:    "cpp",
			minChunks:   10,
			expectTypes: []string{"function", "class"},
			expectNames: []string{"Dog", "UserRepository"},
		},
		{
			name:        "Ruby",
			file:        "testdata/sample.rb",
			language:    "ruby",
			minChunks:   10,
			expectTypes: []string{"method", "class", "module"},
			expectNames: []string{"greet", "Animal", "Dog"},
		},
		{
			name:        "PHP",
			file:        "testdata/sample.php",
			language:    "php",
			minChunks:   10,
			expectTypes: []string{"function", "class"},
			expectNames: []string{"greet", "Dog", "UserRepository"},
		},
		{
			name:        "Scala",
			file:        "testdata/sample.scala",
			language:    "scala",
			minChunks:   10,
			expectTypes: []string{"function", "class"},
			expectNames: []string{"greet", "Dog", "UserRepository"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create semantic chunker
			chunker, err := NewSemantic()
			require.NoError(t, err, "Failed to create semantic chunker")

			// Get absolute path to test file
			absPath, err := filepath.Abs(tt.file)
			require.NoError(t, err, "Failed to get absolute path")

			// Chunk the file
			chunks, err := chunker.ChunkFile(absPath, tt.language)
			require.NoError(t, err, "Failed to chunk %s file", tt.language)

			// Verify minimum chunks extracted
			assert.GreaterOrEqual(t, len(chunks), tt.minChunks,
				"Expected at least %d chunks for %s, got %d", tt.minChunks, tt.language, len(chunks))

			// Verify chunks have correct language
			for _, chunk := range chunks {
				assert.Equal(t, tt.language, chunk.Language,
					"Chunk should have language '%s'", tt.language)
				assert.Equal(t, "code", chunk.EmbeddingType,
					"Chunk should have embedding_type 'code'")
				assert.NotEmpty(t, chunk.Code, "Chunk should have content")
				assert.Greater(t, chunk.LineEnd, 0, "Chunk should have valid line numbers")
			}

			// Verify expected chunk types exist
			foundTypes := make(map[string]bool)
			for _, chunk := range chunks {
				foundTypes[chunk.ChunkType] = true
			}
			for _, expectedType := range tt.expectTypes {
				assert.True(t, foundTypes[expectedType],
					"Expected to find chunk type '%s' in %s file", expectedType, tt.language)
			}

			// Verify expected names exist
			foundNames := make(map[string]bool)
			for _, chunk := range chunks {
				if chunk.Name != "" {
					foundNames[chunk.Name] = true
				}
			}
			for _, expectedName := range tt.expectNames {
				assert.True(t, foundNames[expectedName],
					"Expected to find chunk named '%s' in %s file", expectedName, tt.language)
			}

			// Log chunk summary for debugging
			t.Logf("%s: extracted %d chunks with types: %v",
				tt.language, len(chunks), getUniqueTypes(chunks))
		})
	}
}

// TestMultiLanguageChunkingDetails verifies detailed extraction for a few languages
func TestMultiLanguageChunkingDetails(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		language string
		checks   []chunkCheck
	}{
		{
			name:     "Python functions and classes",
			file:     "testdata/sample.py",
			language: "python",
			checks: []chunkCheck{
				{chunkType: "function", name: "simple_function", minLines: 2},
				{chunkType: "class", name: "BaseClass", minLines: 5},
				{chunkType: "class", name: "DerivedClass", minLines: 10},
			},
		},
		{
			name:     "Rust structs and impls",
			file:     "testdata/sample.rs",
			language: "rust",
			checks: []chunkCheck{
				{chunkType: "function", name: "greet", minLines: 2},
				{chunkType: "struct", name: "Point", minLines: 3},
				{chunkType: "enum", name: "Status", minLines: 4},
			},
		},
		{
			name:     "Java classes and methods",
			file:     "testdata/Sample.java",
			language: "java",
			checks: []chunkCheck{
				{chunkType: "class", name: "User", minLines: 10},
				{chunkType: "class", name: "UserRepository", minLines: 10},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunker, err := NewSemantic()
			require.NoError(t, err)

			absPath, err := filepath.Abs(tt.file)
			require.NoError(t, err)

			chunks, err := chunker.ChunkFile(absPath, tt.language)
			require.NoError(t, err)

			// Verify each expected chunk
			for _, check := range tt.checks {
				found := false
				for _, chunk := range chunks {
					if chunk.ChunkType == check.chunkType && chunk.Name == check.name {
						found = true
						lines := chunk.LineEnd - chunk.LineStart + 1
						assert.GreaterOrEqual(t, lines, check.minLines,
							"Chunk %s.%s should have at least %d lines, got %d",
							check.chunkType, check.name, check.minLines, lines)
						break
					}
				}
				assert.True(t, found, "Expected to find %s named '%s'",
					check.chunkType, check.name)
			}
		})
	}
}

// chunkCheck defines expected chunk properties
type chunkCheck struct {
	chunkType string
	name      string
	minLines  int
}

// getUniqueTypes returns unique chunk types from a slice of chunks
func getUniqueTypes(chunks []Chunk) []string {
	seen := make(map[string]bool)
	var types []string
	for _, chunk := range chunks {
		if !seen[chunk.ChunkType] {
			seen[chunk.ChunkType] = true
			types = append(types, chunk.ChunkType)
		}
	}
	return types
}
