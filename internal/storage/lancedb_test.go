package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jlanders/code-scout/internal/chunker"
)

// createTempDir creates a temporary directory for testing
func createTempDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "lancedb-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})
	return tmpDir
}

// TestNewLanceDBStore tests store creation
func TestNewLanceDBStore(t *testing.T) {
	tmpDir := createTempDir(t)

	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	if store.conn == nil {
		t.Error("Connection should not be nil")
	}

	// Verify directory was created
	dbDir := filepath.Join(tmpDir, DefaultDBDir)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		t.Errorf("Database directory was not created: %s", dbDir)
	}
}

// TestStoreChunksEmpty tests storing empty chunks
func TestStoreChunksEmpty(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Storing empty chunks should not error
	err = store.StoreChunks([]chunker.Chunk{}, [][]float64{})
	if err != nil {
		t.Errorf("StoreChunks with empty data should not error: %v", err)
	}
}

// TestStoreChunksMismatch tests mismatched chunk/embedding counts
func TestStoreChunksMismatch(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	chunks := []chunker.Chunk{
		{
			ID:            "test-1",
			FilePath:      "/test/file.go",
			LineStart:     1,
			LineEnd:       10,
			Language:      "go",
			Code:          "func test() {}",
			EmbeddingType: "code",
		},
	}
	embeddings := [][]float64{
		make([]float64, VectorDimension),
		make([]float64, VectorDimension), // Extra embedding
	}

	err = store.StoreChunks(chunks, embeddings)
	if err == nil {
		t.Error("Expected error for mismatched chunk and embedding counts")
	}
}

// TestStoreAndRetrieveChunks tests basic store and search
func TestStoreAndRetrieveChunks(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Create test chunks
	chunks := []chunker.Chunk{
		{
			ID:            "chunk-1",
			FilePath:      "/test/file1.go",
			LineStart:     1,
			LineEnd:       10,
			Language:      "go",
			Code:          "func testFunction() { return 42 }",
			ChunkType:     "function",
			EmbeddingType: "code",
		},
		{
			ID:            "chunk-2",
			FilePath:      "/test/file2.md",
			LineStart:     1,
			LineEnd:       5,
			Language:      "markdown",
			Code:          "# Test Documentation\n\nThis is a test.",
			ChunkType:     "section",
			EmbeddingType: "docs",
			Metadata: map[string]string{
				"heading":       "Test Documentation",
				"heading_level": "1",
			},
		},
	}

	// Create embeddings (use small vectors for first VectorDimension elements)
	embeddings := make([][]float64, len(chunks))
	for i := range embeddings {
		embeddings[i] = make([]float64, VectorDimension)
		// Set some distinctive values
		embeddings[i][0] = float64(i) * 0.1
		embeddings[i][1] = float64(i) * 0.2
	}

	// Store chunks
	err = store.StoreChunks(chunks, embeddings)
	if err != nil {
		t.Fatalf("StoreChunks failed: %v", err)
	}

	// Search for chunks
	queryVector := make([]float64, VectorDimension)
	queryVector[0] = 0.05 // Close to first chunk
	queryVector[1] = 0.10

	results, err := store.Search(queryVector, 10, "")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one result from search")
	}

	// Verify we can retrieve chunk data
	if len(results) > 0 {
		firstResult := results[0]
		if firstResult["chunk_id"] == nil {
			t.Error("Result should contain chunk_id")
		}
		if firstResult["file_path"] == nil {
			t.Error("Result should contain file_path")
		}
		if firstResult["code"] == nil {
			t.Error("Result should contain code")
		}
	}
}

