## Issue Tracking

This project uses **bd (beads)** for issue tracking.
Run `bd prime` for workflow context, or install hooks (`bd hooks install`) for auto-injection.

**Quick reference:**
- `bd ready` - Find unblocked work
- `bd create "Title" --type task --priority 2` - Create issue
- `bd close <id>` - Complete work
- `bd sync` - Sync with git (run at session end)

For full workflow details: `bd prime`

## Before You Start Developing

**Quick check**: If you need to build the project, you MUST have the LanceDB native libraries downloaded and use our Make files for testing.

## Dogfooding: Use Code Scout CLI

**IMPORTANT**: When working in this repo, **dogfood the code-scout CLI** to understand the codebase. This is what we're building!

### What is Code Scout?

A semantic code search tool that indexes codebases using embeddings. It understands code structure (functions, methods, types) and finds relevant code based on semantic similarity, not just text matching.

### When to Use It

**Use code-scout when:**
- ✅ Finding where functionality is implemented ("where is authentication handled?")
- ✅ Understanding code structure ("what functions deal with parsing?")
- ✅ Exploring unfamiliar parts of the codebase
- ✅ Finding related code across multiple files

**Use grep/other tools when:**
- ❌ Finding exact text matches (variable names, strings)
- ❌ You already know exactly what file/function you need

### Basic Usage

```bash
# Index the repo (run from repo root)
./dist/code-scout-darwin_arm64/code-scout index

# Search for code semantically
./dist/code-scout-darwin_arm64/code-scout search "tree-sitter parsing" --json

# Search returns:
# - File paths
# - Line numbers
# - Matching code chunks (functions, methods, types)
# - Relevance scores
```

**Keep it up to date!** 
- Use `code-scout search` to understand this codebase while you work on it.
- Use `code-scout index` to update the database after every code change

