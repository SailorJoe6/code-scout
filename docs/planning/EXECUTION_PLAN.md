# Execution Plan: Ralph Standalone Repository Refactoring

## Overview

Transform the `ralph` workflow tool from a code_scout-specific script into a standalone, reusable git repository. This enables any project to use ralph's design → plan → execute workflow while maintaining backward compatibility with code_scout.

**Integration Approach:** Ralph is cloned as a regular git repository into the project directory and added to `.gitignore`. This keeps ralph's git history separate from the host project while making it easy to update ralph independently.

**Status:** In progress
**Estimated Complexity:** Medium - Primarily bash refactoring, git operations, and documentation

**CRITICAL IMPLEMENTATION NOTE:**

The original `ralph` script has been renamed to `ralph-script` to avoid naming conflict with the cloned repository. We must maintain the existing `ralph-script` and prompts in `.claude/commands/` functional throughout implementation. Files should be **COPIED** (not moved) when creating the new ralph repository. Only in the final phase should we delete `ralph-script` and replace the prompts with symlinks.

---

## Prerequisites

- GitHub account with access to create new repository (or equivalent git hosting)
- Existing code_scout repository cloned locally
- Basic bash scripting knowledge
- Ralph repository will be cloned (not added as submodule)

---

## Phase 1: Create Ralph Repository Structure

**Location:** Work in a separate directory, NOT inside code_scout

### Tasks

1. **Create new GitHub repository** ✅ COMPLETED
   - Repository name: `ralph`
   - Description: "Reusable AI-assisted development workflow tool (design → plan → execute)"
   - Visibility: Public
   - License: None (public domain)
   - Initialize with main branch
   - Clone to local directory: `git clone https://github.com/<username>/ralph.git`

   **Status:** Repository created, cloned locally, and added to code_scout's .gitignore temporarily

2. **Create directory structure**
   ```bash
   cd ralph/
   mkdir prompts
   # plans/ directory will be created on-demand by workflow (gitignored)
   ```

3. **Copy prompt files from code_scout as templates**
   ```bash
   cp /Users/jlanders/gitlab_local/code_scout/.claude/commands/design.md prompts/design.example.md
   cp /Users/jlanders/gitlab_local/code_scout/.claude/commands/plan.md prompts/plan.example.md
   cp /Users/jlanders/gitlab_local/code_scout/.claude/commands/execute.md prompts/execute.example.md
   ```

   **CRITICAL:** Prompts are project-specific (they reference DEVELOPERS.md, docs/README.md, etc.). The `.example.md` files serve as templates. Users must copy and customize them for their project. The actual `.md` files will be gitignored.

   **IMPORTANT:** This is a COPY operation. Do NOT delete or move the original files in code_scout. The existing ralph-script needs access to these prompts throughout the implementation process.

4. **Create `.gitignore`**
   ```gitignore
   # Planning documents (ephemeral, per-project)
   plans/

   # Configuration (project-specific)
   .env

   # Prompts (project-specific, customized from .example.md templates)
   prompts/*.md

   # Log files (created in project root, not ralph directory)
   ralph-error.md
   ralph-output.md
   ```

5. **Verify file structure**
   ```bash
   ls -la
   # Expected: .git/, prompts/, .gitignore
   ls prompts/
   # Expected: design.example.md, plan.example.md, execute.example.md
   ```

**Critical Files:**
- `ralph/.gitignore` (created)
- `ralph/prompts/design.example.md` (template)
- `ralph/prompts/plan.example.md` (template)
- `ralph/prompts/execute.example.md` (template)

---

## Phase 2: Create Configuration Template

### Tasks

1. **Create `.env.example`** with complete documentation:

```bash
# Ralph Configuration
# Copy this file to .env and customize for your project

# === Prompt File Paths ===
# Default: ralph/prompts/design.md
#DESIGN_PROMPT=ralph/prompts/design.md

# Default: ralph/prompts/plan.md
#PLAN_PROMPT=ralph/prompts/plan.md

# Default: ralph/prompts/execute.md
#EXECUTE_PROMPT=ralph/prompts/execute.md

# === Planning Document Paths ===
# Default: ralph/plans/SPECIFICATION.md
#SPECIFICATION=ralph/plans/SPECIFICATION.md

# Default: ralph/plans/EXECUTION_PLAN.md
#EXECUTION_PLAN=ralph/plans/EXECUTION_PLAN.md

# === Container Configuration ===
# Container name (leave empty if not using containers)
# Default: empty
#CONTAINER_NAME=

# Container working directory (defaults to /<basename> where basename is current directory name)
# Default: /<basename>
#CONTAINER_WORKDIR=

# Container runtime (docker or podman)
# Default: docker
#CONTAINER_RUNTIME=docker

# === Behavior Flags ===
# Uncomment to enable unattended mode (only applies during execute phase)
#UNATTENDED=1

# Uncomment to use Codex instead of Claude
#USE_CODEX=1

# Uncomment to run a callback script after each pass
#CALLBACK=./validate.sh
```