// TestDeleteChunksByFilePath tests deletion functionality
func TestDeleteChunksByFilePath(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Store test chunks
	chunks := []chunker.Chunk{
		{
			ID:            "chunk-1",
			FilePath:      "/test/file1.go",
			LineStart:     1,
			LineEnd:       10,
			Language:      "go",
			Code:          "func test1() {}",
			EmbeddingType: "code",
		},
		{
			ID:            "chunk-2",
			FilePath:      "/test/file2.go",
			LineStart:     1,
			LineEnd:       10,
			Language:      "go",
			Code:          "func test2() {}",
			EmbeddingType: "code",
		},
	}

	embeddings := make([][]float64, len(chunks))
	for i := range embeddings {
		embeddings[i] = make([]float64, VectorDimension)
		embeddings[i][0] = float64(i)
	}

	err = store.StoreChunks(chunks, embeddings)
	if err != nil {
		t.Fatalf("StoreChunks failed: %v", err)
	}

	// Delete chunks from file1.go
	err = store.DeleteChunksByFilePath([]string{"/test/file1.go"})
	if err != nil {
		t.Fatalf("DeleteChunksByFilePath failed: %v", err)
	}

	// Search should now only return file2.go chunks
	queryVector := make([]float64, VectorDimension)
	results, err := store.Search(queryVector, 10, "")
	if err != nil {
		t.Fatalf("Search after delete failed: %v", err)
	}

	// Check that file1.go is not in results
	for _, result := range results {
		if filePath, ok := result["file_path"].(string); ok {
			if filePath == "/test/file1.go" {
				t.Error("Deleted file path should not appear in results")
			}
		}
	}
}

// TestDeleteChunksEmpty tests delete with empty paths
func TestDeleteChunksEmpty(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Delete with empty paths should not error
	err = store.DeleteChunksByFilePath([]string{})
	if err != nil {
		t.Errorf("DeleteChunksByFilePath with empty paths should not error: %v", err)
	}
}

// TestDeleteChunksNonExistentTable tests delete when table doesn't exist
func TestDeleteChunksNonExistentTable(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Delete from non-existent table should not error
	err = store.DeleteChunksByFilePath([]string{"/test/file.go"})
	if err != nil {
		t.Errorf("DeleteChunksByFilePath on non-existent table should not error: %v", err)
	}
}

// TestDeleteChunksWithQuotes tests deletion with file paths containing quotes
func TestDeleteChunksWithQuotes(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Store chunk with single quote in path
	chunks := []chunker.Chunk{
		{
			ID:            "chunk-1",
			FilePath:      "/test/file's.go", // Single quote in path
			LineStart:     1,
			LineEnd:       10,
			Language:      "go",
			Code:          "func test() {}",
			EmbeddingType: "code",
		},
	}

	embeddings := [][]float64{make([]float64, VectorDimension)}
	err = store.StoreChunks(chunks, embeddings)
	if err != nil {
		t.Fatalf("StoreChunks failed: %v", err)
	}

	// Delete should handle quote escaping
	err = store.DeleteChunksByFilePath([]string{"/test/file's.go"})
	if err != nil {
		t.Fatalf("DeleteChunksByFilePath with quotes failed: %v", err)
	}
}

// TestSearchWithFilter tests filtered search
func TestSearchWithFilter(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Store chunks with different embedding types
	chunks := []chunker.Chunk{
		{
			ID:            "chunk-1",
			FilePath:      "/test/code.go",
			LineStart:     1,
			LineEnd:       10,
			Language:      "go",
			Code:          "func test() {}",
			EmbeddingType: "code",
		},
		{
			ID:            "chunk-2",
			FilePath:      "/test/doc.md",
			LineStart:     1,
			LineEnd:       5,
			Language:      "markdown",
			Code:          "# Documentation",
			EmbeddingType: "docs",
		},
	}

	embeddings := make([][]float64, len(chunks))
	for i := range embeddings {
		embeddings[i] = make([]float64, VectorDimension)
		embeddings[i][0] = float64(i)
	}

	err = store.StoreChunks(chunks, embeddings)
	if err != nil {
		t.Fatalf("StoreChunks failed: %v", err)
	}

	// Search with filter for code only
	queryVector := make([]float64, VectorDimension)
	results, err := store.Search(queryVector, 10, "embedding_type = 'code'")
	if err != nil {
		t.Fatalf("Filtered search failed: %v", err)
	}

	// Verify all results are code type
	for _, result := range results {
		if embType, ok := result["embedding_type"].(string); ok {
			if embType != "code" {
				t.Errorf("Expected only 'code' embedding_type in filtered results, got: %s", embType)
			}
		}
	}
}

