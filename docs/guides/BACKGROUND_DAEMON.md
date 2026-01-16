# Background Indexing Daemon

The background indexing daemon automatically re-indexes your codebase when files change, eliminating the need to manually run `code-scout index` after every code change.

## Overview

### The Problem

**Without the daemon:**
1. Edit code files
2. Remember to run `code-scout index`
3. Wait for indexing to complete
4. Run `code-scout search`

**Pain points:**
- Easy to forget to re-index
- Interrupts workflow
- Search results become stale
- Manual indexing is tedious

### The Solution

**With the daemon running:**
1. Edit code files
2. ~~Remember to run `code-scout index`~~ ← Daemon does this automatically
3. ~~Wait for indexing to complete~~ ← Happens in background
4. Run `code-scout search` ← Always fresh results!

**Benefits:**
- ✅ Zero manual indexing
- ✅ Always up-to-date search results
- ✅ Automatic debouncing (waits for editing to finish)
- ✅ Respects `.gitignore` and `.code-scout-ignore`
- ✅ Runs in background (doesn't block terminal)

## How It Works

The daemon uses file system watching to detect changes and trigger re-indexing:

```
File Change (write/create/delete/rename)
       ↓
Daemon detects change
       ↓
Start 5-second debounce timer
       ↓
More changes? → Reset timer
       ↓
No more changes for 5 seconds
       ↓
Run incremental indexing
       ↓
Update search database
       ↓
Ready for searches (fresh results!)
```

### Key Features

1. **File Watching** - Monitors all directories in your repository
2. **Debouncing** - Waits 5 seconds after last change before indexing
3. **Incremental Indexing** - Only re-indexes changed files
4. **Ignore Patterns** - Respects `.gitignore` and `.code-scout-ignore`
5. **Graceful Shutdown** - Handles SIGTERM and SIGINT cleanly

## Installation

The daemon is built into the `code-scout` CLI. No separate installation needed!

```bash
# Check daemon is available
./code-scout daemon --help
```

## Usage

### Starting the Daemon

```bash
# Start daemon in background
code-scout daemon start
```

**Output:**
```
✓ Daemon started (PID: 12345)
  Log file: .code-scout/daemon.log
```

**What happens:**
1. Daemon process starts in background
2. PID file created at `.code-scout/daemon.pid`
3. Log file created at `.code-scout/daemon.log`
4. Initial index runs after 2-second startup delay
5. File watcher starts monitoring for changes

### Checking Daemon Status

```bash
# Check if daemon is running
code-scout daemon status
```

**Output (Running):**
```
Status: Running (PID: 12345)
Log file: .code-scout/daemon.log
Last activity: 2026-01-16 10:30:45 Indexing complete
```

**Output (Not Running):**
```
Status: Not running
```

### Viewing Daemon Logs

```bash
# Show full daemon log
code-scout daemon logs
```

**Example log output:**
```
2026/01/16 10:25:30 Daemon starting...
2026/01/16 10:25:30 Watching 25 directories for changes
2026/01/16 10:25:32 Running initial index...
2026/01/16 10:25:45 Initial indexing complete
2026/01/16 10:30:15 File change detected: internal/storage/store.go (WRITE)
2026/01/16 10:30:20 Debounce timeout reached, starting indexing...
2026/01/16 10:30:45 Indexing complete
```

### Stopping the Daemon

```bash
# Stop daemon gracefully
code-scout daemon stop
```

**Output:**
```
✓ Daemon stopped
```

**What happens:**
1. SIGTERM signal sent to daemon process
2. Daemon finishes current indexing (if any)
3. File watcher stops
4. Process exits
5. PID file removed

## Configuration

### Debounce Delay

The daemon waits 5 seconds after the last file change before starting indexing. This delay is hardcoded but will be configurable in a future release.

**Current behavior:**
- File change detected → Start 5-second timer
- Another change within 5 seconds → Reset timer
- No changes for 5 seconds → Start indexing

**Why 5 seconds?**
- Short enough to feel responsive
- Long enough to avoid re-indexing during active editing
- Balances responsiveness vs resource usage

### Indexing Concurrency

The daemon uses these default settings for indexing:

```bash
# Daemon equivalent to:
code-scout index --workers 10 --batch-size 8
```

These settings provide good performance without overwhelming the system. They will be configurable in a future release.

### File Watching

The daemon watches all directories in your repository except:
- Hidden directories (starting with `.`)
- Directories matching `.gitignore` patterns
- Directories matching `.code-scout-ignore` patterns

**Watched events:**
- File writes (edits)
- File creates (new files)
- File deletes (removed files)
- File renames (moved files)

**Ignored events:**
- Directory operations (not relevant for indexing)
- Hidden files (starting with `.`)
- Files matching ignore patterns

## Process Management

### PID File

Location: `.code-scout/daemon.pid`

Contains the daemon's process ID:
```
12345
```

Used by `daemon stop` and `daemon status` commands to track the daemon.

### Log File

Location: `.code-scout/daemon.log`

Contains timestamped daemon activity:
```
2026/01/16 10:25:30 Daemon starting...
2026/01/16 10:25:30 Watching 25 directories for changes
2026/01/16 10:25:32 Running initial index...
```

Append-only (grows over time). Safe to truncate or delete while daemon is stopped.

### Signal Handling

The daemon handles these signals:

**SIGTERM** (graceful shutdown):
```bash
kill -TERM $(cat .code-scout/daemon.pid)
# Or use: code-scout daemon stop
```

**SIGINT** (Ctrl+C, if running in foreground):
```bash
^C
2026/01/16 10:35:00 Received shutdown signal
2026/01/16 10:35:00 Shutting down...
```

Both signals trigger graceful shutdown:
1. Stop accepting new file change events
2. Complete current indexing (if any)
3. Close file watcher
4. Exit cleanly

## AI Agent Integration

The background daemon is **specifically designed for AI coding agents** like Claude Code.

### Recommended Workflow for AI Agents

**With daemon running:**

```python
# 1. Start daemon (once per repository)
run("code-scout daemon start")

# 2. Make code changes
edit_file("main.go", new_content)
create_file("utils.go", utility_code)

# 3. Search immediately (daemon handles indexing)
results = run("code-scout search 'authentication'")
# Results are fresh (daemon auto-indexed changes)
```

**Without daemon (manual indexing):**

```python
# 1. Make code changes
edit_file("main.go", new_content)

# 2. Remember to re-index
run("code-scout index")  # Easy to forget!

# 3. Search
results = run("code-scout search 'authentication'")
```

### Benefits for AI Agents

1. **Simpler Agent Code** - No need to track which files changed
2. **Always Fresh Results** - Search results reflect latest code
3. **Background Processing** - Indexing doesn't block agent
4. **Automatic Debouncing** - No duplicate indexing during multi-file edits

### Best Practices for AI Agents

**✅ DO:**
- Start daemon at beginning of session: `code-scout daemon start`
- Use search without manual indexing: `code-scout search "query"`
- Stop daemon at end of session: `code-scout daemon stop`
- Check daemon status if searches seem stale: `code-scout daemon status`

**❌ DON'T:**
- Run `code-scout index` manually when daemon is running (wasteful)
- Forget to stop daemon when session ends (uses resources)
- Start multiple daemons (only one per repository)

### Example: Claude Code Integration

```bash
# In .claude/hooks/session-start.sh
code-scout daemon start

# During session - just search, no manual indexing!
code-scout search "error handling"

# In .claude/hooks/session-end.sh
code-scout daemon stop
```

## Integration with TEI Wrapper

The daemon currently uses Code Scout's default embedding configuration (Ollama endpoint on port 11434).

**If using TEI wrapper:**
1. Start TEI wrapper first
2. Start daemon (will use wrapper automatically)

```bash
# Start TEI wrapper (port 11434)
./tei-wrapper &

# Start daemon (uses wrapper automatically)
code-scout daemon start
```

**Future enhancement:** The daemon will support starting/managing the TEI wrapper automatically.

## Troubleshooting

### Daemon won't start: "already running"

**Problem:** PID file exists from previous session

**Check:**
```bash
cat .code-scout/daemon.pid
# Shows PID, e.g. 12345

# Check if process is actually running
ps -p 12345
```

**Solution (if process is dead):**
```bash
# Remove stale PID file
rm .code-scout/daemon.pid

# Start daemon
code-scout daemon start
```

### Daemon not detecting file changes

**Problem:** File changes aren't triggering re-indexing

**Check logs:**
```bash
code-scout daemon logs | tail -20
```

**Likely causes:**
1. File is in `.gitignore` or `.code-scout-ignore`
2. File is hidden (starts with `.`)
3. File is in hidden directory
4. Daemon not watching the directory

**Debug:**
```bash
# Make a test change
echo "// test" >> main.go

# Check if daemon logged it
code-scout daemon logs | grep "File change detected"
```

### Indexing too slow

**Problem:** Daemon indexing takes too long

**Current settings (hardcoded):**
- Workers: 10
- Batch size: 8

**Future solution:** Configuration file will allow tuning:
```json
{
  "daemon": {
    "workers": 6,
    "batch_size": 4,
    "debounce_delay": "3s"
  }
}
```

**Temporary workaround:**
1. Stop daemon: `code-scout daemon stop`
2. Index manually with lower concurrency: `code-scout index --workers 4 --batch-size 4`
3. Restart daemon: `code-scout daemon start`

### High CPU usage

**Problem:** Daemon using too much CPU

**Likely causes:**
1. Watching too many directories (large repository)
2. Frequent file changes (active editing)
3. Indexing not finishing before next change

**Check:**
```bash
# See what daemon is doing
code-scout daemon logs | tail -50
```

**Solutions:**
1. Add build directories to `.code-scout-ignore`
2. Increase debounce delay (future config option)
3. Use manual indexing instead for very large repos

### Daemon crashes or stops unexpectedly

**Check exit status:**
```bash
code-scout daemon status
# Status: Not running
```

**Check logs for errors:**
```bash
code-scout daemon logs | tail -100
```

**Common causes:**
- Out of memory (reduce workers/batch size)
- Permission errors (can't write to `.code-scout/`)
- Embedding server not responding (check TEI/Ollama)

**Recover:**
```bash
# Check for stale PID file
rm -f .code-scout/daemon.pid

# Restart daemon
code-scout daemon start
```

## Development Status

**Implemented:**
- ✅ File system watching (fsnotify)
- ✅ Debouncing (5-second delay)
- ✅ Incremental indexing
- ✅ Process management (start/stop/status/logs)
- ✅ PID file tracking
- ✅ Log file output
- ✅ Signal handling (SIGTERM/SIGINT)
- ✅ Initial index on startup

**Future Enhancements:**
- Configuration file for tuning (workers, batch size, debounce delay)
- TEI wrapper lifecycle management
- Automatic embedding server detection
- Systemd/launchd service files
- Metrics and monitoring endpoints
- Multiple repository support

## Files and Directories

### Created by Daemon

```
.code-scout/
├── daemon.pid          # Process ID (created on start, removed on stop)
├── daemon.log          # Activity log (append-only)
├── index.db/           # LanceDB database (created by indexing)
└── metadata.json       # Indexing metadata (file mod times)
```

### Source Code

```
cmd/code-scout/daemon.go      # Main daemon implementation
cmd/code-scout/daemon_test.go # Tests
```

## Comparison: Manual vs Background Indexing

| Feature | Manual Indexing | Background Daemon |
|---------|----------------|-------------------|
| **User action** | Run `code-scout index` | One-time `daemon start` |
| **When to index** | After every change | Automatic |
| **Search freshness** | Depends on discipline | Always fresh |
| **Workflow interruption** | Yes (blocks) | No (background) |
| **AI agent complexity** | Must track changes | Just search |
| **Resource usage** | Only when indexing | Continuous (light) |
| **Best for** | CI/CD, git hooks | Development, AI agents |

## Next Steps

- **For TEI wrapper setup:** See [TEI_WRAPPER.md](TEI_WRAPPER.md)
- **For embedding configuration:** See [TEI_SETUP.md](TEI_SETUP.md) or [OLLAMA_SETUP.md](OLLAMA_SETUP.md)
- **For contributing:** See [DEVELOPERS.md](../../DEVELOPERS.md)
