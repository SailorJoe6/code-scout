# Ollama Setup for Code Scout

This guide covers setting up Ollama for Code Scout. Ollama is the **simplest option** for running embedding models locally, but requires reduced concurrency settings for Code Scout's two-pass architecture.

## Why Ollama?

**Ollama is the best choice when you prioritize simplicity over performance:**

- ✅ **Dead simple** - Binary install, no compilation
- ✅ **Cross-platform** - Works on M1/M2/M3, Intel Macs, Linux, Windows
- ✅ **Automatic model switching** - No need to run multiple instances
- ✅ **Fast model loading** - 1-2 seconds
- ✅ **Clean CLI** - Easy model management

**Trade-offs:**
- ⚠️ **Can't handle high concurrency** - Must use `--workers 2 --batch-size 2`
- ⚠️ **Slower indexing** - 2-3x slower than TEI on large repos
- ⚠️ **Not optimized for embeddings** - General-purpose model server

## When to Use Ollama vs TEI

| Use Ollama When... | Use TEI When... |
|-------------------|----------------|
| You want simplest setup | You need maximum performance |
| Small repos (<100 files) | Large repos (>500 files) |
| Development/testing | Production use |
| You're on any platform | You're on Apple Silicon |
| You value simplicity | You value speed |

See [TEI_SETUP.md](TEI_SETUP.md) for the faster TEI option on M2/Apple Silicon.

## Installation

### macOS (Apple Silicon or Intel)

```bash
# Download and install
brew install ollama

# Or download directly from https://ollama.com/download
```

### Linux

```bash
curl -fsSL https://ollama.com/install.sh | sh
```

### Windows

Download installer from https://ollama.com/download

### Verify Installation

```bash
ollama --version
```

## Pull Embedding Models

Ollama needs to download the models before first use:

```bash
# Pull code embedding model
ollama pull nomic-embed-text

# Verify models are installed
ollama list
```

**Note:** Ollama uses `nomic-embed-text` for both code and documentation embeddings. While not optimal (the specialized `CodeRankEmbed` performs better on code), it's simpler and works well enough for most use cases.

**Model sizes:**
- `nomic-embed-text:latest` - ~274MB download

## Configure Code Scout for Ollama

Ollama runs on `http://localhost:11434` by default, which is Code Scout's default endpoint.

### Option 1: Use Defaults (Recommended)

No configuration needed! Code Scout's defaults match Ollama:

```bash
# No config file needed, just run:
./code-scout index --workers 2 --batch-size 2
```

### Option 2: Explicit Configuration

Create `.code-scout.json` in your repo or `~/.code-scout/config.json` globally:

```json
{
  "endpoint": "http://localhost:11434",
  "code_model": "nomic-embed-text",
  "text_model": "nomic-embed-text"
}
```

**Note:** Using the same model for both passes is fine. Ollama will automatically switch between requests.

## Critical: Reduce Concurrency

**You MUST reduce concurrency when using Ollama** due to its model-switching overhead:

```bash
# CORRECT - Low concurrency
./code-scout index --workers 2 --batch-size 2

# WRONG - Will be very slow or fail
./code-scout index --workers 10 --batch-size 10
```

### Why Reduce Concurrency?

Ollama handles Code Scout's two-pass architecture by:
1. Load code model
2. Process code chunks
3. **Unload code model, load text model** ← Slow!
4. Process docs chunks

With high concurrency, Ollama thrashes between model loads, causing:
- Extremely slow indexing (10x+ slower)
- High memory usage
- Possible crashes or timeouts

**Recommended settings:**
- Small repos (<100 files): `--workers 2 --batch-size 2`
- Medium repos (100-500 files): `--workers 2 --batch-size 4`
- Large repos (>500 files): Consider TEI instead

## Usage

### Start Ollama Service

```bash
# Start Ollama (runs in background)
ollama serve

# Verify it's running
curl http://localhost:11434/api/version
```

**Note:** On macOS with the Ollama app, the service starts automatically. You don't need `ollama serve`.

### Index Repository with Ollama

```bash
# Navigate to your repo
cd /path/to/your/repo

# Index with reduced concurrency
./code-scout index --workers 2 --batch-size 2
```

**Progress output:**
```
Indexing codebase...
Indexing 150 file(s) (80 go, 70 markdown)
Total chunks: 2500
Code chunks: 850, Docs chunks: 1650

Pass 1: Generating code embeddings...
Using 2 concurrent workers
[█████████████████████████████] 850/850 (100%)

Pass 2: Generating docs embeddings...
Using 2 concurrent workers
[█████████████████████████████] 1650/1650 (100%)

Indexed 2500 chunks in 5m 32s
```

### Search Repository

```bash
# Semantic search (uses code model)
./code-scout search "authentication middleware"

# Search code only
./code-scout search "parse JSON" --mode code

# Search docs only
./code-scout search "installation guide" --mode docs
```

