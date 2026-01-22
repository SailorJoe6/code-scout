# Execution Plan: Local Linux ARM64 Build With LanceDB Native Libraries

## Summary

Implement an integrated, automatic Linux ARM64 build path that compiles LanceDB native libraries from the latest `lancedb-go` release tag when prebuilt artifacts are unavailable. Update `build.sh` and `build-and-run.sh` to use this flow and document it for developers.

## Assumptions / Constraints

- Target platform is Linux ARM64 (aarch64).
- Network access is available for pulling the latest `lancedb-go` release tag and cloning the repo.
- We can install build dependencies on this machine.
- Prebuilt LanceDB artifacts remain the default for other platforms.

## Work Breakdown

1. **Audit upstream `lancedb-go` native build process**
   - Locate the official build entrypoint (scripts/Makefile/Cargo commands).
   - Identify required system dependencies (Rust toolchain, compiler toolchain, cmake, pkg-config, etc.).
   - Determine where the built `liblancedb_go.{a,so}` live and how to export them.

2. **Create a Linux ARM64 native build helper**
   - Add a repo-local script (e.g., `scripts/build-lancedb-linux-arm64.sh`) that:
     - Detects Linux ARM64.
     - Installs missing dependencies (package-manager-aware, or document supported distro).
     - Resolves latest `lancedb-go` release tag (GitHub API) with optional override.
     - Clones/updates `lancedb-go` into a cache directory (e.g., `.cache/lancedb-go`).
     - Checks out the latest release tag.
     - Runs the native build and copies `liblancedb_go.a` and `liblancedb_go.so` into `lib/linux_arm64/`.
     - Validates the resulting `.so` architecture (aarch64) and indexes the `.a` (`ranlib`).

3. **Integrate helper into `build.sh` and `build-and-run.sh`**
   - Add a preflight step for target `linux_arm64`:
     - If `lib/linux_arm64/` is missing or contains incompatible binaries, invoke the helper.
   - Keep the existing artifact download flow for other platforms.
   - Ensure failure modes are explicit (missing deps/build errors).

4. **Update documentation**
   - Add Linux ARM64 build instructions to `DEVELOPERS.md` if it fits cleanly, or
     create a new guide in `docs/guides/` and link it from `DEVELOPERS.md`.
   - Document:
     - Auto-build behavior for Linux ARM64.
     - Required dependencies and install behavior.
     - Output locations (`lib/linux_arm64/`, `dist/code-scout-linux_arm64/`).

5. **Validation**
   - Run `./build.sh` on Linux ARM64 and confirm:
     - Native libs are built and placed in `lib/linux_arm64/`.
     - Bundle `dist/code-scout-linux_arm64/` is created.
   - Run `./build-and-run.sh` and confirm it completes successfully.

## Open Decisions / Follow-ups

- Exact dependency list for `lancedb-go` native build (confirm from upstream).
- Package manager support (apt-only vs. multi-distro).
- Where to cache `lancedb-go` sources and whether to allow override via env var.

## Status

- ✅ Created `scripts/build-lancedb-linux-arm64.sh` helper for native builds.
- ✅ Integrated helper into `build.sh` and `build-and-run.sh` preflight flow.
- ✅ Documented Linux ARM64 workflow in `docs/guides/LINUX_ARM64_BUILD.md` and linked from `DEVELOPERS.md`.
- ✅ **VALIDATION COMPLETE** (2026-01-22): Successfully built and validated on Linux ARM64.
  - Dependencies installed: Go 1.25.2, Rust 1.92.0, protoc 25.1
  - LanceDB native libraries built from source (v0.1.2) as AArch64
  - `dist/code-scout-linux_arm64/` bundle created with correct ARM64 binaries
  - Functional validation passed: binaries run without library or architecture errors
  - Integrated workflow (`build-and-run.sh`) completes successfully

## Unblocking Validation

### Current Blockers

