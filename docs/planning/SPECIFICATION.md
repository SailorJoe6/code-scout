# Specification: Ralph - Reusable AI-Assisted Development Workflow Tool

## Overview

Refactor the `ralph` bash script and its dependencies into a standalone git repository that can be cloned into any project. Ralph implements a design → plan → execute workflow for AI-assisted development with support for Claude and Codex CLI tools.

**Integration Approach:** Ralph is cloned as a regular git repository into the project directory and added to `.gitignore`. This keeps ralph's git history separate from the host project while making it easy to update ralph independently.

## Current State

The ralph workflow currently exists in the code_scout project with these components:

**Files:**
- `ralph-script` - Main bash script at project root (renamed from `ralph` to avoid naming conflict with cloned repository)
- `.claude/commands/design.md` - Design phase prompt
- `.claude/commands/plan.md` - Planning phase prompt
- `.claude/commands/execute.md` - Execution phase prompt
- `docs/planning/SPECIFICATION.md` - Specification document created in design phase
- `docs/planning/EXECUTION_PLAN.md` - Execution plan created in planning phase

**Current ralph script hardcoded values:**
- `ERROR_LOG="ralph-error.md"` - Error output file
- `OUT_LOG="ralph-output.md"` - Standard output file (unattended mode)
- `DESIGN_PROMPT=".claude/commands/design.md"` - Design phase prompt path
- `PLAN_PROMPT=".claude/commands/plan.md"` - Planning phase prompt path
- `EXECUTE_PROMPT=".claude/commands/execute.md"` - Execution phase prompt path
- `CONTAINER_WORKDIR="/workspaces/code_scout"` - Hardcoded container working directory

**Current ralph script behavior:**
- Loops through design → plan → execute phases based on which planning documents exist
- Supports `--unattended` mode for automated execution
- Supports `--codex` flag to use Codex instead of Claude
- Supports `--container <name>` to exec into a running dev container
- Supports `--callback <script>` to run a script after each pass
- Uses `CONTAINER_RUNTIME` (defaults to docker) for container operations
- No handoff context between passes

## Problems with Current Implementation

1. **Not reusable** - Tightly coupled to code_scout project structure
2. **Hardcoded paths** - Prompt paths, planning doc paths, and container workdir are project-specific
3. **No configuration** - No way to customize paths or behavior per-project without editing the script
4. **Scattered files** - Prompt files in `.claude/commands/`, planning docs in `docs/planning/`
5. **Container assumptions** - Assumes VS Code dev container path structure (`/workspaces/code_scout`)
6. **No handoff between passes** - No mechanism to reconnect to the AI session and provide handoff context after each execution pass

## Goals

Transform ralph into a standalone, reusable tool that:

1. Lives in its own git repository and can be cloned into any project (added to host project's `.gitignore`)
2. Provides sensible defaults that work out-of-the-box
3. Allows per-project configuration via `.env` file and environment variables
4. Supports flexible container working directory configuration
5. Keeps all ralph-specific files contained within the cloned directory
6. Maintains backward compatibility with existing code_scout workflow

## Target State

### Repository Structure

The new `ralph` repository will have this structure:

```
ralph/
├── start                        # Main script (renamed from ralph)
├── .gitignore                   # Ignores plans/, .env, log files, and prompts/*.md
├── .env.example                 # Example configuration with defaults
├── README.md                    # Setup and usage documentation
├── prompts/
│   ├── design.example.md       # Example design phase prompt (template)
│   ├── plan.example.md         # Example planning phase prompt (template)
│   ├── execute.example.md      # Example execution phase prompt (template)
│   ├── design.md               # Actual design prompt (gitignored, user customizes)
│   ├── plan.md                 # Actual plan prompt (gitignored, user customizes)
│   └── execute.md              # Actual execute prompt (gitignored, user customizes)
└── plans/                       # Planning documents (gitignored)
    ├── SPECIFICATION.md        # Created during design phase
    └── EXECUTION_PLAN.md       # Created during planning phase
```

**Important:** Prompts are project-specific because they reference project-specific documentation (like DEVELOPERS.md, docs/README.md, etc.). The `.example.md` files provide templates that users must copy and customize for their project. The actual `.md` files are gitignored.

### Configuration System

**Configuration precedence** (highest to lowest):
1. Command-line arguments
2. Environment variables
3. `.env` file (if exists)
4. Script defaults

**Configurable via .env and ENV vars:**

Planning document paths:
- `SPECIFICATION` - Default: `ralph/plans/SPECIFICATION.md`
- `EXECUTION_PLAN` - Default: `ralph/plans/EXECUTION_PLAN.md`

Container variables:
- `CONTAINER_NAME` - Default: empty (no container)
- `CONTAINER_WORKDIR` - Default: `/<basename>` where basename is the current directory name
- `CONTAINER_RUNTIME` - Default: `docker` (can be set to `podman`)

Behavior flags (commented out in `.env.example`):
- `UNATTENDED` - Default: `0` (set to `1` to enable)
- `USE_CODEX` - Default: `0` (set to `1` to use Codex instead of Claude)
- `CALLBACK` - Default: empty (path to callback script)

**Hardcoded (not configurable):**
- Prompt paths (must match .gitignore pattern):
  - `ralph/prompts/design.md`
  - `ralph/prompts/plan.md`
  - `ralph/prompts/execute.md`
  - `ralph/prompts/handoff.md` - Called at end of each pass to reconnect to session and provide handoff context
- Log files (always in project root):
  - `ERROR_LOG="ralph-error.md"`
  - `OUT_LOG="ralph-output.md"`

### Command-Line Interface

**Invocation:**
```bash
ralph/start [OPTIONS]
```

**Options:**
- `-u, --unattended` - Run in unattended mode (only during execute phase)
- `--codex` - Use Codex instead of Claude
- `--container <name>` - Execute commands inside the specified container
- `--workdir <path>` - Container working directory (defaults to `/<basename>`)
- `--callback <script>` - Run script after each pass

**Examples:**
```bash
# Basic usage
ralph/start

# Unattended execution
ralph/start --unattended

# Execute in dev container with custom workdir
ralph/start --container my-dev-container --workdir /workspace/myproject

# Execute in dev container with default workdir (/<basename>)
ralph/start --container my-dev-container

# Use Codex instead of Claude
ralph/start --codex

# Run callback after each pass
ralph/start --callback ./validate.sh
```

### .env.example Content

```bash
# Ralph Configuration
# Copy this file to .env and customize for your project

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

**Note:** Prompt paths (`ralph/prompts/*.md`) are hardcoded and not configurable. This ensures they match the `.gitignore` pattern.

### .gitignore Content

```
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

### README.md Content

The README will include:

1. **What is Ralph?** - Overview of the design → plan → execute workflow
2. **Installation** - How to clone ralph into a project
3. **Prompt Customization** - How to copy and customize prompts for the project (REQUIRED)
4. **Configuration** - How to create and customize `.env` file
5. **Usage** - Command-line options and examples
6. **Container Support** - Using `--container` and `--workdir` for dev containers
7. **Workflow Phases** - Description of design, plan, and execute phases
8. **Integration with AI Assistants** - Optional symlink setup for slash commands

Example installation instructions:
```bash
# Clone ralph into your project
git clone https://github.com/username/ralph.git ralph

# Add ralph/ to .gitignore
echo "ralph/" >> .gitignore

# Copy and customize prompts (REQUIRED - prompts are project-specific)
cp ralph/prompts/design.example.md ralph/prompts/design.md
cp ralph/prompts/plan.example.md ralph/prompts/plan.md
cp ralph/prompts/execute.example.md ralph/prompts/execute.md
# Edit each prompt to reference your project's specific documentation

# (Optional) Create .env configuration
cp ralph/.env.example ralph/.env
# Edit ralph/.env with project-specific settings

# (Optional) Symlink prompts for Claude/Codex slash commands
mkdir -p .claude/commands
ln -s ../../ralph/prompts/design.md .claude/commands/design.md
ln -s ../../ralph/prompts/plan.md .claude/commands/plan.md
ln -s ../../ralph/prompts/execute.md .claude/commands/execute.md
```

## Changes Required

### In New Ralph Repository

1. **Create repository structure**
   - Create `ralph` repository
   - Create directory structure (prompts/, plans/)
   - Copy prompt files from code_scout to prompts/ as `.example.md` templates
   - Rename `ralph` script to `start`
   - Add helpful error messages if prompts are missing

2. **Update start script**
   - Remove hardcoded `CONTAINER_WORKDIR`
   - Add `--workdir` flag with `/<basename>` default
   - Add `.env` file loading at script start
   - Update all path variables to use new defaults
   - Support ENV var overrides for all configurable values
   - Update default paths to `ralph/prompts/*` and `ralph/plans/*`
   - Add helpful error messages when prompts are missing (guide users to copy from .example.md)

3. **Create configuration files**
   - Create `.env.example` with defaults and comments
   - Create `.gitignore` with plans/, .env, prompts/*.md, log files

4. **Create README.md**
   - Installation instructions
   - Configuration guide
   - Usage examples
   - Container setup documentation
   - Slash command integration instructions

### In Code Scout Project

1. **Clone ralph repository**
   - Clone ralph repository: `git clone https://github.com/username/ralph.git ralph`
   - Add `ralph/` to `.gitignore` (already done)

2. **Create symlinks for slash commands**
   - `ln -s ../../ralph/prompts/design.md .claude/commands/design.md`
   - `ln -s ../../ralph/prompts/plan.md .claude/commands/plan.md`
   - `ln -s ../../ralph/prompts/execute.md .claude/commands/execute.md`

3. **Migrate planning documents**
   - Move `docs/planning/SPECIFICATION.md` to `ralph/plans/SPECIFICATION.md` (if exists and active)
   - Move `docs/planning/EXECUTION_PLAN.md` to `ralph/plans/EXECUTION_PLAN.md` (if exists and active)

4. **Update .gitignore**
   - Already has `ralph/` entry
   - Log files (`ralph-error.md`, `ralph-output.md`) removed from .gitignore since they're now created in project root

5. **Update documentation**
   - Update CLAUDE.md if it references ralph paths
   - Update any other docs that reference the workflow

6. **Remove old files**
   - Delete `ralph-script` backup from project root (keep for now during transition)
   - Delete `.claude/commands/*.md` files (replaced by symlinks)

## Handoff Feature

**Purpose:** After each execution pass, ralph reconnects to the most recent AI session and runs a handoff prompt to provide context for the next iteration or for future sessions.

**When it runs:** ONLY during the execute phase (not design or plan phases)

**How it works:**
1. At the end of each pass through the execute loop, after the callback (if any)
2. Ralph reconnects to the most recent AI session using session continuation flags
3. Ralph runs the handoff prompt (`ralph/prompts/handoff.md`)
4. The AI provides a summary, status update, or other handoff information

**Session continuation commands:**
- **Claude:** `claude --continue` or `claude -c` reconnects to the most recent session
- **Codex:** `codex resume --last` reconnects to the most recent session

**Handoff prompt:**
- Location: `ralph/prompts/handoff.md` (hardcoded, project-specific)
- Template: `ralph/prompts/handoff.example.md` (committed to ralph repository)
- Like other prompts, users must copy from .example.md and customize for their project

**Error handling:**
- If handoff command fails, log error but continue (don't stop the workflow)
- Handoff failures are non-fatal

## Container Working Directory Behavior

The `--workdir` flag sets the working directory inside the container when using `docker exec` or `podman exec`.

**Default behavior:**
- If `--container` is specified without `--workdir`, ralph uses `/<basename>` where basename is the current directory name
- Example: Running from `/Users/jlanders/gitlab_local/code_scout` → defaults to `/code_scout`

**Failure modes:**
- If the specified workdir doesn't exist in the container, `docker exec` fails immediately with clear error
- User must provide correct `--workdir` explicitly in this case

**Configuration:**
- Can be set via `--workdir` CLI flag
- Can be set via `CONTAINER_WORKDIR` in `.env` or environment
- CLI flag takes precedence over `.env`/environment

## Migration Path for Existing Projects

For projects already using ralph (like code_scout):

1. Clone ralph repository into project: `git clone https://github.com/username/ralph.git ralph`
2. Add `ralph/` to project's `.gitignore`
3. Create `.env` if customization is needed (otherwise use defaults)
4. Create symlinks to ralph/prompts/* if using slash commands
5. Update invocation from `./ralph-script` to `ralph/start`
6. Migrate any active planning documents to `ralph/plans/`
7. Remove old ralph script backup and prompt files

## Non-Goals

1. **Not changing workflow** - The design → plan → execute workflow remains unchanged
2. **Not modifying prompt content** - Prompts move as-is; content refinement is a separate effort
3. **Not adding new features** - This is a refactoring for reusability, not feature development
4. **Not supporting multiple projects simultaneously** - Ralph still assumes one active project at a time

## Success Criteria

1. ✅ Ralph repository exists and is structured as specified
2. ✅ `ralph/start` script supports all configuration options
3. ✅ `.env.example` documents all configurable variables with defaults
4. ✅ `.gitignore` properly excludes ephemeral files
5. ✅ README.md provides complete setup and usage documentation
6. ✅ Code scout project successfully uses ralph as cloned repository
7. ✅ Code scout's existing workflow continues to function
8. ✅ `--container` flag works with configurable `--workdir`
9. ✅ `.env` file loading works correctly with ENV var precedence
10. ✅ Ralph can be cloned into other projects and works out-of-the-box