## Performance Tuning

### Small Repos (< 50 files, < 5K chunks)

```bash
--workers 2 --batch-size 2
```

**Expected time:** 2-5 minutes

### Medium Repos (50-200 files, 5K-20K chunks)

```bash
--workers 2 --batch-size 4
```

**Expected time:** 10-20 minutes

### Large Repos (> 200 files, > 20K chunks)

```bash
--workers 2 --batch-size 4
```

**Expected time:** 30-60+ minutes

**Recommendation:** For large repos, consider switching to TEI for 3-4x faster indexing.

## Troubleshooting

### Ollama service not running

**Error:** `failed to connect to http://localhost:11434`

**Solution:**

```bash
# Start Ollama service
ollama serve

# Or on macOS, launch Ollama app
open -a Ollama
```

### Model not found

**Error:** `model nomic-embed-text not found`

**Solution:**

```bash
# Pull the model
ollama pull nomic-embed-text

# Verify it's available
ollama list
```

### Indexing is extremely slow

**Likely cause:** Concurrency too high

**Solution:**

```bash
# Reduce concurrency
./code-scout index --workers 2 --batch-size 2
```

### Out of memory errors

**Likely cause:** Batch size too large

**Solution:**

```bash
# Reduce batch size
./code-scout index --workers 2 --batch-size 1
```

### Model switching thrashing

**Symptoms:** Indexing gets slower over time, high CPU usage

**Solution:** This is expected behavior with Ollama. Consider:
1. Reduce concurrency further: `--workers 1 --batch-size 1`
2. Switch to TEI if on M2 (see [TEI_SETUP.md](TEI_SETUP.md))

## Comparison: Ollama vs TEI

| Feature | Ollama | TEI (M2) |
|---------|--------|----------|
| **Setup** | ✅ Easy (brew install) | ⚠️ Moderate (cargo build) |
| **Platforms** | ✅ All (Mac/Linux/Windows) | ⚠️ M2 only |
| **Startup** | ✅ 1-2 seconds | ✅ 2-3 seconds |
| **Model Switching** | ✅ Automatic | ❌ Manual (2 instances) |
| **Concurrency** | ❌ Low (2 workers) | ✅ High (6-10 workers) |
| **Indexing Speed** | ⚠️ Slow | ✅ Fast (3-4x) |
| **Memory** | ✅ ~524MB | ✅ ~524MB |
| **Use Case** | Dev/testing/small repos | Production/large repos |

## Advanced: Custom Models

### Use Different Models for Code vs Docs

Ollama supports custom models via Modelfile. You could create optimized models:

```bash
# Create custom code model (future enhancement)
cat > Modelfile.code <<EOF
FROM nomic-embed-text
PARAMETER temperature 0.1
EOF

ollama create code-scout-code -f Modelfile.code

# Update config
{
  "code_model": "code-scout-code",
  "text_model": "nomic-embed-text"
}
```

**Note:** This is experimental and not currently recommended.

### Run Ollama on Different Port

```bash
# Set custom port
OLLAMA_HOST=0.0.0.0:8080 ollama serve

# Update Code Scout config
{
  "endpoint": "http://localhost:8080"
}
```

## Performance Benchmarks

### Indexing Times (M2 MacBook Pro)

**Small repo (~50 files, ~5K chunks):**
- Ollama (--workers 2 --batch-size 2): ~5-7 minutes
- TEI (--workers 6 --batch-size 6): ~2-3 minutes
- **TEI is 2x faster**

**Medium repo (~150 files, ~15K chunks):**
- Ollama (--workers 2 --batch-size 2): ~15-20 minutes
- TEI (--workers 6 --batch-size 6): ~5-7 minutes
- **TEI is 3x faster**

**Large repo (~500 files, ~50K chunks):**
- Ollama (--workers 2 --batch-size 2): ~60-90 minutes
- TEI (--workers 6 --batch-size 6): ~20-30 minutes
- **TEI is 3-4x faster**

### Memory Usage

Both Ollama and TEI use similar memory (~524MB for models), but:
- Ollama has slightly higher overhead due to model switching
- TEI keeps both models loaded (more consistent memory usage)

## Next Steps

- For better performance on M2, see [TEI_SETUP.md](TEI_SETUP.md)
- For cloud hosting, see README.md configuration section
- For contributing, see [DEVELOPERS.md](../DEVELOPERS.md)

## Summary

**Ollama is great for:**
- Getting started quickly
- Small to medium repos
- Development and testing
- Any platform (not just M2)

**Switch to TEI when:**
- You have a large codebase (>500 files)
- You need fast indexing (3-4x speedup)
- You're on Apple Silicon M2
- You're comfortable with Rust compilation

Both options produce identical search results - the only difference is indexing speed.