// TestSearchUninitializedTable tests search before table initialization
func TestSearchUninitializedTable(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Search without storing anything first
	queryVector := make([]float64, VectorDimension)
	_, err = store.Search(queryVector, 10, "")
	if err == nil {
		t.Error("Expected error when searching uninitialized table")
	}
}

// TestOpenTable tests opening an existing table
func TestOpenTable(t *testing.T) {
	tmpDir := createTempDir(t)

	// Create and populate a table
	store1, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}

	chunks := []chunker.Chunk{
		{
			ID:            "chunk-1",
			FilePath:      "/test/file.go",
			LineStart:     1,
			LineEnd:       10,
			Language:      "go",
			Code:          "func test() {}",
			EmbeddingType: "code",
		},
	}
	embeddings := [][]float64{make([]float64, VectorDimension)}
	err = store1.StoreChunks(chunks, embeddings)
	if err != nil {
		t.Fatalf("StoreChunks failed: %v", err)
	}
	store1.Close()

	// Open existing table in new store instance
	store2, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store2.Close()

	err = store2.OpenTable()
	if err != nil {
		t.Fatalf("OpenTable failed: %v", err)
	}

	// Should be able to search the existing data
	queryVector := make([]float64, VectorDimension)
	results, err := store2.Search(queryVector, 10, "")
	if err != nil {
		t.Fatalf("Search on reopened table failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected to find chunks in reopened table")
	}
}

// TestMultipleBatches tests storing chunks in multiple batches
func TestMultipleBatches(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Store first batch
	chunks1 := []chunker.Chunk{
		{
			ID:            "chunk-1",
			FilePath:      "/test/file1.go",
			LineStart:     1,
			LineEnd:       10,
			Language:      "go",
			Code:          "func test1() {}",
			EmbeddingType: "code",
		},
	}
	embeddings1 := [][]float64{make([]float64, VectorDimension)}
	err = store.StoreChunks(chunks1, embeddings1)
	if err != nil {
		t.Fatalf("First StoreChunks failed: %v", err)
	}

	// Store second batch
	chunks2 := []chunker.Chunk{
		{
			ID:            "chunk-2",
			FilePath:      "/test/file2.go",
			LineStart:     1,
			LineEnd:       10,
			Language:      "go",
			Code:          "func test2() {}",
			EmbeddingType: "code",
		},
	}
	embeddings2 := [][]float64{make([]float64, VectorDimension)}
	err = store.StoreChunks(chunks2, embeddings2)
	if err != nil {
		t.Fatalf("Second StoreChunks failed: %v", err)
	}

	// Search should find both chunks
	queryVector := make([]float64, VectorDimension)
	results, err := store.Search(queryVector, 10, "")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("Expected at least 2 results, got %d", len(results))
	}
}

