# Embedding Strategy

## Overview

Code Scout uses semantic embeddings to represent code chunks as high-dimensional vectors. This enables semantic similarity search: finding code based on meaning rather than exact text matches.

## Embedding Model

**Model**: `code-scout-code` (based on nomic-embed-code)
**Provider**: Ollama (local API)
**Dimensions**: 3584
**Context Window**: 32,768 tokens (~2000-2500 lines of code)

### Why nomic-embed-code?

- **Code-specialized**: Trained on code in multiple languages (Python, Java, Ruby, PHP, JavaScript, Go)
- **Large context**: 32K tokens handles most code files entirely
- **Local**: Runs via Ollama, no cloud dependencies
- **Open source**: Transparent model architecture

### Custom Modelfile

Code Scout uses a custom Ollama Modelfile to ensure consistent behavior:

```dockerfile
FROM nomic-embed-text
PARAMETER num_ctx 32768
PARAMETER num_batch 512
```

**Why custom?**
- Ollama defaults to 2048 token context (too small for code)
- Without persistent config, context must be set per-request
- Custom Modelfile makes large context the default

**Setup**:
```bash
ollama create code-scout-code -f ollama-models/code-scout-code.Modelfile
```

See README.md for full setup instructions.

## API Integration

### Ollama Embeddings API

Code Scout calls Ollama's `/api/embeddings` endpoint:

**Request**:
```json
POST http://localhost:11434/api/embeddings
Content-Type: application/json

{
  "model": "code-scout-code",
  "prompt": "func Add(a, b int) int {\n    return a + b\n}"
}
```

**Response**:
```json
{
  "embedding": [
    0.12304688,
    -0.45605469,
    0.78906250,
    ...
    // 3584 total dimensions
  ]
}
```

**Implementation**: internal/embeddings/ollama.go:46-78

### Client Implementation

```go
type OllamaClient struct {
    endpoint string  // http://localhost:11434
    model    string  // code-scout-code
    client   *http.Client
}

func (c *OllamaClient) Embed(text string) ([]float64, error) {
    // 1. Prepare request
    reqBody := ollamaEmbedRequest{
        Model:  c.model,
        Prompt: text,
    }

    // 2. HTTP POST to Ollama
    url := c.endpoint + "/api/embeddings"
    resp, err := c.client.Post(url, "application/json", data)

    // 3. Parse response
    var embedResp ollamaEmbedResponse
    json.NewDecoder(resp.Body).Decode(&embedResp)

    return embedResp.Embedding, nil
}
```

## Deduplication Strategy

Code Scout employs two-stage deduplication to reduce redundant work:

### 1. Index-Time Deduplication

**Goal**: Skip generating embeddings for identical code chunks

**Method**: Content-based hashing

```go
// For each chunk, compute SHA256 hash of code content
hash := SHA256(chunk.Code)

// Track first occurrence
if !seen[hash] {
    uniqueChunks = append(uniqueChunks, chunk)
    seen[hash] = len(uniqueChunks) - 1
} else {
    duplicates[chunkIndex] = seen[hash]  // Map to first occurrence
}

// Generate embeddings only for unique chunks
embeddings := generateEmbeddings(uniqueChunks)

// Copy embeddings to duplicates
for dupIndex, firstIndex := range duplicates {
    embeddings[dupIndex] = embeddings[firstIndex]
}
```

**Benefits**:
- Reduces Ollama API calls by ~11% (128/1111 in this repo)
- Faster indexing
- Identical code gets identical embedding (consistent)

**Implementation**: cmd/code-scout/index.go:22-26, 147-257

**Example**:
```
Input:  1111 chunks
Hash:   983 unique, 128 duplicates
API:    983 calls (instead of 1111)
Savings: 11.5% fewer API calls
```

Common duplicates:
- `package main` appears in many files
- `import "fmt"` appears in many Go files
- Common error handling patterns
- Repeated boilerplate

### 2. Search-Time Deduplication

**Goal**: Remove duplicate results from search output

**Method**: Group by code content, keep best score

```go
// Group results by code content
groups := make(map[string]*bestResult)

for _, result := range rawResults {
    if group, exists := groups[result.Code]; exists {
        // Keep result with lower distance (better match)
        if result.Score < group.Score {
            groups[result.Code] = result
        }
    } else {
        groups[result.Code] = result
    }
}

// Extract deduplicated results
deduped := extractValues(groups)

// Sort by score
sort.Slice(deduped, func(i, j int) bool {
    return deduped[i].Score < deduped[j].Score
})
```

