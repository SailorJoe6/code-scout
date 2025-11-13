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

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ✅ Store AI planning docs in `docs/planning/` directory
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems
- ❌ Do NOT clutter repo root with planning documents

For more details, see README.md and QUICKSTART.md.