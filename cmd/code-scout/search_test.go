package main

import (
	"testing"

	"github.com/jlanders/code-scout/internal/config"
)

// Test resolveSearchMode function
func TestResolveSearchMode(t *testing.T) {
	tests := []struct {
		name        string
		codeFlag    bool
		docsFlag    bool
		hybridFlag  bool
		expected    searchMode
		expectError bool
	}{
		{
			name:        "no flags defaults to hybrid",
			codeFlag:    false,
			docsFlag:    false,
			hybridFlag:  false,
			expected:    modeHybrid,
			expectError: false,
		},
		{
			name:        "code flag only",
			codeFlag:    true,
			docsFlag:    false,
			hybridFlag:  false,
			expected:    modeCode,
			expectError: false,
		},
		{
			name:        "docs flag only",
			codeFlag:    false,
			docsFlag:    true,
			hybridFlag:  false,
			expected:    modeDocs,
			expectError: false,
		},
		{
			name:        "hybrid flag only",
			codeFlag:    false,
			docsFlag:    false,
			hybridFlag:  true,
			expected:    modeHybrid,
			expectError: false,
		},
		{
			name:        "code and docs flags (mutually exclusive)",
			codeFlag:    true,
			docsFlag:    true,
			hybridFlag:  false,
			expected:    "",
			expectError: true,
		},
		{
			name:        "code and hybrid flags (mutually exclusive)",
			codeFlag:    true,
			docsFlag:    false,
			hybridFlag:  true,
			expected:    "",
			expectError: true,
		},
		{
			name:        "docs and hybrid flags (mutually exclusive)",
			codeFlag:    false,
			docsFlag:    true,
			hybridFlag:  true,
			expected:    "",
			expectError: true,
		},
		{
			name:        "all three flags (mutually exclusive)",
			codeFlag:    true,
			docsFlag:    true,
			hybridFlag:  true,
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global flags
			codeMode = tt.codeFlag
			docsMode = tt.docsFlag
			hybridMode = tt.hybridFlag

			result, err := resolveSearchMode()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected mode %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

// Test filterForMode function
func TestFilterForMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     searchMode
		expected string
	}{
		{
			name:     "code mode",
			mode:     modeCode,
			expected: "embedding_type = 'code'",
		},
		{
			name:     "docs mode",
			mode:     modeDocs,
			expected: "embedding_type = 'docs'",
		},
		{
			name:     "hybrid mode",
			mode:     modeHybrid,
			expected: "",
		},
		{
			name:     "empty mode defaults to no filter",
			mode:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterForMode(tt.mode)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Test formatResults function
func TestFormatResults(t *testing.T) {
	tests := []struct {
		name     string
		input    []map[string]interface{}
		expected []SearchResult
	}{
		{
			name:     "empty results",
			input:    []map[string]interface{}{},
			expected: []SearchResult{},
		},
		{
			name: "single complete result",
			input: []map[string]interface{}{
				{
					"chunk_id":       "abc123",
					"file_path":      "/path/to/file.go",
					"line_start":     int64(10),
					"line_end":       int64(20),
					"language":       "Go",
					"code":           "func main() {}",
					"_distance":      0.5,
					"embedding_type": "code",
					"chunk_type":     "function",
					"heading":        "Main Function",
					"heading_level":  "1",
					"parent_heading": "Module",
				},
			},
			expected: []SearchResult{
				{
					ChunkID:       "abc123",
					FilePath:      "/path/to/file.go",
					LineStart:     10,
					LineEnd:       20,
					Language:      "Go",
					Code:          "func main() {}",
					Score:         0.5,
					EmbeddingType: "code",
					ChunkType:     "function",
					Heading:       "Main Function",
					HeadingLevel:  "1",
					ParentHeading: "Module",
				},
			},
		},
		{
			name: "result with missing fields uses defaults",
			input: []map[string]interface{}{
				{
					"file_path": "/path/to/file.go",
					"_distance": 0.8,
				},
			},
			expected: []SearchResult{
				{
					ChunkID:       "",
					FilePath:      "/path/to/file.go",
					LineStart:     0,
					LineEnd:       0,
					Language:      "",
					Code:          "",
					Score:         0.8,
					EmbeddingType: "",
					ChunkType:     "",
					Heading:       "",
					HeadingLevel:  "",
					ParentHeading: "",
				},
			},
		},
		{
			name: "handles different int types",
			input: []map[string]interface{}{
				{
					"line_start": int(5),
					"line_end":   int32(10),
					"_distance":  0.3,
				},
				{
					"line_start": int64(15),
					"line_end":   float64(25.7), // Should truncate
					"_distance":  0.4,
				},
			},
			expected: []SearchResult{
				{
					LineStart: 5,
					LineEnd:   10,
					Score:     0.3,
				},
				{
					LineStart: 15,
					LineEnd:   25,
					Score:     0.4,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatResults(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d results, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("result[%d]: expected %+v, got %+v", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

// Test helper functions
func TestGetStringOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		input        map[string]interface{}
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "key exists with string value",
			input:        map[string]interface{}{"foo": "bar"},
			key:          "foo",
			defaultValue: "default",
			expected:     "bar",
		},
		{
			name:         "key does not exist",
			input:        map[string]interface{}{"foo": "bar"},
			key:          "missing",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "key exists but not a string",
			input:        map[string]interface{}{"foo": 123},
			key:          "foo",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "empty map",
			input:        map[string]interface{}{},
			key:          "foo",
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringOrDefault(tt.input, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetIntOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		input        map[string]interface{}
		key          string
		defaultValue int
		expected     int
	}{
		{
			name:         "int value",
			input:        map[string]interface{}{"num": 42},
			key:          "num",
			defaultValue: 0,
			expected:     42,
		},
		{
			name:         "int32 value",
			input:        map[string]interface{}{"num": int32(42)},
			key:          "num",
			defaultValue: 0,
			expected:     42,
		},
		{
			name:         "int64 value",
			input:        map[string]interface{}{"num": int64(42)},
			key:          "num",
			defaultValue: 0,
			expected:     42,
		},
		{
			name:         "float64 value truncates",
			input:        map[string]interface{}{"num": float64(42.7)},
			key:          "num",
			defaultValue: 0,
			expected:     42,
		},
		{
			name:         "missing key",
			input:        map[string]interface{}{"other": 100},
			key:          "num",
			defaultValue: 99,
			expected:     99,
		},
		{
			name:         "wrong type",
			input:        map[string]interface{}{"num": "not a number"},
			key:          "num",
			defaultValue: 99,
			expected:     99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIntOrDefault(tt.input, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetFloat64OrDefault(t *testing.T) {
	tests := []struct {
		name         string
		input        map[string]interface{}
		key          string
		defaultValue float64
		expected     float64
	}{
		{
			name:         "float64 value",
			input:        map[string]interface{}{"num": 3.14},
			key:          "num",
			defaultValue: 0.0,
			expected:     3.14,
		},
		{
			name:         "missing key",
			input:        map[string]interface{}{"other": 1.0},
			key:          "num",
			defaultValue: 2.71,
			expected:     2.71,
		},
		{
			name:         "wrong type",
			input:        map[string]interface{}{"num": "not a number"},
			key:          "num",
			defaultValue: 2.71,
			expected:     2.71,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFloat64OrDefault(tt.input, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// NOTE: TestCosineSimilarity was removed as the cosineSimilarity function
// was replaced with proper cross-encoder reranking (see code_scout-n5n)

// Test deduplicateResults function
func TestDeduplicateResults(t *testing.T) {
	tests := []struct {
		name     string
		input    []SearchResult
		expected []SearchResult
	}{
		{
			name:     "empty results",
			input:    []SearchResult{},
			expected: []SearchResult{},
		},
		{
			name: "no duplicates",
			input: []SearchResult{
				{Code: "func a() {}", Score: 0.1},
				{Code: "func b() {}", Score: 0.2},
			},
			expected: []SearchResult{
				{Code: "func a() {}", Score: 0.1},
				{Code: "func b() {}", Score: 0.2},
			},
		},
		{
			name: "duplicates - keeps best score",
			input: []SearchResult{
				{Code: "func main() {}", FilePath: "/a.go", Score: 0.5},
				{Code: "func main() {}", FilePath: "/b.go", Score: 0.3}, // Better score
				{Code: "func main() {}", FilePath: "/c.go", Score: 0.7},
			},
			expected: []SearchResult{
				{Code: "func main() {}", FilePath: "/b.go", Score: 0.3},
			},
		},
		{
			name: "mixed duplicates and uniques",
			input: []SearchResult{
				{Code: "func a() {}", Score: 0.1},
				{Code: "func b() {}", Score: 0.4},
				{Code: "func a() {}", Score: 0.2}, // Duplicate
				{Code: "func c() {}", Score: 0.3},
			},
			expected: []SearchResult{
				{Code: "func a() {}", Score: 0.1},
				{Code: "func c() {}", Score: 0.3},
				{Code: "func b() {}", Score: 0.4},
			},
		},
		{
			name: "results are sorted by score",
			input: []SearchResult{
				{Code: "func c() {}", Score: 0.9},
				{Code: "func a() {}", Score: 0.1},
				{Code: "func b() {}", Score: 0.5},
			},
			expected: []SearchResult{
				{Code: "func a() {}", Score: 0.1},
				{Code: "func b() {}", Score: 0.5},
				{Code: "func c() {}", Score: 0.9},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateResults(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d results, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i].Code != tt.expected[i].Code || result[i].Score != tt.expected[i].Score {
					t.Errorf("result[%d]: expected (code=%q, score=%f), got (code=%q, score=%f)",
						i, tt.expected[i].Code, tt.expected[i].Score, result[i].Code, result[i].Score)
				}
			}
		})
	}
}

// Test buildRerankText function
func TestBuildRerankText(t *testing.T) {
	tests := []struct {
		name     string
		input    SearchResult
		expected string
	}{
		{
			name: "complete result",
			input: SearchResult{
				FilePath:      "/path/to/file.go",
				Language:      "Go",
				ChunkType:     "function",
				Heading:       "Main",
				HeadingLevel:  "1",
				ParentHeading: "Module",
				Code:          "func main() {}",
			},
			expected: "File: /path/to/file.go\nLanguage: Go\nChunk: function\nHeading: Main (level 1)\nParents: Module\nfunc main() {}",
		},
		{
			name: "minimal result (code only)",
			input: SearchResult{
				Code: "func test() {}",
			},
			expected: "func test() {}",
		},
		{
			name: "partial fields",
			input: SearchResult{
				FilePath: "/file.go",
				Language: "Go",
				Code:     "package main",
			},
			expected: "File: /file.go\nLanguage: Go\npackage main",
		},
		{
			name: "heading without level or parent",
			input: SearchResult{
				Heading: "Introduction",
				Code:    "Some text",
			},
			expected: "Heading: Introduction\nSome text",
		},
		{
			name: "heading with level but no parent",
			input: SearchResult{
				Heading:      "Section",
				HeadingLevel: "2",
				Code:         "Content",
			},
			expected: "Heading: Section (level 2)\nContent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRerankText(tt.input)
			if result != tt.expected {
				t.Errorf("expected:\n%s\n\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

// Test float64Ptr helper
func TestFloat64Ptr(t *testing.T) {
	val := 3.14
	ptr := float64Ptr(val)
	if ptr == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *ptr != val {
		t.Errorf("expected %f, got %f", val, *ptr)
	}
}

// Test rerankTopK function
func TestRerankTopK(t *testing.T) {
	tests := []struct {
		name           string
		globalConfig   *config.Config
		limit          int
		expected       int
	}{
		{
			name:         "nil globalConfig",
			globalConfig: nil,
			limit:        10,
			expected:     0,
		},
		{
			name: "empty RerankModel",
			globalConfig: &config.Config{
				RerankModel: "",
			},
			limit:    10,
			expected: 0,
		},
		{
			name: "RerankModel set, RerankTopK not set (uses limit)",
			globalConfig: &config.Config{
				RerankModel: "rerank-model",
				RerankTopK:  0,
			},
			limit:    20,
			expected: 20,
		},
		{
			name: "RerankModel set, RerankTopK set",
			globalConfig: &config.Config{
				RerankModel: "rerank-model",
				RerankTopK:  50,
			},
			limit:    10,
			expected: 50,
		},
		{
			name: "RerankModel set, negative RerankTopK (uses limit)",
			globalConfig: &config.Config{
				RerankModel: "rerank-model",
				RerankTopK:  -5,
			},
			limit:    15,
			expected: 15,
		},
		{
			name: "RerankModel set, both limit and topK are 0 (defaults to 10)",
			globalConfig: &config.Config{
				RerankModel: "rerank-model",
				RerankTopK:  0,
			},
			limit:    0,
			expected: 10,
		},
		{
			name: "RerankModel set, negative limit (defaults to 10)",
			globalConfig: &config.Config{
				RerankModel: "rerank-model",
				RerankTopK:  0,
			},
			limit:    -1,
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original globalConfig
			originalConfig := globalConfig
			defer func() { globalConfig = originalConfig }()

			globalConfig = tt.globalConfig

			result := rerankTopK(tt.limit)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
