# Developer Guide

## Building code-scout

This project uses **lancedb-go**, which requires native C libraries and special CGO configuration.

### Prerequisites

- **Go 1.17+**
- **CGO enabled** (required for C interoperability)
- **curl** (for downloading native libraries)

**Note:** You do NOT need Rust or a C compiler toolchain - the libraries are pre-built.

### Initial Setup

#### 1. Download Native Libraries

Run the artifact downloader script to get platform-specific libraries:

```bash
curl -sSL https://raw.githubusercontent.com/lancedb/lancedb-go/main/scripts/download-artifacts.sh | bash
```

Or for a specific version:
```bash
curl -sSL https://raw.githubusercontent.com/lancedb/lancedb-go/main/scripts/download-artifacts.sh | bash -s v1.0.0
```

This creates:
- `lib/{platform}_{arch}/` - Platform-specific native libraries
- `include/lancedb.h` - Required C header file

#### 2. Build the Project

Use the provided build script (automatically detects your platform):

```bash
./build.sh
```

The script will:
- Detect your OS and architecture
- Set the required CGO flags for your platform
- Build the `code-scout` binary

**Note:** After building, you may need to set the library path for runtime:

**macOS:**
```bash
export DYLD_LIBRARY_PATH=$(pwd)/lib/darwin_arm64:$DYLD_LIBRARY_PATH
```

**Linux:**
```bash
export LD_LIBRARY_PATH=$(pwd)/lib/linux_amd64:$LD_LIBRARY_PATH
```

Then run the binary:
```bash
./code-scout --help
```

### Development Workflow

#### Rebuilding After Changes

After making code changes, simply run:

```bash
./build.sh
```

#### Running Tests

The build script sets environment variables for the current shell session only. For tests, you can source the script's logic or set the flags manually:

```bash
# macOS example (adjust for your platform)
export CGO_CFLAGS="-I$(pwd)/include"
export CGO_LDFLAGS="-L$(pwd)/lib/darwin_arm64 -llancedb_go -framework Security -framework CoreFoundation"

go test ./...
```

### Common Issues

#### "undefined symbol" linker errors

**Symptom:**
```
Undefined symbols for architecture arm64:
  "_simple_lancedb_connect", referenced from:
```

**Solution:** You forgot to set `CGO_LDFLAGS`. See step 2 above.

#### "lancedb.h: No such file or directory"

**Symptom:**
```
fatal error: lancedb.h: No such file or directory
```

**Solution:** Run the artifact downloader script (step 1) or set `CGO_CFLAGS` correctly (step 2).

#### Libraries not found at runtime

**Symptom:**
```
dyld: Library not loaded: liblancedb_go.dylib
```

**Solution:** Set the library path environment variable before running:

**macOS:**
```bash
export DYLD_LIBRARY_PATH=$(pwd)/lib/darwin_arm64:$DYLD_LIBRARY_PATH
./code-scout index .
```

**Linux:**
```bash
export LD_LIBRARY_PATH=$(pwd)/lib/linux_amd64:$LD_LIBRARY_PATH
./code-scout index .
```

Add this to your shell profile (`.bashrc`, `.zshrc`, etc.) to make it permanent when working on this project.

### Project Structure

```
code-scout/
├── cmd/
│   └── code-scout/        # CLI entry point
│       ├── main.go
│       ├── index.go       # Index command
│       └── search.go      # Search command
├── internal/
│   ├── scanner/           # File scanning
│   ├── chunker/           # Code chunking
│   ├── embeddings/        # Ollama client
│   └── storage/           # LanceDB storage
├── include/               # C headers (from download script)
├── lib/                   # Native libraries (from download script)
├── .code-scout/           # Local vector database
├── test_data/             # Test fixtures
└── examples/              # LanceDB usage examples
```

### Issue Tracking

This project uses **beads (bd)** for issue tracking. See `AGENTS.md` for detailed instructions.

Quick reference:
```bash
# Check for ready work
bd ready --json

# Claim an issue
bd update <id> --status in_progress

# Complete an issue
bd close <id> --reason "Done"
```

### Resources

- [LanceDB Go SDK](https://github.com/lancedb/lancedb-go)
- [Project README](README.md)
- [Agent Guidelines](AGENTS.md)
