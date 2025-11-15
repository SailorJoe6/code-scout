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

Run the artifact downloader script to get platform-specific libraries (if not already present):

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

#### 2. Build the Project (per-platform bundles)

The build now outputs a self-contained bundle for each platform under `dist/code-scout-<os>_<arch>/` and also produces a matching `code-scout-<os>_<arch>.tar.gz` archive for distribution.

```bash
# Build every platform that has native libs under ./lib/
./build.sh

# Or limit to specific targets
TARGETS="darwin_arm64 linux_amd64" ./build.sh
```

For each target, the script:
- Sets `GOOS`, `GOARCH`, and the matching LanceDB CGO flags
- Links in an rpath that points to the bundled `lib/` directory
- Copies the correct native libraries next to the binary
- Tars everything into `dist/code-scout-<os>_<arch>.tar.gz`
- Renames the compiled binary to `code-scout.bin` and adds a platform-specific `code-scout` wrapper that sets the right library path before delegating to the binary

#### Running After Build

**For local development** (after `./build.sh`):

The build script creates ready-to-run bundles in `dist/`. Run directly without extracting:

```bash
# On Apple Silicon
./dist/code-scout-darwin_arm64/code-scout --help

# On Intel macOS
./dist/code-scout-darwin_amd64/code-scout --help

# On Linux x86_64
./dist/code-scout-linux_amd64/code-scout --help
```

**For distribution** (sharing builds with others):

The `.tar.gz` archives are self-contained and can be extracted anywhere:

```bash
tar -xzf dist/code-scout-darwin_arm64.tar.gz
./code-scout-darwin_arm64/code-scout --help
```

**Important:** Always launch the wrapper (`code-scout`), which ensures `DYLD_LIBRARY_PATH`/`LD_LIBRARY_PATH` points at the bundled `lib/` directory even inside sandboxed shells. You only need `code-scout.bin` if you are debugging without the wrapper, in which case export the library path manually.

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

**Solution:** Run the packaged wrapper from the extracted bundle (for example `code-scout-darwin_arm64/code-scout`), which exports the correct library path before chaining to `code-scout.bin`. If you run `code-scout.bin` directly, `go run`, or another ad-hoc build, you must export `DYLD_LIBRARY_PATH`/`LD_LIBRARY_PATH` manually as before.

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