// TestChunkMetadata tests storing and retrieving chunk metadata
func TestChunkMetadata(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Store chunk with metadata
	chunks := []chunker.Chunk{
		{
			ID:            "chunk-1",
			FilePath:      "/test/doc.md",
			LineStart:     1,
			LineEnd:       10,
			Language:      "markdown",
			Code:          "# Test Heading\n\nContent here",
			ChunkType:     "section",
			EmbeddingType: "docs",
			Metadata: map[string]string{
				"heading":        "Test Heading",
				"heading_level":  "1",
				"parent_heading": "Parent",
			},
		},
	}

	embeddings := [][]float64{make([]float64, VectorDimension)}
	err = store.StoreChunks(chunks, embeddings)
	if err != nil {
		t.Fatalf("StoreChunks failed: %v", err)
	}

	// Search and verify metadata
	queryVector := make([]float64, VectorDimension)
	results, err := store.Search(queryVector, 10, "")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	result := results[0]
	if result["heading"] != "Test Heading" {
		t.Errorf("Expected heading 'Test Heading', got: %v", result["heading"])
	}
	if result["heading_level"] != "1" {
		t.Errorf("Expected heading_level '1', got: %v", result["heading_level"])
	}
	if result["parent_heading"] != "Parent" {
		t.Errorf("Expected parent_heading 'Parent', got: %v", result["parent_heading"])
	}
}

// TestCloseStore tests closing the store
func TestCloseStore(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}

	// Store some data to initialize table
	chunks := []chunker.Chunk{
		{
			ID:            "chunk-1",
			FilePath:      "/test/file.go",
			LineStart:     1,
			LineEnd:       10,
			Language:      "go",
			Code:          "func test() {}",
			EmbeddingType: "code",
		},
	}
	embeddings := [][]float64{make([]float64, VectorDimension)}
	err = store.StoreChunks(chunks, embeddings)
	if err != nil {
		t.Fatalf("StoreChunks failed: %v", err)
	}

	// Close should not error
	err = store.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Closing again should not panic
	err = store.Close()
	if err != nil {
		t.Errorf("Second Close failed: %v", err)
	}
}

// TestVectorPadding tests that vectors shorter than VectorDimension are padded
func TestVectorPadding(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Store chunk
	chunks := []chunker.Chunk{
		{
			ID:            "chunk-1",
			FilePath:      "/test/file.go",
			LineStart:     1,
			LineEnd:       10,
			Language:      "go",
			Code:          "func test() {}",
			EmbeddingType: "code",
		},
	}
	embeddings := [][]float64{make([]float64, VectorDimension)}
	err = store.StoreChunks(chunks, embeddings)
	if err != nil {
		t.Fatalf("StoreChunks failed: %v", err)
	}

	// Search with shorter query vector
	shortQueryVector := make([]float64, 10) // Much shorter than VectorDimension
	_, err = store.Search(shortQueryVector, 10, "")
	if err != nil {
		t.Fatalf("Search with short vector should work (with padding): %v", err)
	}
}

// TestLoadMetadata_NewFile tests loading metadata when file doesn't exist
func TestLoadMetadata_NewFile(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Load metadata when file doesn't exist
	metadata, err := store.LoadMetadata()
	if err != nil {
		t.Fatalf("LoadMetadata should not error on missing file: %v", err)
	}

	// Should return empty metadata
	if metadata == nil {
		t.Fatal("Expected non-nil metadata")
	}

	if !metadata.LastIndexTime.IsZero() {
		t.Error("Expected zero LastIndexTime for new metadata")
	}

	if metadata.FileModTimes == nil {
		t.Error("Expected non-nil FileModTimes map")
	}

	if len(metadata.FileModTimes) != 0 {
		t.Errorf("Expected empty FileModTimes, got %d entries", len(metadata.FileModTimes))
	}
}

