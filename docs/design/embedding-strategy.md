# Embedding Strategy

## Overview

Code Scout uses semantic embeddings to represent code chunks as high-dimensional vectors. This enables semantic similarity search: finding code based on meaning rather than exact text matches.

## Embedding Model

Code Scout uses a two-model embedding strategy tuned for code and documentation.

**Code model**: `nomic-ai/CodeRankEmbed`  
**Text model**: `nomic-ai/nomic-embed-text-v1.5`  
**Provider**: TEI via the OpenAI-compatible TEI wrapper  
**Default endpoint**: `http://localhost:11435`

### Why CodeRankEmbed + nomic-embed-text?

- **Code-specialized**: Trained on code in multiple languages (Python, Java, Ruby, PHP, JavaScript, Go)
- **Lightweight text model**: Fast and accurate for docs
- **Local**: Runs via TEI, no cloud dependencies
- **Open source**: Transparent model architecture

## API Integration

### OpenAI-Compatible Embeddings API

Code Scout calls an OpenAI-compatible `/v1/embeddings` endpoint (TEI wrapper by default):

**Request**:
```json
POST http://localhost:11435/v1/embeddings
Content-Type: application/json

{
  "model": "nomic-ai/CodeRankEmbed",
  "input": ["func Add(a, b int) int {\n    return a + b\n}"]
}
```

**Response**:
```json
{
  "data": [
    {
      "embedding": [0.12304688, -0.45605469, 0.78906250, ...]
    }
  ]
}
```

**Implementation**: internal/embeddings/client.go:151-206

### Client Implementation

```go
type OpenAIClient struct {
    endpoint string  // http://localhost:11435
    model    string  // nomic-ai/CodeRankEmbed
    client   *http.Client
}

func (c *OpenAIClient) Embed(text string) ([]float64, error) {
    // 1. Prepare request
    reqBody := openAIEmbedRequest{
        Model: c.model,
        Input: []string{text},
    }

    // 2. HTTP POST to embeddings endpoint
    url := c.endpoint + "/v1/embeddings"
    resp, err := c.client.Post(url, "application/json", data)

    // 3. Parse response
    var embedResp openAIEmbedResponse
    json.NewDecoder(resp.Body).Decode(&embedResp)

    return embedResp.Data[0].Embedding, nil
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
- Reduces embedding API calls by ~11% (128/1111 in this repo)
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

Embedding generation is I/O-bound (network calls to the embedding endpoint). Code Scout uses a worker pool to generate embeddings concurrently:

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
- ~100 chunks/minute (depends on embedding server)
- Progress reported every 50 embeddings

**Implementation**: cmd/code-scout/index.go:169-224

**Tuning**:
```bash
# Increase workers for faster embedding (if the embedding server can handle it)
code-scout index --workers 20

# Decrease workers if the embedding server is overloaded
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

### 1. Batch + Concurrent

Code Scout batches multiple chunks per request inside a worker pool:

```go
embeddings := Embed([]string{"chunk1", "chunk2", ...})
```

**Why batching?**
- Reduces HTTP overhead per embedding
- TEI supports multiple inputs per request
- Improves throughput on GPU-backed servers

**Why a worker pool?**
- Parallelism across batches
- Progress feedback per chunk
- Resilient to per-batch failures
- Tunable concurrency (`--workers`) and batch size (`--batch-size`)

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

### Embedding Endpoint Connection Errors

```go
embedding, err := client.Embed(text)
if err != nil {
    // Check if the TEI wrapper is running
    if strings.Contains(err.Error(), "connection refused") {
        return fmt.Errorf("Embeddings endpoint not running. Start the TEI wrapper")
    }
    return err
}
```

### Model Not Found

```go
if strings.Contains(err.Error(), "model not found") {
    return fmt.Errorf("Model 'nomic-ai/CodeRankEmbed' not found. Check .code-scout.json")
}
```

### Automatic Token Truncation

Code Scout automatically truncates text that exceeds the embedding model's token limit to prevent indexing failures.

**Default limits:**
- Maximum tokens: 8192 (TEI default for nomic-ai/CodeRankEmbed and nomic-embed-text-v1.5)
- Character limit: ~19000 chars (conservative estimate: 8192 tokens × 2.3 chars/token)