**Critical Files:**
- `ralph/.env.example` (created)

---

## Phase 3: Refactor Start Script

### Tasks

1. **Copy base script**
   ```bash
   cp /Users/jlanders/gitlab_local/code_scout/ralph start
   chmod +x start
   ```

   **IMPORTANT:** This is a COPY operation. Do NOT delete the original ralph script at the code_scout project root. We're using that script to implement this plan, so it must remain functional.

2. **Add .env file loading** (insert after line 7, before UNATTENDED=0):
   ```bash
   # Load .env file if it exists (lowest precedence)
   RALPH_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
   if [[ -f "$RALPH_DIR/.env" ]]; then
     set -a
     source "$RALPH_DIR/.env"
     set +a
   fi
   ```

3. **Update default variables** (replace lines 3-14):
   ```bash
   # Log files (always in project root)
   ERROR_LOG="ralph-error.md"
   OUT_LOG="ralph-output.md"

   # Default paths (can be overridden by ENV or .env)
   : "${DESIGN_PROMPT:=$RALPH_DIR/prompts/design.md}"
   : "${PLAN_PROMPT:=$RALPH_DIR/prompts/plan.md}"
   : "${EXECUTE_PROMPT:=$RALPH_DIR/prompts/execute.md}"
   : "${SPECIFICATION:=$RALPH_DIR/plans/SPECIFICATION.md}"
   : "${EXECUTION_PLAN:=$RALPH_DIR/plans/EXECUTION_PLAN.md}"

   # Behavior flags (can be overridden by ENV or .env)
   : "${UNATTENDED:=0}"
   : "${USE_CODEX:=0}"
   PASS=0
   : "${CALLBACK:=}"
   : "${CONTAINER_NAME:=}"
   # CONTAINER_WORKDIR will be set dynamically if not provided
   : "${CONTAINER_RUNTIME:=docker}"
   ```

4. **Add --workdir flag** (update argument parsing section around line 19-50):
   ```bash
   CONTAINER_WORKDIR_SET=0  # Track if explicitly set via CLI

   while [[ $# -gt 0 ]]; do
     case "$1" in
       -u|--unattended)
         UNATTENDED=1
         shift
         ;;
       --codex)
         USE_CODEX=1
         shift
         ;;
       --container)
         if [[ $# -lt 2 ]]; then
           echo "Error: --container requires a container name"
           exit 2
         fi
         CONTAINER_NAME="$2"
         shift 2
         ;;
       --workdir)
         if [[ $# -lt 2 ]]; then
           echo "Error: --workdir requires a path"
           exit 2
         fi
         CONTAINER_WORKDIR="$2"
         CONTAINER_WORKDIR_SET=1
         shift 2
         ;;
       --callback)
         if [[ $# -lt 2 ]]; then
           echo "Error: --callback requires a script path"
           exit 2
         fi
         CALLBACK="$2"
         shift 2
         ;;
       *)
         echo "Usage: $0 [-u|--unattended] [--codex] [--container <name>] [--workdir <path>] [--callback <script>]"
         exit 2
         ;;
     esac
   done
   ```

5. **Add workdir default calculation** (insert after argument parsing, before callback validation):
   ```bash
   # Calculate default workdir if using container and not explicitly set
   if [[ -n "$CONTAINER_NAME" && "$CONTAINER_WORKDIR_SET" -eq 0 && -z "$CONTAINER_WORKDIR" ]]; then
     BASENAME="$(basename "$(pwd)")"
     CONTAINER_WORKDIR="/$BASENAME"
   fi
   ```

