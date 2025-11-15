# **Code Scout** - The codebase context engine for AI

*A dual-pass, multi-model embedding system for giving LLM agents full comprehension of a codebase.*

## **1. Goal**

Build a self-hostable, open-source system that provides LLM agents with **deep understanding of a codebase** by embedding:

* **Code files** (functions, classes, modules)
* **Documentation files** (Markdown, text, READMEs, guides)

into a unified vector database, using **domain-optimized embedding models** for each type.

The system supports RAG workflows, agentic coding tools, and natural-language querying against both docs and code simultaneously.

---

## **2. Key Insight**

No single embedding model can optimally represent both **source code semantics** and **natural language documentation**.

**Solution:**
Perform embedding in **two independent passes** using **two separate models**, then unify all embeddings in a single vector DB namespace:

1. **Code Embedding Pass**

   * Load a *code-optimized* embedding model
   * Chunk & embed only code files
   * Write vectors + metadata to DB

2. **Docs Embedding Pass**

   * Load a *text-optimized* embedding model
   * Chunk & embed only documentation files
   * Write vectors + metadata to same DB

LLM agents can later choose the correct embedding model for each query type (code question vs. documentation question) or leave it unspecified and get insights form both!

---

## **3. System Architecture**

### **3.1 High-Level Diagram**

```
                  ┌─────────────────────┐
                  │   Source Codebase   │
                  └─────────┬───────────┘
                            │
                 Two-Pass Embedding Pipeline
                            │
        ┌───────────────────┴─────────────────────┐
        │                                         │
┌──────────────────────┐                   ┌──────────────────────┐
│  Code Embedding      │                   │  Docs Embedding      │
│  (API: Code Model)   │                   │  (API: Text Model)   │
└───────────┬──────────┘                   └──────────┬───────────┘
            │                                         │
    ┌───────▼────────┐                      ┌────────▼────────────┐
    │ Code Chunks    │                      │ Docs Chunks         │
    │ + Metadata     │                      │ + Metadata          │
    └───────┬────────┘                      └────────┬────────────┘
            │                                        │
      ┌─────▼────────────────────────────────────────▼──────┐
      │              LanceDB Vector Database                │
      │         (Both Embedding Spaces Stored Together)     │
      │              (.code-scout/vectors.lance)            │
      └───────────────────┬─────────────────────────────────┘
                          │
                    Query Engine
                          │
               ┌──────────▼───────────┐
               │   Dual Retrieval     │
               │ (Code + Docs Models) │
               └──────────┬───────────┘
                          │
                  LLM / Coding Agent
```

---

## **4. Why This Design Works**

### **4.1 Code Requires Different Embeddings**

Code semantics are structured and hierarchical. A model trained on code learns:

* AST-like patterns
* function and class boundaries
* variable naming relationships
* multi-file dependencies

Natural-language models don't capture these well.

### **4.2 Docs Require Strong Natural-Language Semantics**

Markdown and text require:

* semantic understanding
* summarization
* conceptual linking

Code models underperform here.

### **4.3 Unified Vector Space (Metadata Layers)**

Although embeddings differ, the system unifies them at:

* **storage level** (same DB)
* **metadata level** (file path, repo, chunk type, language, code_regions, headings)
* **retrieval orchestration level** (dual-query strategy)

We do *not* require vector spaces to be numerically aligned.

---

## **5. Components**

### **5.1 File Classifier**

Detects file types:

* Code files (py, ts, js, cpp, go, rs, java, etc.)
* Docs (md, txt, rst)

### **5.2 Chunker**

Two strategies:

#### Code Chunking:

* Break by function / class / module
* Include import graph metadata
* Include surrounding context window

#### Docs Chunking:

* Break by markdown headers (H1/H2/H3)
* Keep sections semantically coherent
* Add heading + subheading metadata

### **5.3 Two-Pass Embedding Engine**

#### Pass A: Code Files

* Load code model via configured embedding API
* Embed only chunks tagged `type=code`
* Store as `embedding_type="code"`

#### Pass B: Documentation Files

* Load text/markdown model via configured embedding API
* Embed only chunks tagged `type=docs`
* Store as `embedding_type="docs"`

### **5.4 Unified Vector DB**

Stores:

* vector
* chunk text
* embedding_type
* file path
* repo name
* last_modified
* language
* chunk_id
* semantic summary (optional)

### **5.5 Retrieval Engine**

When queries come in:

1. Classify query as **code-oriented**, **doc-oriented**, or **hybrid**
2. Use correct embedding model to generate query vector
3. Retrieve against *all* stored vectors
4. Merge/sort results
5. Provide unified context to LLM agent