**Implementation:** [`internal/embeddings/client.go`](../../internal/embeddings/client.go)

```go
// Truncate text before sending to embedding API
func truncateText(text string) (string, bool) {
    if len(text) <= MaxChars {
        return text, false
    }
    truncated := text[:MaxChars]
    log.Printf("Warning: Text truncated from %d to %d chars to fit %d token limit",
               len(text), MaxChars, MaxTokens)
    return truncated, true
}
```

**Behavior:**
- Text under 19000 chars: Passed through unchanged
- Text over 19000 chars: Truncated with warning logged
- Prevents "Input validation error: inputs must have less than 8192 tokens" errors
- Maintains semantic quality for most code and documentation chunks

**When truncation occurs:**
- Large documentation files (>19K chars)
- Very large code functions or classes
- Files with extensive comments

**Note:** For models with larger context windows (e.g., 32K tokens), the limits can be adjusted in the embedding client constants.

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
1. Update `DefaultCodeModel` in internal/embeddings/client.go
2. Update embedding dimension in internal/storage/lancedb.go
3. Re-index from scratch (embeddings incompatible across models)

See [extension-points.md](extension-points.md) for detailed guide.

## Performance Benchmarks

**Indexing** (this repo, 1111 chunks):
- Content deduplication: 983 unique chunks (11.5% reduction)
- Workers: 10 concurrent
- Time: ~2 minutes (depends on embedding endpoint performance)
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

## Reranking

### Overview

Reranking is an optional second-stage that improves search precision by re-scoring the top-K results using a cross-encoder model.

**Why Rerank?**

Vector search finds results based on distance in embedding space, but this can miss nuances:
- Initial embedding is generated without seeing the query
- Bi-encoder models encode query and document separately
- Distance metrics may not perfectly capture relevance
- Top results may benefit from query-aware scoring

Reranking addresses this by:
1. Using a cross-encoder (reranker) model that jointly encodes query and result
2. Computing relevance scores that consider the relationship between query and result
3. Reordering based on these more precise cross-attention scores

**Cross-Encoder vs Bi-Encoder:**
- **Bi-encoder** (used for initial search): Encodes query and documents independently, fast but less precise
- **Cross-encoder** (used for reranking): Jointly encodes query+document pairs, slower but more accurate

### Configuration

**Config File** (`.code-scout.json` or `~/.code-scout/config.json`):
```json
{
  "endpoint": "http://localhost:11435",
  "code_model": "nomic-ai/CodeRankEmbed",
  "text_model": "nomic-ai/nomic-embed-text-v1.5",
  "rerank_model": "BAAI/bge-reranker-base",
  "rerank_top_k": 25
}
```

**Parameters**:
- `rerank_model`: Cross-encoder model name for reranking (optional, empty = disabled)
- `rerank_top_k`: Number of top results to rerank (optional, default = search limit)
- `endpoint`: Single endpoint for all operations (tei-wrapper handles routing to appropriate TEI instances)

**Deployment Architecture**:
- **tei-wrapper** manages both embedding and reranker TEI instances
- **Single endpoint**: Code Scout communicates only with tei-wrapper (default: `http://localhost:11435`)
- **Dual TEI instances**: tei-wrapper spawns embeddings TEI immediately and starts reranker TEI on-demand
- **Auto-routing**: tei-wrapper routes `/v1/embeddings` to embeddings TEI and `/rerank` to reranker TEI

**Dynamic Model Loading**:

Both embedding and reranker models support dynamic loading and switching:

1. **Configuration-Driven**: Models are configured in `.code-scout.json` (single source of truth)
2. **On-Demand Loading**: Reranker models load automatically on first use
3. **Runtime Switching**: Both embedding and reranker models can switch mid-session
4. **Priority Order**: Request model > Config file > CLI flag (deprecated) > Default

**Example**: Switching reranker models mid-session:
```bash
# Initial search uses model from config
code-scout search "authentication"  # Uses BAAI/bge-reranker-base

# Edit .code-scout.json to change rerank_model to BAAI/bge-reranker-large

# Next search automatically switches models
code-scout search "error handling"  # Uses BAAI/bge-reranker-large
```

