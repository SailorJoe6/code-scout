#!/bin/bash
set -euo pipefail

REPO_ROOT="$(pwd)"
LIB_DIR="${REPO_ROOT}/lib"
DIST_DIR="${REPO_ROOT}/dist"
CACHE_DIR="${REPO_ROOT}/.cache/go"
SCRIPTS_DIR="${REPO_ROOT}/scripts"
LANCEDB_LINUX_ARM64_SCRIPT="${SCRIPTS_DIR}/build-lancedb-linux-arm64.sh"

detect_host_target() {
    local os
    local arch
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"

    case "${arch}" in
        x86_64)
            arch="amd64"
            ;;
        aarch64|arm64)
            arch="arm64"
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

if [[ ! -d "${LIB_DIR}" ]]; then
    echo "Missing lib directory at ${LIB_DIR}"
    exit 1
fi

mkdir -p "${DIST_DIR}"

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
        echo "Skipping ${target}: no native libraries at ${lib_path}"
        return
    fi

    echo "Building for ${target}..."
    if ! (
        set -euo pipefail
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
    ); then
        echo "✗ Build failed for ${target}"
        return 1
    fi

    echo "✓ Output directory: ${output_dir}"
    echo "  Archive: ${archive_path}"
}

failures=()
targets=()

if [[ -n "${TARGETS:-}" ]]; then
    for target in ${TARGETS}; do
        targets+=("${target}")
    done
else
    for dir in "${LIB_DIR}"/*; do
        [[ -d "${dir}" ]] || continue
        targets+=("$(basename "${dir}")")
    done

    host_target="$(detect_host_target)"
    if [[ "${host_target}" == "linux_arm64" && ! -d "${LIB_DIR}/${host_target}" ]]; then
        targets+=("${host_target}")
    fi
fi

for target in "${targets[@]}"; do
    if ! build_target "${target}"; then
        failures+=("${target}")
    fi
done

if (( ${#failures[@]} )); then
    echo "Builds completed with failures: ${failures[*]}"
    exit 1
fi

echo "All builds complete. Archives are in ${DIST_DIR}/"