1. **Rust toolchain missing** - `cargo` and `rustc` not installed
2. **Protocol Buffers compiler missing** - `protoc` not installed
3. **Go not on PATH** - Installed at `/usr/local/go/bin/go` but not accessible
4. **Wrong architecture artifacts** - `lib/linux_arm64/liblancedb_go.so` is x86_64, not ARM64
5. **Environment constraints** - No sudo access, network may be restricted

### Detailed Unblocking Steps

#### Step 1: Fix Go PATH

**Goal:** Make Go accessible to build scripts

**Commands:**
```bash
# Verify Go exists
ls -la /usr/local/go/bin/go

# Add to current shell
export PATH="/usr/local/go/bin:$PATH"

# Verify Go is now accessible
go version

# Make permanent (add to ~/.bashrc or ~/.profile)
echo 'export PATH="/usr/local/go/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

**Success criteria:** `go version` works and shows Go 1.17+

#### Step 2: Install Rust Toolchain (User-Local, No Sudo)

**Goal:** Install Rust toolchain for building LanceDB native libraries

**Commands:**
```bash
# Install rustup (Rust toolchain manager) in user home directory
# This does NOT require sudo
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

# Load Rust environment
source "$HOME/.cargo/env"

# Verify installation
rustc --version
cargo --version

# Make permanent (rustup installer should have added this to ~/.bashrc)
# Verify this line exists in ~/.bashrc:
grep 'source "$HOME/.cargo/env"' ~/.bashrc
```

**Network requirement:** Requires internet access to download rustup and Rust toolchain

**Success criteria:**
- `rustc --version` shows 1.70+
- `cargo --version` works
- `~/.cargo/bin` is on PATH

#### Step 3: Install Protocol Buffers Compiler (User-Local, No Sudo)

**Goal:** Install `protoc` without sudo access

**Option A: Download Prebuilt Binary (Preferred)**
```bash
# Set version
PROTOC_VERSION="25.1"
ARCH="aarch_64"  # or "x86_64" for Intel

# Download prebuilt protoc
cd /tmp
curl -LO "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-${ARCH}.zip"

# Extract to user-local directory
mkdir -p ~/.local
unzip "protoc-${PROTOC_VERSION}-linux-${ARCH}.zip" -d ~/.local

# Add to PATH
export PATH="$HOME/.local/bin:$PATH"

# Verify installation
protoc --version

# Make permanent
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
```

**Option B: Build from Source (If Network/Downloads Fail)**
```bash
# Clone protobuf repository
cd /tmp
git clone --depth 1 --branch v25.1 https://github.com/protocolbuffers/protobuf.git
cd protobuf

# Build and install to ~/.local
./autogen.sh
./configure --prefix=$HOME/.local
make -j$(nproc)
make install

# Add to PATH (if not already done)
export PATH="$HOME/.local/bin:$PATH"
```

**Success criteria:** `protoc --version` shows 3.0+

#### Step 4: Clean and Rebuild LanceDB Native Libraries

**Goal:** Build correct ARM64 native libraries from scratch

**Commands:**
```bash
# Navigate to code-scout root
cd /home/sailorjoe6/Code/code-scout

# Ensure PATH includes all required tools
export PATH="/usr/local/go/bin:$HOME/.cargo/bin:$HOME/.local/bin:$PATH"

# Clean existing (wrong architecture) artifacts
rm -rf lib/linux_arm64/
mkdir -p lib/linux_arm64/

# Run the build helper directly to see detailed output
./scripts/build-lancedb-linux-arm64.sh

# Verify artifacts are ARM64
readelf -h lib/linux_arm64/liblancedb_go.so | grep Machine
# Should show: Machine: AArch64

# Verify static archive is valid
ar t lib/linux_arm64/liblancedb_go.a
# Should list object files, not error
```

**Success criteria:**
- `lib/linux_arm64/liblancedb_go.so` exists and is ARM64 (readelf shows "AArch64")
- `lib/linux_arm64/liblancedb_go.a` exists and is valid (ar can read it)
- Script completes without errors

#### Step 5: Build code-scout Bundle

**Goal:** Create working `dist/code-scout-linux_arm64/` bundle

**Commands:**
```bash
# Ensure PATH is set
export PATH="/usr/local/go/bin:$HOME/.cargo/bin:$HOME/.local/bin:$PATH"

