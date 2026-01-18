package parser

import (
	"embed"
	"fmt"
	"sync"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

//go:embed queries/*.scm
var queryFiles embed.FS

// QueryCache manages compiled tree-sitter queries with thread-safe caching
type QueryCache struct {
	queries map[Language]*sitter.Query
	mu      sync.RWMutex
}

// globalQueryCache is the singleton query cache instance
var globalQueryCache = &QueryCache{
	queries: make(map[Language]*sitter.Query),
}

// LoadQuery loads and compiles a tree-sitter query for the given language
// Returns cached query if already compiled, otherwise loads from embedded .scm file
func (qc *QueryCache) LoadQuery(lang Language, tsLang *sitter.Language) (*sitter.Query, error) {
	// Check cache first (read lock)
	qc.mu.RLock()
	if query, ok := qc.queries[lang]; ok {
		qc.mu.RUnlock()
		return query, nil
	}
	qc.mu.RUnlock()

	// Not in cache, load and compile (write lock)
	qc.mu.Lock()
	defer qc.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have loaded it)
	if query, ok := qc.queries[lang]; ok {
		return query, nil
	}

	// Load query file from embedded filesystem
	queryPath := fmt.Sprintf("queries/%s.scm", lang.String())
	queryContent, err := queryFiles.ReadFile(queryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read query file %s: %w", queryPath, err)
	}

	// Compile query
	query, err := sitter.NewQuery(tsLang, string(queryContent))
	if err != nil {
		return nil, fmt.Errorf("failed to compile query for %s: %w", lang.String(), err)
	}

	// Cache for future use
	qc.queries[lang] = query

	return query, nil
}

// ClearCache clears all cached queries (useful for testing)
func (qc *QueryCache) ClearCache() {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	qc.queries = make(map[Language]*sitter.Query)
}

// CacheSize returns the number of cached queries
func (qc *QueryCache) CacheSize() int {
	qc.mu.RLock()
	defer qc.mu.RUnlock()
	return len(qc.queries)
}
