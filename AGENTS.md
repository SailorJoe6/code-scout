## Before You Start Developing

**IMPORTANT**: Before making any code changes, review the [Developer Guide](DEVELOPERS.md) for:
- Build requirements and setup instructions
- Required environment variables for CGO/LanceDB
- Common build issues and solutions
- Project structure overview

**Quick check**: If you need to build the project, you MUST have the LanceDB native libraries downloaded and CGO environment variables configured. See DEVELOPERS.md for details.

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

**Current Status (Slice 2):**
- ✅ Go support with semantic chunking (functions, methods, structs, interfaces)
- ⏳ Documentation indexing coming in Slice 3a

**Keep it up to date!** 
- Use `code-scout search` to understand this codebase while you work on it.
- Use `code-scout index` to update the database after every code change

## Issue Tracking with beads (bd)

**IMPORTANT**: This project uses **beads (bd)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why beads?

- Dependency-aware: Track blockers and relationships between issues
- Git-friendly: Auto-syncs to JSONL for version control
- Agent-optimized: JSON output, ready work detection, discovered-from links
- Prevents duplicate tracking systems and confusion

### Quick Start

**Check for ready work:**
```bash
bd ready --json
```

**Create new issues:**
```bash
bd create "Issue title" -t bug|feature|task -p 0-4 --json
bd create "Issue title" -p 1 --deps discovered-from:bd-123 --json
```

**Claim and update:**
```bash
bd update bd-42 --status in_progress --json
bd update bd-42 --priority 1 --json
```

**Complete work:**
```bash
bd close bd-42 --reason "Completed" --json
```

### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

## Elephant Carpaccio Development

**IMPORTANT**: This project follows the **Elephant Carpaccio** approach to development. Always deliver value in thin, vertical slices.

### Planning Principle: Vertical Slices

When planning work, **always break down features into small vertical slices** that deliver end-to-end value:

✅ **Good (Vertical Slices):**
- Slice 1: Index one Python file and search it (complete flow, immediate value)
- Slice 2: Add semantic chunking (better results, still works end-to-end)
- Slice 3: Add documentation support (more features, complete system)

❌ **Bad (Horizontal Layers):**
- Task 1: Build file scanner (no value until everything else is done)
- Task 2: Build chunker (still no value)
- Task 3: Build embeddings client (still no value)
- Task 4: Build storage (finally works, but took 4 tasks)

**Each slice must:**
- Work end-to-end (user can actually use it)
- Deliver incremental value (better than the previous slice)
- Be independently testable
- Take hours or days, not weeks

**Think:** "What's the smallest thing I can build that actually works?"

### Implementation Principle: Commit Per Slice

**Always commit after completing each vertical slice:**

```bash
# After Slice 1 is working end-to-end
git add .
git commit -m "Implement Slice 1: Basic indexing works"

# After Slice 2 is working end-to-end
git add .
git commit -m "Implement Slice 2: Add semantic chunking"

# And so on...
```

**Benefits:**
- ✅ Clean git history showing incremental progress
- ✅ Easy to revert to last working state
- ✅ Each commit is a functioning system
- ✅ Beads issues stay in sync with code (commit `.beads/issues.jsonl` together)
- ✅ Clear milestones for reviewing progress

**Never commit:**
- ❌ Half-finished features that don't work
- ❌ Multiple slices bundled together
- ❌ Broken/untested code

### Workflow for AI Agents

1. **Check ready work**: `bd ready` shows unblocked issues
2. **Claim your task**: `bd update <id> --status in_progress`
3. **Work on it**: Implement, test, document
4. **Discover new work?** Create linked issue:
   - `bd create "Found bug" -p 1 --deps discovered-from:<parent-id>`
5. **Complete**: `bd close <id> --reason "Done"`
6. **Commit together**: Always commit the `.beads/issues.jsonl` file together with the code changes so issue state stays in sync with code state

### Auto-Sync

bd automatically syncs with git:
- Exports to `.beads/issues.jsonl` after changes (5s debounce)
- Imports from JSONL when newer (e.g., after `git pull`)
- No manual export/import needed!

### ⚠️ CRITICAL: NEVER Edit JSONL Files Directly Unless to resolve a conflict. 

**NEVER, EVER read, edit, or update the `.beads/issues.jsonl` file directly to manage beads issues**

- **The database is the source of truth** - The JSONL file is a git-friendly export, not the primary storage
- **Direct edits will be overwritten** - The beads database will overwrite your changes
- **Use the beads MCP server** - Prefer `mcp__beads__*` functions if available
- **Use the CLI** - Use `bd` commands as a fallback
- **The JSONL file is read-only from your perspective** - Only beads itself should modify it

Any attempt to directly edit `.beads/issues.jsonl` will result in data loss and corruption.  The only time this rule can be relaxed is in the case of a merge conflict, in which case you are allowed to take the best course of action to resolve the conflict.  

### MCP Server (Recommended)

If using Claude or MCP-compatible clients, install the beads MCP server:

```bash
pip install beads-mcp
```

Add to MCP config (e.g., `~/.config/claude/config.json`):
```json
{
  "beads": {
    "command": "beads-mcp",
    "args": []
  }
}
```

Then use `mcp__beads__*` functions instead of CLI commands.

### Managing AI-Generated Planning Documents

AI assistants often create planning and design documents during development:
- PLAN.md, IMPLEMENTATION.md, ARCHITECTURE.md
- DESIGN.md, CODEBASE_SUMMARY.md, INTEGRATION_PLAN.md
- TESTING_GUIDE.md, TECHNICAL_DESIGN.md, and similar files

**Best Practice: Use a dedicated directory for these ephemeral files**

**Recommended approach:**
- Create a `docs/planning/` directory in the project root
- Store ALL AI-generated planning/design docs in `docs/planning/`
- Keep the repository root clean and focused on permanent project files
- Only access `docs/planning/` when explicitly asked to review past planning

**Example .gitignore entry (optional):**
```
# AI planning documents (ephemeral)
docs/planning/
```

**Benefits:**
- ✅ Clean repository root
- ✅ Clear separation between ephemeral and permanent documentation
- ✅ Easy to exclude from version control if desired
- ✅ Preserves planning history for archeological research
- ✅ Reduces noise when browsing the project

### Important Rules

- ✅ **Review DEVELOPERS.md** before building or modifying code
- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ✅ Store AI planning docs in `docs/planning/` directory
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems
- ❌ Do NOT clutter repo root with planning documents

For more details, see README.md, DEVELOPERS.md, and QUICKSTART.md.