// TestSaveAndLoadMetadata tests saving and loading metadata
func TestSaveAndLoadMetadata(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Create test metadata
	now := time.Now()
	metadata := &IndexMetadata{
		LastIndexTime: now,
		FileModTimes: map[string]time.Time{
			"/test/file1.go": now.Add(-1 * time.Hour),
			"/test/file2.py": now.Add(-2 * time.Hour),
		},
	}

	// Save metadata
	err = store.SaveMetadata(metadata)
	if err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	// Verify file was created
	metadataPath := filepath.Join(store.dbDir, metadataFileName)
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Error("Metadata file was not created")
	}

	// Load metadata back
	loaded, err := store.LoadMetadata()
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}

	// Verify loaded data matches original
	if !loaded.LastIndexTime.Equal(metadata.LastIndexTime) {
		t.Errorf("LastIndexTime mismatch: expected %v, got %v", metadata.LastIndexTime, loaded.LastIndexTime)
	}

	if len(loaded.FileModTimes) != len(metadata.FileModTimes) {
		t.Errorf("FileModTimes count mismatch: expected %d, got %d", len(metadata.FileModTimes), len(loaded.FileModTimes))
	}

	for path, expectedTime := range metadata.FileModTimes {
		loadedTime, ok := loaded.FileModTimes[path]
		if !ok {
			t.Errorf("File path %s not found in loaded metadata", path)
			continue
		}
		// Compare with some tolerance for JSON marshaling precision
		if loadedTime.Unix() != expectedTime.Unix() {
			t.Errorf("ModTime mismatch for %s: expected %v, got %v", path, expectedTime, loadedTime)
		}
	}
}

// TestLoadMetadata_CorruptedFile tests loading corrupted metadata
func TestLoadMetadata_CorruptedFile(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Write invalid JSON to metadata file
	metadataPath := filepath.Join(store.dbDir, metadataFileName)
	err = os.WriteFile(metadataPath, []byte("invalid json {{{"), 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted metadata: %v", err)
	}

	// Load should return error
	_, err = store.LoadMetadata()
	if err == nil {
		t.Error("Expected error when loading corrupted metadata")
	}
}

// TestSaveMetadata_NilFileModTimes tests saving metadata with nil map
func TestSaveMetadata_NilFileModTimes(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Create metadata with nil FileModTimes
	metadata := &IndexMetadata{
		LastIndexTime: time.Now(),
		FileModTimes:  nil, // Explicitly nil
	}

	// Save should not error
	err = store.SaveMetadata(metadata)
	if err != nil {
		t.Fatalf("SaveMetadata with nil FileModTimes failed: %v", err)
	}

	// Load it back
	loaded, err := store.LoadMetadata()
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}

	// Should have non-nil map even if saved as nil
	if loaded.FileModTimes == nil {
		t.Error("Expected non-nil FileModTimes after loading")
	}
}

// TestNewLanceDBStore_InvalidPath tests store creation with invalid path
func TestNewLanceDBStore_InvalidPath(t *testing.T) {
	// Try to create store in a path that can't be created (null byte in path)
	_, err := NewLanceDBStore("/tmp/invalid\x00path")
	if err == nil {
		t.Error("Expected error when creating store with invalid path")
	}
}

// TestSaveMetadata_ReadOnlyDir tests saving to read-only directory
func TestSaveMetadata_ReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Test cannot run as root (permissions are not enforced)")
	}

	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}
	defer store.Close()

	// Make db directory read-only
	dbDir := filepath.Join(tmpDir, DefaultDBDir)
	if err := os.Chmod(dbDir, 0444); err != nil {
		t.Fatalf("Failed to chmod directory: %v", err)
	}
	defer os.Chmod(dbDir, 0755) // Restore permissions for cleanup

	// Save should fail due to permissions
	metadata := &IndexMetadata{
		LastIndexTime: time.Now(),
		FileModTimes:  make(map[string]time.Time),
	}
	err = store.SaveMetadata(metadata)
	if err == nil {
		t.Error("Expected error when saving to read-only directory")
	}
}

// TestCloseWithNilTable tests closing when table is nil
func TestCloseWithNilTable(t *testing.T) {
	tmpDir := createTempDir(t)
	store, err := NewLanceDBStore(tmpDir)
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}

	// Store has nil table since we never initialized it
	if store.table != nil {
		t.Skip("Table is not nil, test assumptions invalid")
	}

	// Close should work even with nil table
	err = store.Close()
	if err != nil {
		t.Errorf("Close with nil table should not error: %v", err)
	}
}