6. **Add prompt validation with helpful error messages** (add after main loop starts):
   ```bash
   # Check if prompt file exists and provide helpful error
   if [[ ! -f "$PROMPT" ]]; then
     echo "Error: Prompt file not found: $PROMPT"
     echo ""
     echo "Prompts are project-specific and must be customized for your project."
     echo "Copy from the example template:"
     echo ""
     if [[ "$PROMPT" == *"/design.md" ]]; then
       echo "  cp ralph/prompts/design.example.md ralph/prompts/design.md"
     elif [[ "$PROMPT" == *"/plan.md" ]]; then
       echo "  cp ralph/prompts/plan.example.md ralph/prompts/plan.md"
     elif [[ "$PROMPT" == *"/execute.md" ]]; then
       echo "  cp ralph/prompts/execute.example.md ralph/prompts/execute.md"
     fi
     echo ""
     echo "Then edit the prompt to reference your project's documentation."
     exit 1
   fi
   ```

7. **Update planning document checks** (replace hardcoded paths at lines 103-110):
   ```bash
   if [[ -f "$SPECIFICATION" && -f "$EXECUTION_PLAN" ]]; then
     PROMPT="$EXECUTE_PROMPT"
   elif [[ -f "$SPECIFICATION" ]]; then
     PROMPT="$PLAN_PROMPT"
   elif [[ -f "$EXECUTION_PLAN" ]]; then
     echo "Error: $EXECUTION_PLAN exists but $SPECIFICATION is missing."
     exit 1
   fi
   ```

**Critical Changes:**
- Line 8+ (after line 7): Add .env loading
- Lines 3-14: Replace with configurable defaults
- Lines 19-50: Add --workdir flag to argument parsing
- After line 50: Add workdir default calculation
- Lines 103-110: Replace hardcoded paths with variables

**Verification:**
```bash
./start --help
# Should show updated usage with --workdir

# Test .env loading (create test .env)
echo 'CONTAINER_RUNTIME=podman' > .env
grep -A10 'Load .env' start
# Verify .env loading code exists
```

**Critical Files:**
- `ralph/start` (refactored from ralph script)

---

## Phase 4: Create Documentation

### Tasks

1. **Create comprehensive README.md** with sections:

```markdown
# Ralph - Reusable AI-Assisted Development Workflow Tool

Ralph implements a design → plan → execute workflow for AI-assisted development with support for Claude and Codex CLI tools.

## What is Ralph?

Ralph orchestrates a structured workflow for AI-assisted development:

1. **Design Phase** - Discuss requirements with AI, create specification
2. **Plan Phase** - AI creates detailed execution plan based on specification
3. **Execute Phase** - AI implements the plan with optional unattended mode

Ralph automatically progresses through phases based on which planning documents exist:
- No docs → runs design phase
- SPECIFICATION.md exists → runs plan phase
- Both docs exist → runs execute phase

## Installation

Clone ralph into your project and add it to `.gitignore`:

```bash
# Clone ralph into your project
git clone https://github.com/<username>/ralph.git ralph

# Add to .gitignore to keep ralph separate from your project
echo "ralph/" >> .gitignore

# (Optional) Create .env configuration
cp ralph/.env.example ralph/.env
# Edit ralph/.env with project-specific settings
```

**Why this approach?** Cloning ralph and adding it to `.gitignore` keeps ralph's git history separate from your project while making it easy to update ralph independently with `git pull` from within the ralph directory.

## Quick Start

```bash
# Basic usage (interactive)
ralph/start

# Unattended execution (execute phase only)
ralph/start --unattended

# Use Codex instead of Claude
ralph/start --codex
```

## Configuration

Ralph can be configured via:
1. Command-line arguments (highest precedence)
2. Environment variables
3. `.env` file (copy from `.env.example`)
4. Script defaults (lowest precedence)

See `.env.example` for all configurable options.

## Command-Line Options

- `-u, --unattended` - Run execute phase in unattended mode (auto-approve)
- `--codex` - Use Codex instead of Claude
- `--container <name>` - Execute commands inside specified container
- `--workdir <path>` - Container working directory (defaults to `/<basename>`)
- `--callback <script>` - Run script after each pass

## Container Support

Ralph can execute AI commands inside a running dev container:

```bash
# Using default workdir (/<basename>)
ralph/start --container my-dev-container

# Custom workdir
ralph/start --container my-dev-container --workdir /workspace/myproject

