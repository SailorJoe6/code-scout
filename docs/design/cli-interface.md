# CLI Interface

## Overview

Code Scout provides a simple command-line interface with two main commands: `index` and `search`. Built with [Cobra](https://github.com/spf13/cobra), the CLI is designed for both human use and AI agent automation.

## Commands

### index

**Purpose**: Index a codebase for semantic search

**Usage**:
```bash
code-scout index [flags]
```

**Flags**:
- `--workers int` - Number of concurrent embedding workers (default: 10)

**Behavior**:
1. Scans current directory for code files
2. Detects new/modified files (incremental)
3. Chunks code with tree-sitter
4. Generates embeddings (with deduplication)
5. Stores in `.code-scout/` vector database

**Example Output**:
```
Indexing codebase...
Removing 2 changed/deleted file(s) from index...
Indexing 10 file(s) (8 go, 2 python)
  - cmd/main.go: 15 chunks
  - internal/parser/extractor.go: 45 chunks
  ...
Total chunks: 234
Generating embeddings...
Found 25 duplicate chunks (will skip 25 embeddings)
Using 10 concurrent workers for embedding generation
  Generated 50/209 unique embeddings (dim: 3584)
  Generated 100/209 unique embeddings (dim: 3584)
  ...
Copying embeddings to 25 duplicate chunks...
Embeddings generated successfully!
Storing in vector database...
✓ Indexing complete!
```

**Implementation**: cmd/code-scout/index.go

---

### search

**Purpose**: Search the indexed codebase semantically

**Usage**:
```bash
code-scout search [query] [flags]
```

**Arguments**:
- `query` - Search query (required)

**Flags**:
- `--json` - Output results as JSON (default: false)
- `--limit int` - Maximum number of results (default: 10)

**Human-Readable Output**:
```bash
$ code-scout search "error handling"

Found 5 unique results (from 10 total) for: error handling

1. internal/embeddings/ollama.go:61-63 (score: 3456.7891)
   Language: go
   if err != nil {
       return nil, fmt.Errorf("failed to make request to Ollama: %w", err)
   }

2. internal/storage/lancedb.go:42-44 (score: 3567.8912)
   Language: go
   if err := store.OpenTable(); err != nil {
       return fmt.Errorf("failed to open table: %w (have you run 'code-scout index' first?)", err)
   }

...
```

**JSON Output**:
```bash
$ code-scout search "error handling" --json

{
  "mode": "code",
  "query": "error handling",
  "total_results": 10,
  "returned": 5,
  "results": [
    {
      "chunk_id": "uuid-abc-123",
      "file_path": "/path/to/internal/embeddings/ollama.go",
      "line_start": 61,
      "line_end": 63,
      "language": "go",
      "code": "if err != nil {\n\treturn nil, fmt.Errorf(\"failed to make request to Ollama: %w\", err)\n}",
      "score": 3456.7891
    },
    ...
  ]
}
```

**Implementation**: cmd/code-scout/search.go

## Workflow Examples

### First-Time Setup

```bash
# 1. Navigate to repository
cd /path/to/my-repo

# 2. Index the codebase
code-scout index

# 3. Search
code-scout search "authentication logic"
```

### Incremental Updates

```bash
# After making code changes
code-scout index  # Only re-indexes changed files

# Search again
code-scout search "new feature"
```

### AI Agent Usage

```bash
# JSON output for programmatic parsing
code-scout search "database connection" --json | jq '.results[] | {file: .file_path, lines: "\(.line_start)-\(.line_end)"}'

# Higher limit for more context
code-scout search "error handling" --json --limit 20
```

### Performance Tuning

```bash
# Faster indexing (if Ollama can handle it)
code-scout index --workers 20

# Slower, lower resource usage
code-scout index --workers 5
```

## Integration with AI Agents

### JSON Output Format

Designed for easy parsing:

```json
{
  "mode": "code",
  "query": "...",
  "total_results": 15,
  "returned": 10,
  "results": [
    {
      "chunk_id": "...",
      "file_path": "...",
      "line_start": 123,
      "line_end": 145,
      "language": "go",
      "code": "...",
      "score": 1234.5678
    }
  ]
}
```

### Claude/Codex Integration

Example prompt:
```
I want to understand error handling in this codebase.

$ code-scout search "error handling" --json

[Results here...]

Based on these search results, explain the error handling patterns used.
```

AI agents can:
1. Run `code-scout search` to find relevant code
2. Parse JSON results
3. Read specific files at line ranges
4. Provide context-aware assistance

### CI/CD Integration

```bash
# In CI pipeline, check if index is up-to-date
code-scout index --dry-run  # Future feature

# Search for security issues
code-scout search "SQL injection" --json | analyze-security

# Find TODO items
code-scout search "TODO FIXME" --json
```

## Exit Codes

- `0` - Success
- `1` - General error
- Specific errors use `fmt.Errorf` for descriptive messages

## Error Messages

### Index Errors

**Ollama not running**:
```
Error: failed to generate embedding: Post "http://localhost:11434/api/embeddings": dial tcp 127.0.0.1:11434: connect: connection refused

Hint: Start Ollama with: ollama serve
```

**Model not found**:
```
Error: Ollama API returned status 404: model 'code-scout-code' not found

Hint: Create model with: ollama create code-scout-code -f ollama-models/code-scout-code.Modelfile
```

**Permission errors**:
```
Error: failed to create LanceDB store: permission denied

Hint: Check write permissions in current directory
```

### Search Errors

**Database not indexed**:
```
Error: failed to open table: table not found (have you run 'code-scout index' first?)

Hint: Run: code-scout index
```

**Empty database**:
```
Found 0 results for: your query

Hint: Database may be empty. Check .code-scout/ exists and contains data.
```

## Configuration

### Environment Variables

Currently none. Future possibilities:

```bash
# Future configuration
export CODE_SCOUT_DB_PATH=./custom-db
export CODE_SCOUT_OLLAMA_URL=http://remote-ollama:11434
export CODE_SCOUT_MODEL=custom-model
```

### Config File

Future: `.code-scout-config.json`
```json
{
  "ollama_endpoint": "http://localhost:11434",
  "model": "code-scout-code",
  "default_workers": 10,
  "default_limit": 10
}
```

## Shell Completions

Cobra supports shell completions:

```bash
# Bash
code-scout completion bash > /etc/bash_completion.d/code-scout

# Zsh
code-scout completion zsh > ~/.zsh/completion/_code-scout

# Fish
code-scout completion fish > ~/.config/fish/completions/code-scout.fish
```

## Future Commands

Potential additions:

**stats**: Show database statistics
```bash
code-scout stats
# Output: 1234 chunks, 856 files, last indexed: 2025-11-15
```

**clean**: Remove database
```bash
code-scout clean
# Removes .code-scout/ directory
```

**validate**: Check database health
```bash
code-scout validate
# Checks for corruption, missing files, etc.
```

**export**: Export database
```bash
code-scout export --format json > db.json
```

**import**: Import database
```bash
code-scout import db.json
```

## Best Practices

**For Humans**:
1. Run `index` after pulling code
2. Use descriptive search queries
3. Increase `--limit` for more results
4. Use `--json` for scripting

**For AI Agents**:
1. Always use `--json` flag
2. Parse results with jq or native JSON
3. Cache results when possible
4. Batch similar queries
5. Navigate to specific line numbers for context

## Debugging

**Verbose output** (future):
```bash
code-scout index --verbose
code-scout search "query" --debug
```

**Check database**:
```bash
ls -lh .code-scout/
du -sh .code-scout/
```

**View raw results** (no deduplication):
```bash
# Future flag
code-scout search "query" --no-dedup
```

---

The CLI is designed to be simple, predictable, and automation-friendly while still providing good UX for humans.
