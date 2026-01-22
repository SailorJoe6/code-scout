# Specification: Local Linux ARM64 Build With LanceDB Native Libraries

## Overview

Add a repeatable, integrated build flow that supports Linux ARM64 (aarch64) in the current environment by building LanceDB native libraries from source when prebuilt artifacts are unavailable. The process must be automatic, use the latest release tag of `lancedb-go`, and output the native libraries into `lib/linux_arm64/` just like downloaded artifacts. The flow should be integrated into existing `build.sh` and `build-and-run.sh`, and documented appropriately in developer docs.

## Current Behavior

- `build.sh` and `build-and-run.sh` assume prebuilt LanceDB native libraries exist under `lib/<os>_<arch>/`.
- The official `lancedb-go` artifact download script does not provide `linux_arm64` libraries.
- On Linux ARM64, the current build fails due to missing or incompatible native libraries.

## Goals

- Support full `code-scout` builds on Linux ARM64 in the current environment.
- Automatically build LanceDB native libraries from source when `linux_arm64` is detected and no valid prebuilt artifacts exist.
- Use the latest release tag of `lancedb-go` for building native libraries.
- Place built artifacts in `lib/linux_arm64/` using the same layout as downloaded libraries.
- Integrate the process into `build.sh` and `build-and-run.sh` so users run the same scripts regardless of platform.
- Document the workflow in developer documentation without making `DEVELOPERS.md` unbalanced; add a new guide only if needed.

## Non-Goals

- Changing runtime behavior of `code-scout`.
- Modifying how non-Linux ARM64 platforms obtain native libraries.
- Replacing or removing the existing artifact download flow for supported platforms.

## Required Outcomes

### Build Flow

- When `build.sh` or `build-and-run.sh` targets `linux_arm64`:
  - Detect that prebuilt artifacts are missing or invalid for `linux_arm64`.
  - Automatically build LanceDB native libraries from the `lancedb-go` repo at the latest release tag.
  - Install or copy outputs into `lib/linux_arm64/`:
    - `liblancedb_go.a`
    - `liblancedb_go.so`
  - Ensure the libraries are valid for `aarch64` and usable by the Go build.

### Dependency Handling

- The workflow should install or provision required build dependencies (Rust toolchain, C compiler/toolchain, and any other required system packages) without manual intervention.
- Dependencies should be documented clearly in the developer docs.

### Script Integration

- `build.sh` should seamlessly support `linux_arm64` by triggering the native build when needed.
- `build-and-run.sh` should do the same before attempting to build and run.
- The automation must not impact existing flows on supported platforms.

### Documentation

- Add developer documentation describing:
  - When and why native libraries are built from source for `linux_arm64`.
  - Required dependencies and how they are installed.
  - How the process integrates with `build.sh` / `build-and-run.sh`.
  - Where outputs are placed (`lib/linux_arm64/`).
- If the content would unbalance `DEVELOPERS.md`, create a dedicated guide under `docs/guides/` and link to it from `DEVELOPERS.md`.

## Acceptance Criteria

- ✅ Running `./build.sh` on Linux ARM64 produces a working `code-scout` bundle under `dist/code-scout-linux_arm64/` using locally built LanceDB libraries.
- ✅ Running `./build-and-run.sh` on Linux ARM64 successfully builds and runs the `tei-wrapper` and the `code-scout` binary (subject to existing runtime dependencies).
- ✅ `lib/linux_arm64/liblancedb_go.{a,so}` are produced by the automated build and are valid ARM64 binaries.
- ✅ Developer docs clearly explain the Linux ARM64 workflow and the automation behavior.

## Implementation Status

**COMPLETED** - 2026-01-22

All acceptance criteria met and validated on Linux ARM64 (aarch64) hardware:
- LanceDB native libraries (v0.1.2) built from source as AArch64
- Build scripts (`build.sh`, `build-and-run.sh`) successfully integrate auto-build flow
- Distribution bundle created: `dist/code-scout-linux_arm64.tar.gz` (173MB)
- Binaries verified: `code-scout.bin`, `tei-wrapper`, and `liblancedb_go.so` all ARM64
- Functional validation passed: binaries execute without library or architecture errors
- Documentation complete: [docs/guides/LINUX_ARM64_BUILD.md](../guides/LINUX_ARM64_BUILD.md)
