# Developer Guide

## Building code-scout

This project uses **lancedb-go**, which requires native C libraries and special CGO configuration.

### Prerequisites

- **Go 1.17+**
- **CGO enabled** (required for C interoperability)
- **curl** (for downloading native libraries)

**Note:** You do NOT need Rust or a C compiler toolchain - the libraries are pre-built.

### Docker Dev Container

If you prefer a containerized development environment, see
[docs/guides/DEV_CONTAINER.md](docs/guides/DEV_CONTAINER.md) for the Docker
dev container setup (Go toolchain, codex CLI, and build dependencies).
The guide also covers running `./ralph --container <name>` against a long-lived
container via `podman exec`.

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

**Important:** After downloading, fix the hardcoded install_name paths in macOS dylibs:

```bash
./fix-dylib-paths.sh
```

This fixes the install_name to use `@rpath` instead of the hardcoded CI build paths.

#### Linux ARM64 Native Libraries

Linux ARM64 does not have prebuilt LanceDB artifacts. The build scripts can
automatically build them from source when needed. See
[docs/guides/LINUX_ARM64_BUILD.md](docs/guides/LINUX_ARM64_BUILD.md) for the
full workflow, dependency list, and overrides.

#### 2. Build the Project (per-platform bundles)

The build now outputs a self-contained bundle for each platform under `dist/code-scout-<os>_<arch>/` and also produces a matching `code-scout-<os>_<arch>.tar.gz` archive for distribution.

```bash
# Build every platform that has native libs under ./lib/
./build.sh

# Or limit to specific targets
TARGETS="darwin_arm64 linux_amd64" ./build.sh
```

### Building Linux Bundles via Container

If you need Linux binaries from macOS/Windows, use the dev container
workflow in [docs/guides/DEV_CONTAINER.md](docs/guides/DEV_CONTAINER.md):

```bash
docker compose -f docker-compose.dev.yml run --rm dev \
  TARGETS=linux_amd64 ./build.sh
```

The repo is bind-mounted, so `dist/` updates on the host.

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

### Understanding the dist/ Folder

After running `./build.sh`, the `dist/` folder contains everything needed to run code-scout:

```
dist/
├── code-scout-darwin_arm64/          # macOS Apple Silicon bundle
│   ├── code-scout                    # Wrapper script (run this!)
│   ├── code-scout.bin                # Actual Go binary
│   ├── tei-wrapper                   # TEI wrapper binary
│   └── lib/
│       └── liblancedb_go.dylib       # Native LanceDB library
├── code-scout-darwin_amd64/          # macOS Intel bundle
│   └── ...
├── code-scout-linux_amd64/           # Linux x86_64 bundle
│   └── ...
├── code-scout-darwin_arm64.tar.gz    # Distributable archive
├── code-scout-darwin_amd64.tar.gz
└── code-scout-linux_amd64.tar.gz
```

**Why two executables?** The `code-scout` wrapper script sets the library path (`DYLD_LIBRARY_PATH` on macOS, `LD_LIBRARY_PATH` on Linux) before launching `code-scout.bin`. This is required because:
- LanceDB uses native C libraries that must be found at runtime
- Sandboxed environments (like some IDEs or CI systems) reset environment variables
- The wrapper ensures the binary always finds its libraries regardless of how it's launched

**Why a bundled tei-wrapper?** `tei-wrapper` is included for convenience so you can run it from the same bundle as the CLI. It still expects `text-embeddings-router` to be available on your PATH (or use `--tei-binary`).

**Which platform bundle to use:**
```bash
# Detect your platform
uname -sm
# "Darwin arm64"  → use darwin_arm64
# "Darwin x86_64" → use darwin_amd64
# "Linux x86_64"  → use linux_amd64
```

### Dogfooding During Development

When working on code-scout, use the built binary to search the codebase itself:

```bash
# Build first (if not already built)
./build.sh

# Detect platform and set an alias (optional but convenient)
case "$(uname -sm)" in
  "Darwin arm64")  CS=./dist/code-scout-darwin_arm64/code-scout ;;
  "Darwin x86_64") CS=./dist/code-scout-darwin_amd64/code-scout ;;
  "Linux x86_64")  CS=./dist/code-scout-linux_amd64/code-scout ;;
esac

# Index the repo
$CS index

# Search semantically
$CS search "tree-sitter parsing"
$CS search "embedding client" --json
```

**Tip:** After making code changes, rebuild with `./build.sh` and re-index to search your latest code.

### Development Workflow

#### Rebuilding After Changes

After making code changes, simply run:

```bash
./build.sh
```

#### Running Tests

**Recommended:** Use the Makefile for automatic platform detection and CGO configuration:

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage report
make test-coverage

# Run only integration tests
make test-integration

