package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/array"
	"github.com/apache/arrow/go/v17/arrow/memory"
	"github.com/jlanders/code-scout/internal/chunker"
	"github.com/lancedb/lancedb-go/pkg/contracts"
	"github.com/lancedb/lancedb-go/pkg/lancedb"
)

const (
	// DefaultDBDir is the default directory for LanceDB storage
	DefaultDBDir = ".code-scout"
	// DefaultTableName is the default table name for code chunks
	DefaultTableName = "code_chunks"
	// VectorDimension is the embedding dimension (nomic-embed-code uses 3584)
	VectorDimension = 3584
)

// LanceDBStore handles storage and retrieval from LanceDB
type LanceDBStore struct {
	conn   contracts.IConnection
	table  contracts.ITable
	schema *arrow.Schema
	dbDir  string
}

// NewLanceDBStore creates a new LanceDB store
func NewLanceDBStore(rootDir string) (*LanceDBStore, error) {
	dbDir := filepath.Join(rootDir, DefaultDBDir)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Connect to LanceDB
	ctx := context.Background()
	conn, err := lancedb.Connect(ctx, dbDir, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LanceDB: %w", err)
	}

	return &LanceDBStore{
		conn:  conn,
		dbDir: dbDir,
	}, nil
}

// StoreChunks stores chunks with their embeddings
func (s *LanceDBStore) StoreChunks(chunks []chunker.Chunk, embeddings [][]float64) error {
	if len(chunks) != len(embeddings) {
		return fmt.Errorf("chunks and embeddings length mismatch: %d vs %d", len(chunks), len(embeddings))
	}

	ctx := context.Background()

	// Define Arrow schema
	fields := []arrow.Field{
		{Name: "chunk_id", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "file_path", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "line_start", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
		{Name: "line_end", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
		{Name: "language", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "code", Type: arrow.BinaryTypes.String, Nullable: false},
		{Name: "vector", Type: arrow.FixedSizeListOf(VectorDimension, arrow.PrimitiveTypes.Float32), Nullable: false},
	}
	s.schema = arrow.NewSchema(fields, nil)

	// Convert to LanceDB schema
	lanceSchema, err := lancedb.NewSchema(s.schema)
	if err != nil {
		return fmt.Errorf("failed to create Lance schema: %w", err)
	}

	// Create table
	s.table, err = s.conn.CreateTable(ctx, DefaultTableName, lanceSchema)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Build Arrow arrays
	pool := memory.NewGoAllocator()

	// Prepare data arrays
	chunkIDs := make([]string, len(chunks))
	filePaths := make([]string, len(chunks))
	lineStarts := make([]int32, len(chunks))
	lineEnds := make([]int32, len(chunks))
	languages := make([]string, len(chunks))
	codes := make([]string, len(chunks))
	allVectors := make([]float32, len(chunks)*VectorDimension)

	for i, chunk := range chunks {
		chunkIDs[i] = chunk.ID
		filePaths[i] = chunk.FilePath
		lineStarts[i] = int32(chunk.LineStart)
		lineEnds[i] = int32(chunk.LineEnd)
		languages[i] = chunk.Language
		codes[i] = chunk.Code

		// Convert float64 embeddings to float32 and flatten
		for j, val := range embeddings[i] {
			allVectors[i*VectorDimension+j] = float32(val)
		}
	}

	// Build column arrays
	chunkIDBuilder := array.NewStringBuilder(pool)
	chunkIDBuilder.AppendValues(chunkIDs, nil)
	chunkIDArray := chunkIDBuilder.NewArray()
	defer chunkIDArray.Release()

	filePathBuilder := array.NewStringBuilder(pool)
	filePathBuilder.AppendValues(filePaths, nil)
	filePathArray := filePathBuilder.NewArray()
	defer filePathArray.Release()

	lineStartBuilder := array.NewInt32Builder(pool)
	lineStartBuilder.AppendValues(lineStarts, nil)
	lineStartArray := lineStartBuilder.NewArray()
	defer lineStartArray.Release()

	lineEndBuilder := array.NewInt32Builder(pool)
	lineEndBuilder.AppendValues(lineEnds, nil)
	lineEndArray := lineEndBuilder.NewArray()
	defer lineEndArray.Release()

	languageBuilder := array.NewStringBuilder(pool)
	languageBuilder.AppendValues(languages, nil)
	languageArray := languageBuilder.NewArray()
	defer languageArray.Release()

	codeBuilder := array.NewStringBuilder(pool)
	codeBuilder.AppendValues(codes, nil)
	codeArray := codeBuilder.NewArray()
	defer codeArray.Release()

	// Build vector array
	vectorFloat32Builder := array.NewFloat32Builder(pool)
	vectorFloat32Builder.AppendValues(allVectors, nil)
	vectorFloat32Array := vectorFloat32Builder.NewArray()
	defer vectorFloat32Array.Release()

	vectorListType := arrow.FixedSizeListOf(VectorDimension, arrow.PrimitiveTypes.Float32)
	vectorArray := array.NewFixedSizeListData(
		array.NewData(vectorListType, len(chunks), []*memory.Buffer{nil},
			[]arrow.ArrayData{vectorFloat32Array.Data()}, 0, 0),
	)
	defer vectorArray.Release()

	// Create record and insert
	columns := []arrow.Array{chunkIDArray, filePathArray, lineStartArray, lineEndArray, languageArray, codeArray, vectorArray}
	record := array.NewRecord(s.schema, columns, int64(len(chunks)))
	defer record.Release()

	if err := s.table.Add(ctx, record, nil); err != nil {
		return fmt.Errorf("failed to add records: %w", err)
	}

	return nil
}

// Search performs vector similarity search
func (s *LanceDBStore) Search(queryVector []float64, limit int) ([]map[string]interface{}, error) {
	if s.table == nil {
		return nil, fmt.Errorf("table not initialized; call StoreChunks first")
	}

	// Convert float64 query vector to float32
	queryVectorFloat32 := make([]float32, len(queryVector))
	for i, v := range queryVector {
		queryVectorFloat32[i] = float32(v)
	}

	ctx := context.Background()
	results, err := s.table.VectorSearch(ctx, "vector", queryVectorFloat32, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	return results, nil
}

// Close closes the database connection
func (s *LanceDBStore) Close() error {
	if s.table != nil {
		if err := s.table.Close(); err != nil {
			return fmt.Errorf("failed to close table: %w", err)
		}
	}
	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}
	return nil
}
