# Vector Storage with LanceDB

## Overview

Code Scout uses [LanceDB](https://lancedb.com/) for vector storage and similarity search. LanceDB is a columnar vector database built on Apache Arrow and Parquet, optimized for fast ANN (Approximate Nearest Neighbor) search.

## Why LanceDB?

**Local-first**: Embedded database, no server required
**Fast**: Columnar storage optimized for vector similarity
**Efficient**: Arrow/Parquet compression reduces disk usage
**Simple**: File-based, no complex setup
**Go support**: Native Go bindings via lancedb-go

## Storage Location

```
.code-scout/
├── code_chunks.lance/          # LanceDB table directory
│   ├── data/
│   │   ├── 0.parquet           # Columnar data files
│   │   └── 1.parquet
│   └── _versions/
│       └── 1.manifest           # Version metadata
└── metadata.json                # Code Scout metadata
```

All data stays local in the `.code-scout/` directory.

## Schema Design

### Arrow Schema

```go
schema := arrow.NewSchema([]arrow.Field{
    {Name: "chunk_id", Type: arrow.BinaryTypes.String},
    {Name: "file_path", Type: arrow.BinaryTypes.String},
    {Name: "line_start", Type: arrow.PrimitiveTypes.Int32},
    {Name: "line_end", Type: arrow.PrimitiveTypes.Int32},
    {Name: "language", Type: arrow.BinaryTypes.String},
    {Name: "code", Type: arrow.BinaryTypes.LargeString},
    {Name: "chunk_type", Type: arrow.BinaryTypes.String, Nullable: true},
    {Name: "heading", Type: arrow.BinaryTypes.String, Nullable: true},
    {Name: "heading_level", Type: arrow.BinaryTypes.String, Nullable: true},
    {Name: "parent_heading", Type: arrow.BinaryTypes.String, Nullable: true},
    {Name: "embedding_type", Type: arrow.BinaryTypes.String},
    {Name: "vector", Type: arrow.FixedSizeListOf(3584, arrow.PrimitiveTypes.Float32)},
}, nil)
```

**Field Details**:
- `chunk_id`: UUID string
- `file_path`: Absolute path to source file
- `line_start`, `line_end`: 1-indexed line numbers
- `language`: "go", "python", "markdown", etc.
- `code`: The actual code or documentation content
- `chunk_type`: Semantic label (function, section, document, etc.)
- `heading` / `heading_level` / `parent_heading`: Markdown metadata for docs chunks
- `embedding_type`: Indicates whether the chunk used the code or docs embedding model
- `vector`: 3584-dimensional float32 array (embedding)

**Implementation**: internal/storage/lancedb.go:83-107

### Why This Schema?

**chunk_id**: Unique identifier for each chunk  
**file_path + line_start/end**: Navigate to source code  
**language**: Filter results by programming language  
**code**: Display in search results  
**chunk_type**: Helps differentiate functions, structs, markdown sections, etc.  
**heading metadata**: Provides docs context for CLI and downstream agents  
**embedding_type**: Indicates which embedding space the vector belongs to  
**vector**: Enables similarity search

## Data Insertion

### Arrow RecordBatch

LanceDB uses Arrow RecordBatch for efficient columnar insertion:

```go
func (s *LanceDBStore) StoreChunks(chunks []Chunk, embeddings [][]float64) error {
    pool := memory.NewGoAllocator()

    // Build arrays for each column
    chunkIDs := buildStringArray(pool, chunks, func(c Chunk) string { return c.ID })
    filePaths := buildStringArray(pool, chunks, func(c Chunk) string { return c.FilePath })
    lineStarts := buildInt32Array(pool, chunks, func(c Chunk) int32 { return int32(c.LineStart) })
    lineEnds := buildInt32Array(pool, chunks, func(c Chunk) int32 { return int32(c.LineEnd) })
    languages := buildStringArray(pool, chunks, func(c Chunk) string { return c.Language })
    codes := buildLargeStringArray(pool, chunks, func(c Chunk) string { return c.Code })
    vectors := buildVectorArray(pool, embeddings)

    // Create RecordBatch
    record := array.NewRecord(schema, []arrow.Array{
        chunkIDs, filePaths, lineStarts, lineEnds, languages, codes, vectors,
    }, int64(len(chunks)))
    defer record.Release()

    // Append to LanceDB table
    err := s.table.Add(ctx, record)
    return err
}
```

**Columnar Format**:
```
chunk_id:       [uuid1, uuid2, uuid3, ...]
file_path:      [path1, path2, path3, ...]
line_start:     [10, 45, 67, ...]
line_end:       [23, 58, 89, ...]
language:       [go, go, markdown, ...]
code:           [func..., type..., "## Heading", ...]
chunk_type:     [function, struct, section, ...]
heading:        ["Authentication", "", "Overview", ...]
heading_level:  ["2", "", "1", ...]
parent_heading: ["Architecture > Auth", "", "", ...]
embedding_type: [code, code, docs, ...]
vector:         [[0.1, 0.2, ...], [0.3, 0.4, ...], ...]
```

Benefits:
- Efficient storage (column compression)
- Fast columnar scans
- Optimized for vector operations

**Implementation**: internal/storage/lancedb.go:111-168

## Vector Search

### ANN Search Query

```go
func (s *LanceDBStore) Search(queryEmbedding []float64, limit int) ([]map[string]interface{}, error) {
    ctx := context.Background()

    // Convert query to float32
    queryVec := make([]float32, len(queryEmbedding))
    for i, v := range queryEmbedding {
        queryVec[i] = float32(v)
    }

    // Execute ANN search
    results, err := s.table.Search(ctx, queryVec).Limit(limit).Execute()
    if err != nil {
        return nil, err
    }

    // Convert Arrow records to maps
    return arrowToMaps(results)
}
```

**Search Process**:
1. Convert query embedding to float32
2. LanceDB finds K-nearest neighbors using IVF+PQ indexing
3. Returns records ordered by distance (ascending)

**Similarity Metric**: Cosine distance (default in LanceDB)

**Implementation**: internal/storage/lancedb.go:170-182

### Result Format

```json
{
  "chunk_id": "uuid-abc-123",
  "file_path": "/path/to/file.go",
  "line_start": 45,
  "line_end": 67,
  "language": "go",
  "code": "func Add(a, b int) int { return a + b }",
  "chunk_type": "function",
  "heading": "",
  "heading_level": "",
  "parent_heading": "",
  "embedding_type": "code",
  "_distance": 0.123
}
```

Documentation chunks populate the heading fields and set `embedding_type` to `docs`. `_distance` is still added by LanceDB.

### Schema Migration

Slice 3 adds the chunk metadata columns above. After pulling, delete the existing `.code-scout/code_chunks.lance` directory (or the entire `.code-scout/` folder) and re-run `code-scout index` so the LanceDB table is recreated with the new schema. Older tables without these columns cannot store markdown metadata.

## Incremental Updates

### Deletion by File Path

When a file is modified, old chunks must be deleted before adding new ones:

```go
func (s *LanceDBStore) DeleteChunksByFilePath(filePaths []string) error {
    ctx := context.Background()

    for _, path := range filePaths {
        // Delete all chunks with this file_path
        predicate := fmt.Sprintf("file_path = '%s'", path)
        err := s.table.Delete(ctx, predicate)
        if err != nil {
            return err
        }
    }

    return nil
}
```

**Deletion Flow** (for modified file):
1. Detect file modification (modtime changed)
2. Delete old chunks: `DELETE WHERE file_path = '/path/to/file.go'`
3. Re-chunk the modified file
4. Insert new chunks

**Implementation**: internal/storage/lancedb.go:149-165

### Metadata Tracking

Separate JSON file tracks which files are indexed:

```json
{
  "last_index_time": "2025-11-15T14:30:45Z",
  "file_mod_times": {
    "/path/to/file1.go": "2025-11-15T10:20:30Z",
    "/path/to/file2.go": "2025-11-14T16:45:12Z"
  }
}
```

**Load/Save**:
```go
func (s *LanceDBStore) LoadMetadata() (*Metadata, error) {
    data, err := os.ReadFile(s.dbPath + "/metadata.json")
    if os.IsNotExist(err) {
        return &Metadata{FileModTimes: make(map[string]time.Time)}, nil
    }
    var metadata Metadata
    json.Unmarshal(data, &metadata)
    return &metadata, nil
}

func (s *LanceDBStore) SaveMetadata(metadata *Metadata) error {
    data, err := json.MarshalIndent(metadata, "", "  ")
    return os.WriteFile(s.dbPath + "/metadata.json", data, 0644)
}
```

**Implementation**: internal/storage/metadata.go:19-65

## Performance

### Storage Efficiency

**Per-chunk overhead**:
- Metadata: ~100 bytes (IDs, paths, lines)
- Code: Variable (average ~500 bytes)
- Vector: 3584 × 4 = 14,336 bytes

**Total**: ~15 KB per chunk (uncompressed)

**With compression** (Parquet):
- Text data: 50-70% compression
- Vectors: Minimal compression (random floats)
- Typical: ~10-12 KB per chunk on disk

**Example**:
- 1000 chunks ≈ 10-12 MB
- 10,000 chunks ≈ 100-120 MB

### Query Performance

**ANN search** (K=10):
- Small database (<10K chunks): <10ms
- Medium database (10K-100K chunks): 10-50ms
- Large database (100K+ chunks): 50-200ms

**Factors**:
- Database size
- Query complexity
- Vector dimensions
- Index type (IVF-PQ, HNSW)

**In practice**: Sub-second search for typical codebases (<100K chunks)

## Database Operations

### Initialization

```go
// Create new database
store, err := storage.NewLanceDBStore("/path/to/repo")

// Create table (first time)
err = store.CreateTable()

// Open existing table (subsequent runs)
err = store.OpenTable()
```

### Lifecycle

```go
// 1. Create or open
store, _ := storage.NewLanceDBStore(".")
if tableExists {
    store.OpenTable()
} else {
    store.CreateTable()
}

// 2. Use
store.StoreChunks(chunks, embeddings)
results, _ := store.Search(queryEmbedding, 10)

// 3. Close
defer store.Close()
```

**Implementation**: internal/storage/lancedb.go:32-148

## Index Strategies

LanceDB supports multiple index types for ANN search:

### IVF (Inverted File Index)

Default for small-medium databases:
- Partitions vector space into clusters
- Searches only nearby clusters
- Fast, good recall

### HNSW (Hierarchical Navigable Small World)

Better for large databases:
- Graph-based index
- Higher memory usage
- Faster query time

**Current**: Code Scout uses default (IVF)
**Future**: Could add HNSW for large codebases

## Error Handling

### Database Corruption

```go
store, err := storage.NewLanceDBStore(".")
if err != nil {
    // Database corrupted or incompatible
    fmt.Println("Database corrupted. Delete .code-scout/ and re-index")
    return err
}
```

### Disk Space

```go
err := store.StoreChunks(chunks, embeddings)
if strings.Contains(err.Error(), "no space left") {
    return fmt.Errorf("disk full: cannot store chunks")
}
```

### Version Incompatibility

```go
// LanceDB version changed, schema incompatible
if err := store.OpenTable(); err != nil {
    return fmt.Errorf("table version incompatible, re-index required")
}
```

## Migration and Versioning

**Schema changes** require re-indexing:
- Adding/removing fields
- Changing vector dimensions
- Changing data types

**Process**:
1. Delete `.code-scout/` directory
2. Run `code-scout index` to rebuild

**Future**: Could add migration scripts for common changes

## Alternative Vector Databases

Code Scout could be adapted to use:

**Milvus**: Distributed vector database (for very large codebases)
**Qdrant**: High-performance Rust-based DB
**Weaviate**: GraphQL-based vector search
**Pinecone**: Cloud-based (requires API)
**pgvector**: PostgreSQL extension

To swap vector DB:
1. Implement storage interface in new file
2. Update table creation/query methods
3. Update schema translation
4. Re-index from scratch

See [extension-points.md](extension-points.md) for guide.

## Monitoring and Debugging

### Check Database Size

```bash
du -sh .code-scout/
```

### Inspect Table

```go
// Count chunks
count, err := table.Count(ctx)

// List all chunks (for debugging)
scanner := table.Scan(ctx)
for scanner.Next() {
    record := scanner.Record()
    // Inspect record
}
```

### Vacuum/Optimize

```go
// LanceDB auto-optimizes over time
// Manual optimization (future):
table.Optimize(ctx)
```

## Best Practices

1. **Close connections**: Use `defer store.Close()`
2. **Batch inserts**: Insert all chunks at once (not one-by-one)
3. **Delete before re-index**: Remove old chunks when file changes
4. **Track metadata**: Use metadata.json for incremental updates
5. **Handle errors**: Check for corruption, disk space, etc.
6. **Backup**: .code-scout/ directory is easily backed up
7. **Clean rebuilds**: Delete .code-scout/ for fresh start

## Future Optimizations

**Partitioning**: Separate tables per language
**Indexing**: Add HNSW index for large codebases
**Compression**: More aggressive vector quantization
**Caching**: In-memory cache for frequent queries
**Sharding**: Distribute across multiple tables for huge repos

---

LanceDB provides a solid foundation for Code Scout's vector storage needs: fast, local, and efficient.