# Run full build (will use lib/linux_arm64/ created in step 4)
./build.sh

# Verify bundle structure
ls -la dist/code-scout-linux_arm64/
# Should show:
#   - code-scout (wrapper script)
#   - code-scout.bin (Go binary)
#   - tei-wrapper
#   - lib/liblancedb_go.so

# Verify binary architecture
file dist/code-scout-linux_arm64/code-scout.bin
# Should show: ARM aarch64

readelf -h dist/code-scout-linux_arm64/code-scout.bin | grep Machine
# Should show: Machine: AArch64
```

**Success criteria:**
- `dist/code-scout-linux_arm64/code-scout.bin` exists and is ARM64
- `dist/code-scout-linux_arm64/lib/liblancedb_go.so` exists and is ARM64
- `dist/code-scout-linux_arm64.tar.gz` is created

#### Step 6: Functional Validation

**Goal:** Verify code-scout actually works on ARM64

**Commands:**
```bash
# Test help command (basic functionality)
./dist/code-scout-linux_arm64/code-scout --help
# Should show usage information without errors

# Test version command
./dist/code-scout-linux_arm64/code-scout version
# Should show version without library errors

# Test index command (requires embedding endpoint)
# Note: This requires tei-wrapper or other embedding endpoint running
./dist/code-scout-linux_arm64/code-scout index
# Should index files without crashing
```

**Success criteria:**
- Commands run without "library not found" errors
- No "wrong ELF class" or architecture mismatch errors
- Can perform basic operations (help, version work at minimum)

#### Step 7: Run build-and-run.sh Validation

**Goal:** Verify the integrated workflow

**Commands:**
```bash
# Clean previous outputs
rm -rf dist/code-scout-linux_arm64/

# Run integrated build-and-run script
./build-and-run.sh

# Script should:
# 1. Detect missing/invalid lib/linux_arm64/ artifacts
# 2. Auto-build via scripts/build-lancedb-linux-arm64.sh
# 3. Build code-scout binary
# 4. Start tei-wrapper
# 5. Start code-scout daemon
```

**Success criteria:**
- Script completes without fatal errors
- `dist/code-scout-linux_arm64/` bundle is created
- Services start successfully (tei-wrapper, daemon)

### Alternative: Use Dev Container

If the above steps fail due to persistent network or permission issues, use the dev container approach:

```bash
# Build using Docker dev container (has all dependencies)
docker compose -f docker-compose.dev.yml run --rm dev bash

# Inside container:
export PATH="/usr/local/go/bin:$PATH"
./scripts/build-lancedb-linux-arm64.sh
./build.sh

# Artifacts appear on host due to bind mount
exit
```

### Validation Checklist

- [x] Go accessible on PATH (`go version` works) - Go 1.25.2
- [x] Rust installed (`cargo --version` works) - Rust 1.92.0
- [x] protoc installed (`protoc --version` works) - protoc 25.1
- [x] `lib/linux_arm64/liblancedb_go.so` is ARM64 (verified with readelf) - AArch64 ✓
- [x] `lib/linux_arm64/liblancedb_go.a` is valid (verified with ar) - 527MB ✓
- [x] `dist/code-scout-linux_arm64/code-scout.bin` exists and is ARM64 - ARM aarch64 ✓
- [x] `./dist/code-scout-linux_arm64/code-scout --help` works - Help displayed ✓
- [x] `./build-and-run.sh` completes successfully - Build + tei-wrapper startup ✓
- [x] No "library not found" or "wrong ELF class" errors - Clean execution ✓

### Post-Validation Tasks

Once validation succeeds:

1. **Update this execution plan** - Mark validation as complete
2. **Close beads issue** - Close `code_context-q4o` with success notes
3. **Update specification** - Mark spec as fully implemented
4. **Commit and push** - Land the validated implementation
5. **Update README** - Confirm Linux ARM64 as supported platform