The tei-wrapper handles all model lifecycle management (downloading, starting, stopping, switching) transparently.

**Model Selection**:
- **Recommended**: `BAAI/bge-reranker-base` (~278M params) - balanced speed and accuracy
- **Higher accuracy**: `BAAI/bge-reranker-large` (~560M params) - slower but more precise
- **Multilingual**: `BAAI/bge-reranker-v2-m3` (~568M params, 8192 token limit)
- **Fast/lightweight**: `cross-encoder/ms-marco-MiniLM-L-6-v2` (~22M params)
- **Speed-optimized**: `jinaai/jina-reranker-v1-turbo-en` (~137M params, 8192 token limit)

**Deployment**:
```bash
# Start tei-wrapper (manages both embedding and reranker TEI instances)
cd cmd/tei-wrapper
go build -o tei-wrapper .
./tei-wrapper \
  --port 11435 \
  --model nomic-ai/nomic-embed-text-v1.5

# tei-wrapper automatically:
# 1. Spawns TEI for embeddings on port 8080 (with hot-swapping)
# 2. Spawns TEI for reranker on port 8081 (on first /rerank request)
# 3. Exposes unified API on port 11435
```

See [RERANKER_SETUP.md](../guides/RERANKER_SETUP.md) for complete setup instructions.

### Reranking Algorithm

**Step 1: Context Building**

Each result is formatted with rich context:
```
File: internal/storage/lancedb.go
Language: go
Chunk: function

func Search(embedding []float64, limit int) ([]map[string]interface{}, error) {
    ...
}
```

This provides more signal than raw code alone.

**Step 2: Cross-Encoder Scoring**

```go
// Build context texts for each result
texts := make([]string, topK)
for i := 0; i < topK; i++ {
    texts[i] = buildRerankText(results[i])
}

// Use TEI /rerank endpoint with cross-encoder model
rerankResults, err := client.Rerank(query, texts)
```

The TEI `/rerank` endpoint:
- Takes a query and list of texts
- Uses a cross-encoder model (e.g., BAAI/bge-reranker-large)
- Returns relevance scores for each query+text pair
- Scores are already sorted by relevance (highest first)

**Request Format**:
```json
{
  "query": "What is Deep Learning?",
  "texts": ["Deep Learning is not...", "Deep learning is..."],
  "raw_scores": false
}
```

**Response Format**:
```json
[
  {"index": 1, "score": 0.9993},
  {"index": 0, "score": 0.2901}
]
```

**Step 3: Score Assignment**

```go
// Map rerank scores back to results
for _, rr := range rerankResults {
    if rr.Index >= 0 && rr.Index < topK {
        reranked[rr.Index] = results[rr.Index]
        reranked[rr.Index].RerankScore = &rr.Score
    }
}
```

**Rerank scores** (range: 0 to 1 for normalized scores):
- Higher scores = more relevant
- Scores reflect cross-attention between query and result
- More precise than distance-based vector similarity

**Step 4: Reordering**

```go
// Sort top-K by rerank_score (descending)
sort.SliceStable(reranked, func(i, j int) bool {
    return *reranked[i].RerankScore > *reranked[j].RerankScore
})

// Append remaining results unchanged
combined := append(reranked, results[topK:]...)
```

### Implementation

**Rerank Client** (internal/embeddings/reranker.go):
```go
type RerankClient struct {
    endpoint string
    apiKey   string
    model    string
    client   *http.Client
}

func (c *RerankClient) Rerank(query string, texts []string) ([]RerankResult, error) {
    reqBody := RerankRequest{
        Query:      query,
        Texts:      texts,
        RawScores:  false, // Use normalized scores
        ReturnText: false,
    }

    // POST to /rerank endpoint
    resp, err := c.client.Post(c.endpoint + "/rerank", "application/json", jsonData)
    // ... handle response

    var results []RerankResult
    json.NewDecoder(resp.Body).Decode(&results)
    return results, nil
}
```

**Factory Function** (cmd/code-scout/embeddings_factory.go):
```go
newRerankClient = func() *embeddings.RerankClient {
    if globalConfig != nil && globalConfig.RerankModel != "" {
        // Always use main endpoint - tei-wrapper handles routing to reranker
        return embeddings.NewRerankClient(
            globalConfig.Endpoint,
            globalConfig.APIKey,
            globalConfig.RerankModel,
        )
    }
    return nil
}
```

