#!/bin/bash
set -euo pipefail

REPO_ROOT="$(pwd)"
LIB_DIR="${REPO_ROOT}/lib"
DIST_DIR="${REPO_ROOT}/dist"

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

    if [[ ! -d "${lib_path}" ]]; then
        echo "Skipping ${target}: no native libraries at ${lib_path}"
        return
    fi

    echo "Building for ${target}..."
    rm -rf "${output_dir}"
    rm -f "${archive_path}"
    mkdir -p "${output_dir}/lib"

    export GOOS="${os}"
    export GOARCH="${arch}"
    export CGO_ENABLED=1
    export CGO_CFLAGS="-I${REPO_ROOT}/include"

    if [[ "${os}" == "darwin" ]]; then
        export CGO_LDFLAGS="-L${lib_path} -llancedb_go -framework Security -framework CoreFoundation -Wl,-rpath,@executable_path/../lib"
    else
        export CGO_LDFLAGS="-L${lib_path} -llancedb_go -Wl,-rpath,\\\$ORIGIN/../lib"
    fi

    go build -o "${output_dir}/code-scout.bin" ./cmd/code-scout
    rsync -a "${lib_path}/" "${output_dir}/lib/"
    create_wrapper "${os}" "${output_dir}"

    tar -czf "${archive_path}" -C "${DIST_DIR}" "${bundle_name}"

    echo "✓ Output directory: ${output_dir}"
    echo "  Archive: ${archive_path}"
}

if [[ -n "${TARGETS:-}" ]]; then
    for target in ${TARGETS}; do
        build_target "${target}"
    done
else
    for dir in "${LIB_DIR}"/*; do
        [[ -d "${dir}" ]] || continue
        target="$(basename "${dir}")"
        build_target "${target}"
    done
fi

echo "All builds complete. Archives are in ${DIST_DIR}/"
