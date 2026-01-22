# Linux ARM64 Builds (LanceDB Native Libraries)

Code Scout relies on native LanceDB libraries. The upstream `lancedb-go` project
does not ship prebuilt Linux ARM64 artifacts, so Code Scout can build them from
source when needed.

## Automatic Build Flow

When you run `./build.sh` or `./build-and-run.sh` on Linux ARM64:

- If `lib/linux_arm64/liblancedb_go.{a,so}` are missing or invalid, the build
  scripts invoke `scripts/build-lancedb-linux-arm64.sh`.
- The helper script downloads the latest `lancedb-go` release tag, builds the
  Rust native library, and installs the artifacts into `lib/linux_arm64/`.
- The build scripts then continue normally and output the bundle in
  `dist/code-scout-linux_arm64/`.

## Dependencies

The helper script installs the dependencies below via your system package
manager (apt, dnf, yum, or apk). If your system is not supported, install these
manually before retrying:

- Build tools (gcc/clang, make)
- `cmake`
- `pkg-config`
- `git`
- `curl`
- `protobuf-compiler`
- Rust toolchain (`cargo`, `rustc`)

The script installs Rust with `rustup` if it is not already available.

## Overrides

You can override the default behavior with environment variables:

```bash
# Build a specific tag instead of the latest release
LANCEDB_GO_TAG=v0.15.0 ./build.sh

# Use a custom cache directory for the lancedb-go clone
LANCEDB_GO_CACHE_DIR=/tmp/lancedb-go-cache ./build.sh

# Use a fork of lancedb-go
LANCEDB_GO_REPO=https://github.com/your-org/lancedb-go.git ./build.sh
```

## Troubleshooting

If the build fails:

1. Confirm the dependencies are installed.
2. Remove the cache directory and retry:
   ```bash
   rm -rf .cache/lancedb-go
   ./build.sh
   ```
3. Inspect the helper output for missing build steps or missing Rust packages.

The helper validates that the resulting shared library is `AArch64` using
`readelf` when available.