---

## **6. Example Workflow**

### **Initial Ingestion**

1. Scan repo
2. Identify code + docs
3. Chunk code
4. Chunk docs
5. Embed code with code model
6. Embed docs with docs model
7. Upload all vectors into DB

### **Query Flow**

1. User asks:

   > “How does the request parser work?”
2. Query is hybrid (code + docs)
3. Run two query embeddings
4. Retrieve top K from both spaces
5. Merge results
6. Agent synthesizes final answer

---

## **7. Advantages**

### ✔ Maximum embedding quality

Use the *best* model for each domain.

### ✔ Flexible embedding provider support

Self-hosted via Ollama (recommended) or cloud APIs (OpenAI, Cohere, etc.).

### ✔ Faster ingestion than running a huge single model

Two smaller models beat one giant “jack of all trades.”

### ✔ Better RAG precision & recall

Clear separation of semantic spaces.

### ✔ Perfect for agentic coding tools

Can answer:

* “Where is this function defined?”
* “What does the authentication flow look like?”
* “Explain the architecture across both code and docs.”

---

## **8. Implementation Plan (Elephant Carpaccio Style)**

Each slice delivers a **working, shippable system** with incrementally increasing capability. Every slice can be tested end-to-end immediately.

### **Slice 1: Simplest Possible End-to-End** ⭐ **MVP START**

**Deliverable**: Index a single Python file and search it.

**Features:**
- CLI: `code-scout index` and `code-scout search "query" --json`
- Hardcoded config (local Ollama, code-scout-code model)
- Single language: Python only
- Naive chunking: Split by blank lines (no Tree-sitter yet)
- LanceDB: Store chunks with embeddings
- Search: Code-only mode (no docs yet)
- JSON output with file path and line numbers

**Why valuable**: Proves the entire architecture works. AI agents can immediately use it on Python codebases.

**Success criteria**:
```bash
cd some-python-project/
code-scout index
code-scout search "authentication function" --json
# Returns relevant Python functions with locations
```

---

### **Slice 2: Semantic Code Chunking**

**Deliverable**: Proper function/class-level chunking with Tree-sitter.

**Features:**
- Tree-sitter integration for Python
- Extract functions, classes, methods as chunks
- Include docstrings and context metadata
- Better relevance in search results

**Why valuable**: Much more precise search results. Functions are complete semantic units.

**Success criteria**: Search returns individual functions, not arbitrary text blocks.

---

### **Slice 3a: Documentation Indexing**

**Deliverable**: Add markdown documentation indexing using the existing code embedding model.

**Features:**
- Scan for `.md`, `.txt`, `.rst` files
- Markdown header-based chunking (H1/H2/H3)
- Index docs using existing code-scout-code model (reuse current architecture)
- Basic search includes both code and docs

**Why valuable**: Proves the chunking and indexing pipeline works for non-code files. Provides immediate value by making docs searchable alongside code.

**Success criteria**:
```bash
code-scout index  # now indexes both .py, .go, and .md files
code-scout search "architecture overview"  # finds README sections
code-scout search "auth implementation"  # finds both code and docs
```

---

### **Slice 3b: Dual-Model Architecture** ⭐ **DUAL-MODEL SYSTEM COMPLETE**

**Deliverable**: Add second embedding model for documentation and implement search modes.

**Features:**
- Second embedding model (code-scout-text) for documentation
- Two-pass embedding pipeline (code model for .py/.go, text model for .md/.txt)
- Search modes: `--code`, `--docs`, `--hybrid` (default)
- Dual-query retrieval logic (query both models, merge results)

**Why valuable**: Completes the dual-model vision. Code and documentation get optimized embeddings from domain-specific models, significantly improving search quality.

**Success criteria**:
```bash
code-scout search "architecture overview" --docs  # uses text model only
code-scout search "parse_config function" --code  # uses code model only
code-scout search "how does auth work"  # hybrid queries both models
```

---

### **Slice 4: Multi-Language Support**

**Deliverable**: Support Go and JavaScript/TypeScript.

**Features:**
- Tree-sitter grammars for Go, JS, TS
- Language detection by file extension
- Language-specific query files
- Test on multi-language repos

**Why valuable**: Works on real-world polyglot codebases.

**Success criteria**: Index and search a project with Python, Go, and JavaScript files.

---

### **Slice 5: Configuration & Provider Flexibility**

**Deliverable**: Config file support and multiple embedding providers.

**Features:**
- `.code-scout/config.yml` for project-specific settings
- Support OpenAI, Cohere, HuggingFace APIs (not just Ollama)
- API key management via environment variables
- Model selection per provider
- Error handling and validation