**Main Reranking Logic** (cmd/code-scout/search.go:336-383):
```go
func rerankResults(results []SearchResult, query string, topK int) ([]SearchResult, error) {
    if len(results) == 0 || topK <= 0 {
        return results, nil
    }

    client := newRerankClient()
    if client == nil {
        return results, nil
    }

    // Build context texts for each result
    texts := make([]string, topK)
    for i := 0; i < topK; i++ {
        texts[i] = buildRerankText(results[i])
    }

    // Use cross-encoder to rerank
    rerankResults, err := client.Rerank(query, texts)
    if err != nil {
        return nil, fmt.Errorf("failed to rerank results: %w", err)
    }

    // Create reranked slice with scores
    reranked := make([]SearchResult, topK)
    for _, rr := range rerankResults {
        if rr.Index >= 0 && rr.Index < topK {
            reranked[rr.Index] = results[rr.Index]
            reranked[rr.Index].RerankScore = &rr.Score
        }
    }

    // Sort by rerank score (highest first)
    sort.SliceStable(reranked, func(i, j int) bool {
        return *reranked[i].RerankScore > *reranked[j].RerankScore
    })

    return append(reranked, results[topK:]...), nil
}
```

### Performance

**Overhead**:
- Cross-encoder inference (batch of 10 query+text pairs): ~100-200ms
- Network request to TEI `/rerank` endpoint: ~10-20ms
- Score processing and sorting: <1ms
- **Total**: ~120-220ms for top-10 reranking

**Why Cross-Encoders are Slower**:
- Must process each query+text pair through transformer
- Cannot precompute like bi-encoders (query-dependent)
- O(n) complexity where n = number of candidates

**Optimization**:
- TEI batches reranking requests efficiently
- GPU acceleration recommended for cross-encoders
- Only rerank top-K (not all results)
- Use smaller rerank model if speed-critical (e.g., `bge-reranker-base` vs `bge-reranker-large`)

**Recommended Settings**:
- **High precision needed**: `rerank_top_k: 50` (rerank more, return fewer)
- **Balanced**: `rerank_top_k: 25` (default)
- **Speed priority**: Disable reranking (`rerank_model: ""`)

### Output Format

**Without Reranking**:
```
1. file.go:10-20 (score: 0.1234)
```

**With Reranking**:
```
1. file.go:10-20 (vector: 0.1234, rerank: 0.8560)
```

**JSON** includes `rerank` metadata:
```json
{
  "rerank": {
    "model": "BAAI/bge-reranker-large",
    "top_k": 25
  },
  "results": [
    {
      "score": 0.1234,
      "rerank_score": 0.8560,
      ...
    }
  ]
}
```

**Rerank score interpretation**:
- Vector score (0.0-1.0): L2 distance (lower is better, inverted for display)
- Rerank score (0.0-1.0): Cross-encoder relevance (higher is better)

### When to Use Reranking

**Use reranking when**:
- Precision is more important than speed
- Queries are complex or multi-faceted
- Top-1 accuracy matters (e.g., AI agent code generation)
- Results will be manually reviewed (worth the extra time)

**Skip reranking when**:
- Speed is critical (<1s response time)
- Retrieving many results for analysis
- Initial vector search is sufficient
- API quota/cost is a concern

## Best Practices

1. **Use semantic chunking**: Better chunks → better embeddings
2. **Enable deduplication**: Saves API calls and search noise
3. **Tune workers**: Match to embedding server capacity
4. **Incremental updates**: Only re-index changed files
5. **Monitor progress**: Watch embedding generation output
6. **Large context**: Use 32K model for full code files
7. **Consistent model**: Query and index with same model
8. **Enable reranking**: Improves top-K precision for critical queries

## Future Improvements

Potential enhancements:
- **Hybrid search**: Combine vector similarity with keyword matching
- **Fine-tuning**: Custom model trained on specific codebase
- **Multi-model**: Use different models for code vs. documentation
- **Caching**: Cache rerank embeddings for repeated queries
- **Learning to rank**: Train a dedicated reranking model