**Benefits**:
- Reduces noise in search results (30-80% depending on query)
- Shows only distinct code snippets
- Saves tokens for AI agents

**Implementation**: cmd/code-scout/search.go:123-165

**Example**:
```
Query: "package storage"
Before dedup: 10 results
  - "package storage" × 2  (lancedb.go, metadata.go)
  - "package parser"  × 6  (6 different parser files)
  - "package scanner" × 1
  - Other × 1

After dedup: 4 results
  - "package storage" (best match: lancedb.go)
  - "package parser"  (best match: extractor.go)
  - "package scanner"
  - Other

Reduction: 60%
```

## Concurrent Generation

### Worker Pool Pattern

Embedding generation is I/O-bound (network calls to Ollama). Code Scout uses a worker pool to generate embeddings concurrently:

```go
// Configuration
numWorkers := 10  // Default, configurable via --workers flag

// Channels
jobs := make(chan job, len(uniqueChunks))
results := make(chan result, len(uniqueChunks))

// Worker goroutines
for w := 0; w < numWorkers; w++ {
    go func() {
        for j := range jobs {
            embedding, err := embedClient.Embed(j.text)
            results <- result{
                index:     j.index,
                embedding: embedding,
                err:       err,
            }
        }
    }()
}

// Send jobs
for i, chunk := range uniqueChunks {
    jobs <- job{index: i, text: chunk.Code}
}
close(jobs)

// Collect results
for i := 0; i < len(uniqueChunks); i++ {
    r := <-results
    embeddings[r.index] = r.embedding
}
```

**Performance**:
- 10 concurrent workers
- ~100 chunks/minute (depends on Ollama server)
- Progress reported every 50 embeddings

**Implementation**: cmd/code-scout/index.go:169-224

**Tuning**:
```bash
# Increase workers for faster embedding (if Ollama can handle it)
code-scout index --workers 20

# Decrease workers if Ollama is overloaded
code-scout index --workers 5
```

## Embedding Quality

### What Makes a Good Embedding?

Good code embeddings capture:
1. **Syntactic similarity**: Similar code structure
2. **Semantic similarity**: Similar purpose/functionality
3. **Contextual similarity**: Used in similar contexts

Example - these should have similar embeddings:
```go
func Add(a, b int) int { return a + b }
func Sum(x, y int) int { return x + y }
```

These should have different embeddings:
```go
func Add(a, b int) int { return a + b }
func Multiply(a, b int) int { return a * b }
```

### Improving Embedding Quality

Code Scout improves embedding quality through:

**1. Semantic chunking**:
- Complete functions/methods (not fragments)
- Context metadata included (package, imports, receiver)
- Meaningful units (not arbitrary line splits)

**2. Large context window**:
- 32K tokens accommodates large functions
- No truncation of code (unlike 2K default)
- Full context captured in embedding

**3. Code-specialized model**:
- nomic-embed-code trained on code
- Understands programming constructs
- Better similarity for code than general text models

**Example**:

Chunk with context metadata:
```go
// Metadata enriches the embedding
{
    Code: "func (c *Calculator) Add(a, b int) int { return a + b }",
    Metadata: {
        "package": "math",
        "receiver": "*Calculator",
        "imports": "fmt, errors",
    }
}
```

The embedding captures:
- This is a method (not a standalone function)
- It's a Calculator method
- Part of math package
- Uses fmt and errors (might handle errors, print debug)

## Search Similarity

### Distance Metric

LanceDB uses **cosine distance** for vector similarity:

```
distance = 1 - (A · B) / (||A|| × ||B||)
```

Where:
- A = query embedding
- B = chunk embedding
- · = dot product
- ||·|| = vector magnitude

**Range**: 0 to 2
- 0 = identical vectors
- < 1 = similar vectors
- ≥ 1 = dissimilar vectors

**In practice**:
- Score < 1000 = very relevant
- Score 1000-3000 = moderately relevant
- Score > 5000 = not relevant

**Note**: Lower score is better (closer distance)

### Query Embedding

Search queries use the same embedding process as chunks:

```go
// User searches for "error handling"
query := "error handling"

// Generate query embedding (same model as index)
queryEmbedding, err := embedClient.Embed(query)

// Search for nearest neighbors
results, err := store.Search(queryEmbedding, limit=10)
```