# With Codex
ralph/start --container my-dev-container --codex
```

The default workdir is `/<basename>` where basename is your current directory name.
Example: Running from `/Users/name/code_scout` → defaults to `/code_scout`

## Integration with AI Assistants (Optional)

For slash command support in Claude/Codex, create symlinks:

```bash
mkdir -p .claude/commands
ln -s ../../ralph/prompts/design.md .claude/commands/design.md
ln -s ../../ralph/prompts/plan.md .claude/commands/plan.md
ln -s ../../ralph/prompts/execute.md .claude/commands/execute.md
```

Then you can run `/design`, `/plan`, or `/execute` directly in your AI assistant.

## Workflow Phases

### Design Phase
- Interactive conversation with AI about requirements
- Creates `ralph/plans/SPECIFICATION.md` with detailed specification
- Next run enters plan phase

### Plan Phase
- AI reads specification and codebase
- Creates `ralph/plans/EXECUTION_PLAN.md` with implementation plan
- Next run enters execute phase

### Execute Phase
- AI implements the plan
- Supports `--unattended` mode for automation
- Loops until work is complete

## Files Created by Ralph

- `ralph/plans/SPECIFICATION.md` - Created during design phase
- `ralph/plans/EXECUTION_PLAN.md` - Created during plan phase
- `ralph-error.md` - Error output (project root)
- `ralph-output.md` - Stdout in unattended mode (project root)

All planning documents are gitignored (ephemeral, per-project).

## Troubleshooting

**Symlinks not working:**
- Ensure relative paths are correct: `../../ralph/prompts/design.md`
- Use `ls -la .claude/commands/` to verify symlink targets

**Container exec fails:**
- Verify container is running: `docker ps | grep <container-name>`
- Check workdir exists in container: `docker exec <name> ls /<workdir>`
- Override with `--workdir` if default is incorrect

**Planning docs not found:**
- Default location is `ralph/plans/` (inside submodule)
- Override via `.env` if using different location
- Ensure paths are absolute or relative to where you run ralph/start

## License

This project is released into the public domain. No license necessary - use it however you want.
```

**Critical Files:**
- `ralph/README.md` (created)

---

## Phase 5: Commit Ralph Repository

### Tasks

1. **Review all files**
   ```bash
   git status
   ls -la
   # Expected: start, .gitignore, .env.example, README.md, prompts/
   ```

2. **Commit and push**
   ```bash
   git add .
   git commit -m "Initial commit: Ralph standalone workflow tool

- Refactored start script with .env support and configurable paths
- Added comprehensive README with installation instructions
- Created .env.example with all configuration options
- Organized prompts into prompts/ directory
- Added .gitignore for plans/, logs, and .env

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"

   git push origin main
   ```

3. **Verify remote**
   ```bash
   git log --oneline
   git remote -v
   # Ensure pushed successfully
   ```

**Verification:**
- Repository pushed to remote
- All files committed
- Ready to add as submodule

---

## Phase 6: Integrate into Code Scout

**Location:** Work in code_scout repository

### Tasks

1. **Verify ralph is cloned and in .gitignore**
   ```bash
   cd /Users/jlanders/gitlab_local/code_scout

   # Check ralph directory exists
   ls -la ralph/
   # Should show: .git/, start, prompts/, .gitignore, README.md, .env.example

   # Verify ralph/ is in .gitignore
   grep "ralph/" .gitignore
   # Should show: ralph/
   ```

   **Status:** Ralph repository already cloned and added to .gitignore.

2. **Create symlinks for slash commands**

   **DEFERRED TO PHASE 8:** Do NOT create symlinks yet. The existing `ralph-script` needs the prompt files at `.claude/commands/` to remain in place. Symlink creation will happen in Phase 8 as the final step.

3. **Migrate active planning documents**
   ```bash
   # Create plans directory in ralph clone
   mkdir -p ralph/plans

   # Copy current specification (this is the ralph refactoring spec)
   cp docs/planning/SPECIFICATION.md ralph/plans/SPECIFICATION.md
   cp docs/planning/EXECUTION_PLAN.md ralph/plans/EXECUTION_PLAN.md
   ```

4. **Delete old ralph-script**

   **DEFERRED TO PHASE 8:** Do NOT delete `ralph-script` yet. We may still need it during implementation. Deletion will happen in Phase 8 as the final step.

5. **Verify integration**
   ```bash
   # Verify cloned ralph directory
   ls -la ralph/
   # Should show: start, prompts/, .gitignore, README.md, .env.example, plans/

   # Test new ralph/start script
   ./ralph/start --help
   # Should show usage with --workdir option
   ```