**Why valuable**: Works in environments without local Ollama. Production-ready configuration.

**Success criteria**: Use OpenAI embeddings instead of Ollama via config file.

---

### **Slice 6: Production Hardening**

**Deliverable**: Robust error handling, better UX, performance.

**Features:**
- Incremental updates (re-index only changed files)
- Progress indicators for long-running operations
- Better error messages and logging
- `--limit`, `--files` flags for filtering
- Relevance score thresholds

**Why valuable**: Professional-grade tool ready for daily use.

---

### **Slice 7: Extended Language Support**

**Deliverable**: Add remaining languages (Java, Rust, C/C++, Ruby, PHP, C#, Kotlin).

**Features:**
- Tree-sitter queries for each language
- Language-specific metadata extraction
- Test suite for each language

**Why valuable**: Comprehensive language support.

---

### **Future Enhancements** (Post-MVP)

- Batch query API
- Query result caching
- MCP server integration
- Git-aware indexing (respect .gitignore)
- Parallel embedding for large repos
- Web UI dashboard (optional, low priority)

---

## **8.1 MVP Scope (Slices 1-3b)**

**Definition**: The Minimum Viable Product consists of **Slices 1-3b**, delivering a complete end-to-end system with dual-model embedding.

**What's included in MVP:**
- ✅ `code-scout index` - Full repository indexing
- ✅ `code-scout search "query" --code/--docs/--hybrid` - All three search modes
- ✅ JSON output for AI agent consumption
- ✅ Python language support (Slice 1-2)
- ✅ Markdown documentation support (Slice 3)
- ✅ Tree-sitter based semantic chunking for code
- ✅ Header-based chunking for docs
- ✅ LanceDB vector storage (per-project `.code-scout/` directory)
- ✅ Dual embedding models (code + text)
- ✅ Hardcoded Ollama configuration (simplest setup)

**What's NOT in MVP (comes in later slices):**
- ❌ Config file support (Slice 5)
- ❌ Multiple embedding providers (Slice 5)
- ❌ Multi-language beyond Python (Slice 4+)
- ❌ Incremental updates (Slice 6)
- ❌ Advanced filtering/limiting (Slice 6)
- ❌ Web UI (future enhancement)

**MVP Success Criteria:**

An AI coding agent (like Claude Code) can:
1. Index a Python project with documentation
2. Search for code implementations: `code-scout search "function name" --code --json`
3. Search for architectural context: `code-scout search "design decisions" --docs --json`
4. Get comprehensive answers: `code-scout search "how feature X works" --json`
5. Receive structured JSON with file paths, line numbers, and relevance scores
6. Navigate directly to relevant code locations

**Time to value**: Slice 1 delivers value in **hours**, complete MVP (Slice 3) in **days**, not weeks.

---

## **9. Selected Embedding Models**

### **✅ Chosen: Code Model**

**nomic-embed-code** (via custom Ollama model: `code-scout-code`)

* **Context Window**: 32,768 tokens (~2,000-2,500 lines of code)
* **Supported Languages**: Python, Java, Ruby, PHP, JavaScript, Go
* **Size**: 7.5 GB
* **Why**: State-of-the-art code embeddings, massive context window for large files, optimized for code-to-code similarity

### **✅ Chosen: Documentation Model**

**nomic-embed-text** (via custom Ollama model: `code-scout-text`)

* **Context Window**: 8,192 tokens (~500-600 lines)
* **Size**: 274 MB
* **Why**: Excellent general-purpose text embeddings, long-context support, perfect for mixed code+documentation content

### **Custom Ollama Configuration**

Both models are deployed as custom Ollama models with persistent context window settings:

```bash
# Custom models created from Modelfiles in ollama-models/
ollama create code-scout-text -f ollama-models/nomic-embed-text.Modelfile
ollama create code-scout-code -f ollama-models/nomic-embed-code.Modelfile
```

**Benefits**:
- Persistent context window configuration (no runtime parameter management)
- Prevents silent truncation from Ollama's default 2048 token limit
- Optimized for maximum context usage during embedding generation
- Simpler client code - no need to specify `num_ctx` per request

### **Model Selection Rationale**

1. **Open Source & Self-Hosted**: Both models run locally via Ollama with no external API dependencies
2. **Proven Performance**: nomic-embed-text outperforms OpenAI's ada-002; nomic-embed-code outperforms Voyage Code 3
3. **Context Window Size**: 32K for code handles large files without chunking; 8K for docs is ample for most documentation
4. **Lightweight**: Combined 7.8 GB fits easily on consumer hardware
5. **Active Development**: Nomic AI actively maintains both models (2025)

---

## **10. Implementation Decisions**

The following architectural decisions have been finalized for the implementation:

### **10.1 Technology Stack**

**Implementation Language: Go**
- Single binary distribution (zero runtime dependencies)
- Fast startup and low memory footprint
- Excellent CLI tooling ecosystem
- Strong HTTP client libraries for API integration
- Native Tree-sitter bindings via `github.com/tree-sitter/go-tree-sitter`

**Vector Database: LanceDB**
- Lightweight, file-based columnar storage (Apache Arrow)
- Per-project embedded database (`.code-scout/` directory)
- 10-20MB overhead vs. 50-100MB for alternatives
- Built specifically for vector workloads
- Zero-copy reads for performance
- Go client: `lancedb/lancedb-go`

**Rationale**: LanceDB's lightweight footprint is critical for the per-project model where developers may have dozens of indexed codebases.

### **10.2 Deployment Architecture**

**Per-Project Embedded Model:**
```
project-root/
├── .code-scout/
│   ├── vectors.lance       # LanceDB vector storage
│   ├── config.yml          # Project-specific config
│   └── metadata.json       # Indexing metadata
├── .git/
├── .beads/
└── src/
```

**Key characteristics:**
- Each codebase has its own isolated vector database
- No centralized server infrastructure
- `.code-scout/` added to `.gitignore` (developers re-index locally)
- Database files stay with the project they describe
- Zero configuration conflicts between projects

### **10.3 Embedding API Architecture**

**Provider-Agnostic Network-Based Design:**

The system makes **no hard dependency on Ollama**. Instead, it uses a configuration-driven approach supporting any embedding provider:

```yaml
# .code-scout/config.yml
embeddings:
  code:
    endpoint: "http://localhost:11434"  # or any URL
    model: "code-scout-code"
    api_format: "ollama"  # or "openai", "cohere", etc.

  docs:
    endpoint: "https://api.openai.com/v1"
    model: "text-embedding-3-small"
    api_format: "openai"
    api_key: "${OPENAI_API_KEY}"
```

**Supported configurations:**
- ✅ Self-hosted Ollama (recommended, provided Modelfiles)
- ✅ OpenAI API
- ✅ Cohere API
- ✅ HuggingFace Inference Endpoints
- ✅ Any OpenAI-compatible endpoint

**Benefits:**
- Users choose between self-hosted (free, private) and cloud (easier setup)
- No vendor lock-in
- Flexible deployment options for different environments
- Ollama Modelfiles provided as the recommended self-hosted path

### **10.4 Language Support Strategy**

**Broad Multi-Language Support from Day One:**

Target languages (10+):
- Python, Go, JavaScript/TypeScript
- Java, Rust, C/C++
- Ruby, PHP, C#, Kotlin

**Implementation approach:**
- **Tree-sitter** for unified AST parsing across all languages
- Per-language query files (~50 lines each) to extract functions/classes
- Language-agnostic chunking framework
- Incremental rollout: Start with Python/Go/JS, add others via query files

**Why Tree-sitter:**
- Consistent API across 50+ languages
- Fast, incremental parsing
- Error-tolerant (handles incomplete code)
- Battle-tested (used by GitHub, Neovim, etc.)
- Go bindings available: `github.com/tree-sitter/go-tree-sitter`

### **10.5 Code Parsing & Chunking**

**Parsing Strategy:**
- Tree-sitter AST parsing for all languages
- Language-specific S-expressions queries to extract:
  - Functions/methods
  - Classes/structs/interfaces
  - Top-level declarations
  - Import/export statements
  - Docstrings/comments

**Chunking Strategy: Hybrid Semantic Approach**

**Preferred: Semantic boundaries**
- Extract complete functions/classes as chunks
- Include surrounding context (imports, parent class)
- Preserve docstrings and comments
- Metadata: file, line range, language, chunk type

**Fallback: Intelligent splitting**
- For functions exceeding context window (rare with 32K)
- Split at logical block boundaries (if/else, loops)
- Maintain overlap between chunks
- Preserve semantic coherence

**Chunk metadata schema:**
```json
{
  "chunk_id": "uuid",
  "type": "function|class|method|module",
  "name": "parse_config",
  "file_path": "src/config/parser.py",
  "line_start": 42,
  "line_end": 67,
  "language": "python",
  "embedding_type": "code",
  "imports": ["json", "pathlib"],
  "parent_class": "ConfigLoader",
  "docstring": "Parse YAML configuration...",
  "code": "def parse_config(...):\n    ..."
}
```

### **10.6 Query & Retrieval Strategy**

**Primary User: AI Coding Agents**

The system is designed to be invoked by AI coding agents (like Claude Code, Cursor, Aider) as a CLI tool to gain codebase insights. Human programmers interact through these agents, asking questions about code and requesting changes.

**Query Modes: Hybrid Default with Explicit Flags**

The system supports three query modes optimized for AI agent efficiency:

#### **Mode 1: Code-Only Search** (`--code`)
```bash
code-scout search "authenticate function definition" --code
code-scout search "UserController class" --code
code-scout search "where is parse_config implemented" --code
```

**When AI agents use this:**
- Searching for specific functions, classes, or implementations
- Need precise code locations for editing
- Want to avoid documentation noise in results

**Benefits:**
- ✅ Faster (1 API call instead of 2)
- ✅ More precise results (no docs chunks)
- ✅ Better token efficiency (critical for AI context windows)

#### **Mode 2: Docs-Only Search** (`--docs`)
```bash
code-scout search "authentication architecture overview" --docs
code-scout search "why was this design chosen" --docs
code-scout search "API design patterns" --docs
```

**When AI agents use this:**
- Understanding architectural decisions
- Learning design patterns and conventions
- Reading conceptual explanations

**Benefits:**
- ✅ Skip irrelevant code chunks
- ✅ Get high-level conceptual understanding
- ✅ Faster than hybrid for pure documentation queries

#### **Mode 3: Hybrid Search** (default, no flag)
```bash
code-scout search "how does authentication work"
code-scout search "error handling patterns"
code-scout search "user authentication flow"  # default behavior
```

**When AI agents use this:**
- Exploring unfamiliar parts of the codebase
- Need both implementation details and conceptual context
- Unsure whether answer is in code or docs

**Implementation:**
- Generate query embeddings with both models
- Search both vector spaces in parallel
- Merge and rank results by relevance score
- Return unified context with both code and docs chunks

**Benefits:**
- ✅ Never misses relevant context
- ✅ Optimal for exploratory queries
- ✅ Comprehensive understanding

**CLI Design for AI Agents:**

```bash
# Explicit mode selection (recommended for efficiency)
code-scout search "query" --code
code-scout search "query" --docs
code-scout search "query" --hybrid

# Shorthand aliases
code-scout search "query" -c  # code-only
code-scout search "query" -d  # docs-only

# Default behavior (no flag)
code-scout search "query"  # hybrid

# JSON output for parsing
code-scout search "query" --code --json

# Limit results for token efficiency
code-scout search "query" --limit 10

# Filter by file patterns
code-scout search "query" --code --files "src/auth/**"
```

**Response Format (JSON):**

```json
{
  "query": "authenticate function",
  "mode": "code",
  "results": [
    {
      "chunk_id": "uuid",
      "type": "function",
      "name": "authenticate",
      "file": "src/auth/handlers.py",
      "line_start": 42,
      "line_end": 67,
      "language": "python",
      "score": 0.94,
      "code": "def authenticate(username, password):\n    ...",
      "context": {
        "imports": ["hashlib", "jwt"],
        "parent_class": "AuthHandler",
        "docstring": "Authenticate user with credentials..."
      }
    }
  ],
  "total_results": 15,
  "returned": 10
}
```

**AI Agent Optimization Features:**

1. **Batch Queries** (future):
   ```bash
   code-scout search --batch queries.json
   ```

2. **Relevance Scoring**: All results include similarity scores so agents can filter low-relevance chunks

3. **Metadata-Rich Results**: File paths, line numbers, language, chunk types for precise navigation

4. **Token-Aware Limiting**: `--limit` flag to control result count and manage context window usage

5. **File Pattern Filtering**: `--files` glob patterns to narrow search scope

**Future Optimizations:**
- Cache frequently-queried embeddings (e.g., common search patterns)
- Pre-compute embeddings for common queries
- Support incremental index updates (only re-embed changed files)
- Query result caching with invalidation on file changes

---

# **11. Conclusion**

This project creates a highly efficient, per-project embedded system for giving LLM agents full, rich comprehension of a codebase. By using a dual-pass model architecture with domain-optimized embeddings and a provider-agnostic API design, it achieves **significantly higher retrieval quality** than single-model approaches while remaining fully flexible in deployment (self-hosted via Ollama or cloud-based APIs).

The Go implementation with LanceDB provides a lightweight, single-binary tool that developers can use across unlimited projects without infrastructure overhead.

This overview intentionally stays high-level to support **vertical-slice implementation** without getting bogged down in premature details.

