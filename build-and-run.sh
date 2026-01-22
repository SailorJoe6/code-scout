#!/bin/bash
set -euo pipefail

REPO_ROOT="$(pwd)"
LIB_DIR="${REPO_ROOT}/lib"
DIST_DIR="${REPO_ROOT}/dist"
CACHE_DIR="${REPO_ROOT}/.cache/go"
SCRIPTS_DIR="${REPO_ROOT}/scripts"
LANCEDB_LINUX_ARM64_SCRIPT="${SCRIPTS_DIR}/build-lancedb-linux-arm64.sh"

# Detect current platform
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    # Map architecture names to Go conventions
    case "$arch" in
        x86_64)
            arch="amd64"
            ;;
        aarch64)
            arch="arm64"
            ;;
        arm64)
            arch="arm64"
            ;;
        *)
            echo "Unsupported architecture: $arch"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}

validate_linux_arm64_libs() {
    local lib_path="${LIB_DIR}/linux_arm64"

    if [[ ! -f "${lib_path}/liblancedb_go.a" || ! -f "${lib_path}/liblancedb_go.so" ]]; then
        return 1
    fi

    if command -v ar >/dev/null 2>&1; then
        if ! ar t "${lib_path}/liblancedb_go.a" >/dev/null 2>&1; then
            return 1
        fi
    fi

    if command -v readelf >/dev/null 2>&1; then
        local machine
        machine="$(readelf -h "${lib_path}/liblancedb_go.so" | awk -F: '/Machine:/ {gsub(/^[[:space:]]+/, "", $2); print $2}')"
        if [[ "${machine}" != *"AArch64"* ]]; then
            return 1
        fi
    fi

    return 0
}

ensure_linux_arm64_libs() {
    if validate_linux_arm64_libs; then
        return
    fi

    if [[ ! -x "${LANCEDB_LINUX_ARM64_SCRIPT}" ]]; then
        echo "Missing Linux ARM64 build helper at ${LANCEDB_LINUX_ARM64_SCRIPT}"
        exit 1
    fi

    echo "Linux ARM64 libraries missing or invalid; building from source..."
    "${LANCEDB_LINUX_ARM64_SCRIPT}"
}

create_wrapper() {
    local os="$1"
    local output_dir="$2"
    local wrapper_path="${output_dir}/code-scout"

    if [[ "${os}" == "darwin" ]]; then
        cat > "${wrapper_path}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
LIB_DIR="${SCRIPT_DIR}/lib"

if [[ -n "${DYLD_LIBRARY_PATH:-}" ]]; then
    export DYLD_LIBRARY_PATH="${LIB_DIR}:${DYLD_LIBRARY_PATH}"
else
    export DYLD_LIBRARY_PATH="${LIB_DIR}"
fi

exec "${SCRIPT_DIR}/code-scout.bin" "$@"
EOF
    else
        cat > "${wrapper_path}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
LIB_DIR="${SCRIPT_DIR}/lib"

if [[ -n "${LD_LIBRARY_PATH:-}" ]]; then
    export LD_LIBRARY_PATH="${LIB_DIR}:${LD_LIBRARY_PATH}"
else
    export LD_LIBRARY_PATH="${LIB_DIR}"
fi

exec "${SCRIPT_DIR}/code-scout.bin" "$@"
EOF
    fi

    chmod +x "${wrapper_path}"
}

build_target() {
    local target="$1"
    local os="${target%_*}"
    local arch="${target#*_}"
    local lib_path="${LIB_DIR}/${target}"
    local bundle_name="code-scout-${target}"
    local output_dir="${DIST_DIR}/${bundle_name}"
    local archive_path="${DIST_DIR}/${bundle_name}.tar.gz"

    if [[ "${target}" == "linux_arm64" ]]; then
        ensure_linux_arm64_libs
    fi

    if [[ ! -d "${lib_path}" ]]; then
        echo "Error: No native libraries at ${lib_path}"
        echo "Run the library download script first:"
        echo "  curl -sSL https://raw.githubusercontent.com/lancedb/lancedb-go/main/scripts/download-artifacts.sh | bash"
        exit 1
    fi

    echo "Building for ${target}..."
    rm -rf "${output_dir}"
    rm -f "${archive_path}"
    mkdir -p "${output_dir}/lib"
    mkdir -p "${CACHE_DIR}/pkg/mod" "${CACHE_DIR}/build"

    export GOOS="${os}"
    export GOARCH="${arch}"
    export CGO_ENABLED=1
    export CGO_CFLAGS="-I${REPO_ROOT}/include"
    export GOMODCACHE="${CACHE_DIR}/pkg/mod"
    export GOCACHE="${CACHE_DIR}/build"

    if [[ "${os}" == "darwin" ]]; then
        export CGO_LDFLAGS="-L${lib_path} -llancedb_go -framework Security -framework CoreFoundation -Wl,-rpath,@executable_path/../lib"
    else
        export CGO_LDFLAGS="-L${lib_path} -llancedb_go -Wl,-rpath,\\\$ORIGIN/../lib"
    fi

    if [[ -f "${lib_path}/liblancedb_go.a" ]]; then
        ranlib "${lib_path}/liblancedb_go.a"
    fi

    go build -o "${output_dir}/code-scout.bin" ./cmd/code-scout
    go build -o "${output_dir}/tei-wrapper" ./cmd/tei-wrapper
    rsync -a "${lib_path}/" "${output_dir}/lib/"
    create_wrapper "${os}" "${output_dir}"

    tar -czf "${archive_path}" -C "${DIST_DIR}" "${bundle_name}"

    echo "✓ Build complete: ${output_dir}"
}

# Main execution
TARGET=$(detect_platform)
echo "Detected platform: ${TARGET}"

if [[ ! -d "${LIB_DIR}" ]]; then
    echo "Missing lib directory at ${LIB_DIR}"
    exit 1
fi

# Kill ALL existing tei-wrapper processes FIRST
echo "Checking for running tei-wrapper processes..."
if pgrep -f "tei-wrapper" > /dev/null 2>&1; then
    echo "Found running tei-wrapper process(es), killing all..."
    pkill -f "tei-wrapper" || true
    sleep 1

    # Double check they're all dead, force kill if needed
    if pgrep -f "tei-wrapper" > /dev/null 2>&1; then
        echo "Warning: tei-wrapper still running, force killing all..."
        pkill -9 -f "tei-wrapper" || true
        sleep 1
    fi

    echo "✓ Killed all existing tei-wrapper processes"
else
    echo "No existing tei-wrapper processes found"
fi

mkdir -p "${DIST_DIR}"

# Build for current platform (provides natural delay for process cleanup)
build_target "${TARGET}"

# Run the newly built tei-wrapper
BUNDLE_NAME="code-scout-${TARGET}"
TEI_WRAPPER="${DIST_DIR}/${BUNDLE_NAME}/tei-wrapper"

if [[ ! -x "${TEI_WRAPPER}" ]]; then
    echo "Error: tei-wrapper not found at ${TEI_WRAPPER}"
    exit 1
fi

LOG_FILE="${REPO_ROOT}/tei-wrapper.log"

echo ""
echo "Starting tei-wrapper from ${TEI_WRAPPER}"
echo "=========================================="

"${TEI_WRAPPER}" > "${LOG_FILE}" 2>&1 &
TEI_PID=$!

echo "✓ Started tei-wrapper (PID: ${TEI_PID})"
echo ""
echo "View logs: tail -f ${LOG_FILE}"
echo "Stop process: kill ${TEI_PID}"