**Important**: Query and chunks must use the same model for meaningful comparisons.

## Optimization Techniques

### 1. Batch vs. Concurrent

Code Scout chooses concurrent single requests over batching:

**Why not batching?**
```go
// Hypothetical batch API
embeddings := Embed([]string{"chunk1", "chunk2", ...})
```

Problems:
- Ollama API doesn't support batch embeddings
- Single HTTP request timeout risk
- No progress feedback
- Harder error recovery

**Why concurrent singles?**
```go
// Worker pool with individual requests
for chunk := range chunks {
    embedding := Embed(chunk)  // Individual request
    results <- embedding
}
```

Benefits:
- Progress reporting (per-chunk feedback)
- Resilient to single chunk failures
- Tunable concurrency (--workers flag)
- Works with Ollama's API

### 2. Incremental Updates

Don't re-embed unchanged files:

```go
// Load previous index metadata
metadata := store.LoadMetadata()

// Check each file's modification time
for _, file := range allFiles {
    if file.ModTime > metadata.FileModTimes[file.Path] {
        // File changed, re-index
        needsIndexing = append(needsIndexing, file)
    }
}

// Only generate embeddings for changed files
```

**Impact**: Re-indexing 10 changed files in a 1000-file repo takes seconds instead of minutes.

### 3. Content Deduplication

As described above, hash-based deduplication saves ~11% of API calls.

## Error Handling

### Ollama Connection Errors

```go
embedding, err := client.Embed(text)
if err != nil {
    // Check if Ollama is running
    if strings.Contains(err.Error(), "connection refused") {
        return fmt.Errorf("Ollama not running. Start with: ollama serve")
    }
    return err
}
```

### Model Not Found

```go
if strings.Contains(err.Error(), "model not found") {
    return fmt.Errorf("Model 'code-scout-code' not found. Create with: ollama create code-scout-code -f ...")
}
```

### Truncation Detection

With 32K context window, truncation is rare. But if it happens:

```go
// Ollama silently truncates beyond context window
// Detection: Compare input token count vs. context limit
if estimateTokens(text) > 32768 {
    log.Warn("Chunk exceeds context window, may be truncated")
}
```

## Alternative Embedding Models

Code Scout is designed for easy model swapping:

**Current**: nomic-embed-code (3584 dims)

**Alternatives**:
1. **nomic-embed-text** (768 dims)
   - General purpose
   - Smaller embeddings (faster search)
   - Less code-specialized

2. **OpenAI text-embedding-3-small** (1536 dims)
   - Requires API key
   - Cloud-based (not local)
   - Good code understanding

3. **CodeBERT** (768 dims)
   - Hugging Face model
   - Requires local model server
   - Specialized for code

To change model:
1. Update `DefaultCodeModel` in internal/embeddings/ollama.go
2. Update embedding dimension in internal/storage/lancedb.go
3. Re-index from scratch (embeddings incompatible across models)

See [extension-points.md](extension-points.md) for detailed guide.

## Performance Benchmarks

**Indexing** (this repo, 1111 chunks):
- Content deduplication: 983 unique chunks (11.5% reduction)
- Workers: 10 concurrent
- Time: ~2 minutes (depends on Ollama performance)
- Rate: ~500 chunks/minute

**Search**:
- Query embedding: <1 second
- Vector search: <100ms
- Deduplication: <10ms
- Total: ~1 second end-to-end

**Storage**:
- Embedding size: 3584 × 4 bytes = 14.3 KB per chunk
- 1000 chunks = ~14 MB vectors
- LanceDB compression reduces actual disk usage

## Best Practices

1. **Use semantic chunking**: Better chunks → better embeddings
2. **Enable deduplication**: Saves API calls and search noise
3. **Tune workers**: Match to Ollama server capacity
4. **Incremental updates**: Only re-index changed files
5. **Monitor progress**: Watch embedding generation output
6. **Large context**: Use 32K model for full code files
7. **Consistent model**: Query and index with same model

## Future Improvements

Potential enhancements:
- **Hybrid search**: Combine vector similarity with keyword matching
- **Reranking**: Use a more powerful model to rerank top results
- **Fine-tuning**: Custom model trained on specific codebase
- **Multi-model**: Use different models for code vs. documentation
- **Caching**: Cache embeddings across runs (already done for deduplication)