# See all available targets
make help
```

The Makefile automatically:
- Detects your platform (darwin_arm64, darwin_amd64, linux_amd64)
- Sets correct CGO_CFLAGS and CGO_LDFLAGS
- Sets library paths (DYLD_LIBRARY_PATH or LD_LIBRARY_PATH)
- Excludes example code from tests

**Alternative:** If you prefer to run tests manually without the Makefile:

```bash
# macOS Apple Silicon example
export CGO_CFLAGS="-I$(pwd)/include"
export CGO_LDFLAGS="-L$(pwd)/lib/darwin_arm64 -llancedb_go -framework Security -framework CoreFoundation -Wl,-rpath,$(pwd)/lib/darwin_arm64"
export DYLD_LIBRARY_PATH="$(pwd)/lib/darwin_arm64:${DYLD_LIBRARY_PATH}"

go test $(go list ./... | grep -v /examples/)
```

### Common Issues

#### "undefined symbol" linker errors

**Symptom:**
```
Undefined symbols for architecture arm64:
  "_simple_lancedb_connect", referenced from:
```

**Solution:**
- Use `make test` which sets CGO flags automatically
- Or manually set `CGO_LDFLAGS` as shown in the manual testing section

#### "Library not loaded" with hardcoded CI path

**Symptom:**
```
dyld: Library not loaded: /Users/runner/work/lancedb-go/lancedb-go/rust/target/...
```

**Solution:** Run `./fix-dylib-paths.sh` to fix the install_name in downloaded dylibs. This is a one-time fix needed after downloading the libraries.

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
│   ├── code-scout/        # Main CLI entry point
│   │   ├── main.go        # Root command
│   │   ├── index.go       # Index command
│   │   ├── search.go      # Search command
│   │   └── daemon.go      # Background daemon commands
│   └── tei-wrapper/       # TEI wrapper (model hot-swapping)
│       ├── main.go        # Wrapper server
│       └── README.md      # Developer documentation
├── internal/
│   ├── scanner/           # File scanning with ignore patterns
│   ├── chunker/           # Semantic code chunking (Tree-sitter)
│   ├── embeddings/        # Embedding clients (OpenAI-compatible)
│   └── storage/           # LanceDB storage layer
├── docs/
│   └── guides/            # User documentation
│       ├── TEI_SETUP.md           # TEI installation guide
│       ├── TEI_WRAPPER.md         # TEI wrapper guide
│       └── BACKGROUND_DAEMON.md   # Background daemon guide
├── include/               # C headers (from LanceDB)
├── lib/                   # Native libraries (from LanceDB)
├── .code-scout/           # Local vector database (runtime)
├── test_data/             # Test fixtures
└── examples/              # LanceDB usage examples
```

### TEI Wrapper Development

The TEI wrapper is a standalone Go HTTP server that provides OpenAI-compatible API access to TEI with automatic model hot-swapping.

**Build the wrapper:**
```bash
cd cmd/tei-wrapper
go build -o tei-wrapper .
```

**Run locally:**
```bash
# Start with default settings (port 11434, TEI on 8080)
./tei-wrapper

# With custom settings
./tei-wrapper --port 8081 --model nomic-ai/CodeRankEmbed --idle-preload
```

**Run tests:**
```bash
cd cmd/tei-wrapper
go test -v
```

**Key files:**
- `cmd/tei-wrapper/main.go` - Main server implementation
- `cmd/tei-wrapper/server_test.go` - Unit tests
- `cmd/tei-wrapper/mock_tei_test.go` - Mock TEI server for testing
- `cmd/tei-wrapper/README.md` - Developer documentation

**See also:**
- [TEI Wrapper Guide](docs/guides/TEI_WRAPPER.md) - User documentation

### Background Daemon Development

The background daemon is built into the `code-scout` CLI and automatically re-indexes the codebase when files change.

**Run daemon locally:**
```bash
# Start daemon
./code-scout daemon start

# Check status
./code-scout daemon status

# View logs
./code-scout daemon logs

# Stop daemon
./code-scout daemon stop
```

**Development:**
```bash
# Run daemon in foreground (for debugging)
cd cmd/code-scout
go run . daemon run
```

**Testing:**
```bash
# Run daemon tests
cd cmd/code-scout
go test -v -run TestDaemon
```

**Key files:**
- `cmd/code-scout/daemon.go` - Daemon implementation
- `cmd/code-scout/daemon_test.go` - Tests
- `internal/scanner/` - File scanning with ignore patterns

**Dependencies:**
- [fsnotify](https://github.com/fsnotify/fsnotify) - Cross-platform file system notifications

**See also:**
- [Background Daemon Guide](docs/guides/BACKGROUND_DAEMON.md) - User documentation

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
