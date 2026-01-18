package parser

import (
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

func TestQueryCache_LoadQuery(t *testing.T) {
	cache := &QueryCache{
		queries: make(map[Language]*sitter.Query),
	}

	tests := []struct {
		name    string
		lang    Language
		wantErr bool
	}{
		// Query files that compile successfully
		{"Python", LanguagePython, false},     // Fixed - simplified query
		{"Go", LanguageGo, false},              // Working
		{"Java", LanguageJava, false},          // Working
		{"Rust", LanguageRust, false},          // Working
		{"C", LanguageC, false},                // Working
		{"Ruby", LanguageRuby, false},          // Working

		// All queries now compile successfully (fixed in this session)
		{"JavaScript", LanguageJavaScript, false}, // Fixed - removed async patterns
		{"TypeScript", LanguageTypeScript, false}, // Fixed - removed async patterns
		{"C++", LanguageCPP, false},            // Fixed - simplified namespace pattern
		{"PHP", LanguagePHP, false},            // Fixed - removed anonymous_function pattern
		{"Scala", LanguageScala, false},        // Fixed - removed case_modifier patterns
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create parser to get the tree-sitter language
			parser, err := NewParser(tt.lang)
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			// Load query
			query, err := cache.LoadQuery(tt.lang, parser.tsLanguage)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && query == nil {
				t.Error("LoadQuery() returned nil query without error")
			}
		})
	}
}

func TestQueryCache_Caching(t *testing.T) {
	cache := &QueryCache{
		queries: make(map[Language]*sitter.Query),
	}

	// Create parser
	parser, err := NewParser(LanguagePython)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// First load
	query1, err := cache.LoadQuery(LanguagePython, parser.tsLanguage)
	if err != nil {
		t.Fatalf("First LoadQuery() failed: %v", err)
	}

	// Check cache size
	if cache.CacheSize() != 1 {
		t.Errorf("CacheSize() = %d, want 1", cache.CacheSize())
	}

	// Second load (should be cached)
	query2, err := cache.LoadQuery(LanguagePython, parser.tsLanguage)
	if err != nil {
		t.Fatalf("Second LoadQuery() failed: %v", err)
	}

	// Should be the same query object
	if query1 != query2 {
		t.Error("LoadQuery() returned different query objects for same language")
	}

	// Cache size should still be 1
	if cache.CacheSize() != 1 {
		t.Errorf("CacheSize() = %d, want 1", cache.CacheSize())
	}
}

func TestQueryCache_ClearCache(t *testing.T) {
	cache := &QueryCache{
		queries: make(map[Language]*sitter.Query),
	}

	// Create parser and load query
	parser, err := NewParser(LanguagePython)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	_, err = cache.LoadQuery(LanguagePython, parser.tsLanguage)
	if err != nil {
		t.Fatalf("LoadQuery() failed: %v", err)
	}

	// Verify cache has entry
	if cache.CacheSize() != 1 {
		t.Errorf("CacheSize() = %d, want 1 before clear", cache.CacheSize())
	}

	// Clear cache
	cache.ClearCache()

	// Verify cache is empty
	if cache.CacheSize() != 0 {
		t.Errorf("CacheSize() = %d, want 0 after clear", cache.CacheSize())
	}
}

func TestQueryCache_InvalidLanguage(t *testing.T) {
	cache := &QueryCache{
		queries: make(map[Language]*sitter.Query),
	}

	// Create parser for a valid language
	parser, err := NewParser(LanguagePython)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Try to load query for unknown language
	_, err = cache.LoadQuery(LanguageUnknown, parser.tsLanguage)
	if err == nil {
		t.Error("LoadQuery() should fail for unknown language")
	}
}