**Critical Files:**
- `ralph/` (cloned repository, in .gitignore)
- `ralph/plans/SPECIFICATION.md` (migrated)
- `ralph/plans/EXECUTION_PLAN.md` (migrated)

**Note:** At this point, both `ralph-script` and `ralph/start` exist. The old script at project root and prompts in `.claude/commands/` remain unchanged and functional.

---

## Phase 7: Update Code Scout Documentation

### Tasks

1. **Update DEV_CONTAINER.md**

   Replace all `./ralph` references with `./ralph/start`:
   - Line 99: `./ralph --codex` → `./ralph/start --codex`
   - Line 129: `./ralph --container code-scout-dev --codex` → `./ralph/start --container code-scout-dev --codex`
   - Line 131: `./ralph --container code-scout-dev` → `./ralph/start --container code-scout-dev`

2. **Update CLAUDE.md if needed**

   Search for ralph references:
   ```bash
   grep -n ralph CLAUDE.md
   ```

   Update any references to reflect new structure (if found).

3. **Verify no other documentation needs updates**
   ```bash
   grep -r "\\./ralph" docs/
   # Check results and update as needed
   ```

**Critical Files:**
- `docs/guides/DEV_CONTAINER.md` (updated)
- `CLAUDE.md` (check and update if needed)

---

## Phase 8: Final Testing and Cutover

**CRITICAL:** This phase replaces the old ralph script with the new one and creates symlinks.

### Tasks

1. **Test new ralph/start script**
   ```bash
   # Test basic invocation
   ./ralph/start --help

   # Test with specification (should enter plan phase)
   ./ralph/start
   # Should detect ralph/plans/SPECIFICATION.md and run plan prompt
   ```

2. **Test .env loading (optional)**
   ```bash
   # Create test .env in ralph/
   echo 'CONTAINER_RUNTIME=podman' > ralph/.env

   # Verify it loads (check script has loading code)
   grep -A5 'Load .env' ralph/start

   # Clean up test
   rm ralph/.env
   ```

3. **Replace prompt files with symlinks**
   ```bash
   # Remove existing prompt files
   rm .claude/commands/design.md
   rm .claude/commands/plan.md
   rm .claude/commands/execute.md

   # Create symlinks (relative paths from .claude/commands/)
   ln -s ../../ralph/prompts/design.md .claude/commands/design.md
   ln -s ../../ralph/prompts/plan.md .claude/commands/plan.md
   ln -s ../../ralph/prompts/execute.md .claude/commands/execute.md
   ```

4. **Delete old ralph-script**
   ```bash
   rm ralph-script  # The old script at project root (backed up in git history)
   ```

5. **Verify symlinks work**
   ```bash
   # Verify symlink targets
   file .claude/commands/design.md
   # Should show: symbolic link to ../../ralph/prompts/design.md

   # Verify content accessible
   head -5 .claude/commands/design.md
   ```

6. **Create comprehensive commit**
   ```bash
   git add .claude/commands/
   git add docs/
   git add .gitignore  # If modified

   git commit -m "Refactor ralph workflow: use cloned repository with symlinked prompts

- Create symlinks from .claude/commands/ to ralph/prompts/
- Update documentation to reference ralph/start
- Remove original ralph-script from project root
- Planning docs migrated to ralph/plans/ (gitignored)

Ralph is now a standalone repository cloned into projects.
See ralph/README.md for installation and usage instructions.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
   ```

7. **Push to remote**
   ```bash
   git pull --rebase
   git push
   git status
   # Verify: "Your branch is up to date with 'origin/main'"
   ```

**Verification:**
- All changes committed
- Symlinks functional
- Ralph workflow operational
- Pushed to remote successfully
- ralph/ directory remains in .gitignore

---

## Phase 9: Archive Planning Documents

### Tasks

1. **Archive this specification**
   ```bash
   # Create archive directory if needed
   mkdir -p docs/planning/archive

   # Move this specification to archive
   mv docs/planning/SPECIFICATION.md docs/planning/archive/ralph-refactoring-2026-01-22.md

   # Commit archival
   git add docs/planning/
   git commit -m "Archive ralph refactoring specification"
   git push
   ```

2. **Clean up ralph/plans if desired**
   ```bash
   # The ralph/plans/ directory is gitignored, so local cleanup only
   # Consider removing ralph/plans/SPECIFICATION.md if no longer needed
   # This is optional - gitignored files don't affect repository
   ```

**Verification:**
- Planning docs archived
- docs/planning/ is clean and ready for next feature

---

## Critical Implementation Notes

### Git Operation Order

**MUST be sequential:**
1. Create and push ralph repository to remote (Phase 1-5)
2. Verify ralph is cloned and in .gitignore (Phase 6, task 1)
3. Keep existing ralph-script and prompts functional until Phase 8
4. Only in Phase 8: Delete old ralph-script and replace prompts with symlinks

**CRITICAL:** Because we're using the existing ralph-script to implement this plan, the original ralph-script at the project root and the prompts in `.claude/commands/` must remain untouched until Phase 8. All earlier phases only COPY files, they do not move or delete the originals.

**IMPORTANT:** The ralph/ directory is in code_scout's .gitignore and should remain there. Ralph is kept as a separate cloned repository, not tracked in code_scout's git history.

### Configuration Precedence

The start script respects this precedence (highest to lowest):
1. Command-line arguments (`--workdir`, etc.)
2. Environment variables (`CONTAINER_WORKDIR`, etc.)
3. `.env` file values
4. Script defaults

### Symlink Paths

From `.claude/commands/` to `ralph/prompts/`:
```
.claude/commands/design.md → ../../ralph/prompts/design.md
```

Path calculation:
- From: `/Users/.../code_scout/.claude/commands/design.md`
- To: `/Users/.../code_scout/ralph/prompts/design.md`
- Relative: up 2 levels (../../), then ralph/prompts/design.md

### Container Workdir Default

Default calculation when using `--container` without `--workdir`:
```bash
BASENAME="$(basename "$(pwd)")"
CONTAINER_WORKDIR="/$BASENAME"
```

Example: Running from `/Users/jlanders/gitlab_local/code_scout` → `CONTAINER_WORKDIR=/code_scout`

### .env File Location

**Important:** The `.env` file lives at `ralph/.env` (inside the cloned ralph directory), NOT at the project root. Each project that clones ralph can have its own `ralph/.env` for project-specific configuration. This file is gitignored by ralph, so it stays local to each project.

For code_scout specifically:
- Default behavior: Running from `code_scout/` → `CONTAINER_WORKDIR=/code_scout`
- If code_scout needs `/workspaces/code_scout`, create `ralph/.env` with:
  ```bash
  CONTAINER_WORKDIR=/workspaces/code_scout
  ```

---

## Rollback Strategy

If issues arise:

**Before Phase 8 commit:**
```bash
# Restore original files
git checkout HEAD -- ralph-script  # Restore old script (if deleted)
git checkout HEAD -- .claude/commands/  # Restore prompts (if symlinked)

# Clean working directory
git reset --hard HEAD
```

**After Phase 8 commit but before push:**
```bash
git reset --soft HEAD~1
# Then follow rollback steps above
```

**After push:**
- Create revert commit
- Manually restore original ralph-script and prompt files from git history
- Remove symlinks and restore original prompt files

**Note:** Ralph clone in ralph/ directory is unaffected by rollback since it's in .gitignore

---

## Success Criteria

Implementation is complete when:

1. ✅ Ralph repository exists at GitHub with correct structure
2. ✅ `ralph/start` script supports .env configuration
3. ✅ `ralph/start` supports --workdir flag with dynamic default
4. ✅ `.env.example` documents all configuration options
5. ✅ `.gitignore` excludes plans/, .env, logs
6. ✅ README.md provides complete documentation
7. ✅ Code scout has ralph cloned and in .gitignore
8. ✅ Symlinks from .claude/commands/ work correctly
9. ✅ Ralph workflow functions in code_scout
10. ✅ All changes committed and pushed to remote

---

## End-to-End Verification

After completing all phases:

```bash
# Fresh clone test (if possible)
cd /tmp
git clone https://github.com/<username>/code_scout.git test-code-scout
cd test-code-scout

# Clone ralph
git clone https://github.com/<username>/ralph.git ralph

# Verify symlinks
ls -la .claude/commands/
# Should show symlinks to ../../ralph/prompts/*.md

# Test ralph
./ralph/start --help
# Should show usage with --workdir

# Clean up
cd ..
rm -rf test-code-scout
```

---

## Configuration Details

**GitHub Repository:**
- Create at personal GitHub account: `github.com/<username>/ralph`

**Container Workdir:**
- Uses dynamic `/<basename>` default as specified
- Projects can override via `ralph/.env` (inside cloned ralph directory, project-specific, gitignored)

---

## End of Execution Plan
