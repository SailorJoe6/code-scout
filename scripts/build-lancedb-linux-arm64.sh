#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")/.." && pwd)"
LIB_DIR="${REPO_ROOT}/lib/linux_arm64"
CACHE_DIR="${LANCEDB_GO_CACHE_DIR:-${REPO_ROOT}/.cache/lancedb-go}"
LANCEDB_GO_REPO="${LANCEDB_GO_REPO:-https://github.com/lancedb/lancedb-go.git}"
LANCEDB_GO_TAG="${LANCEDB_GO_TAG:-}"
PROTOC_CACHE_DIR="${PROTOC_CACHE_DIR:-${REPO_ROOT}/.cache/protoc}"
PROTOC_VERSION="${PROTOC_VERSION:-}"

require_linux_arm64() {
    local os
    local arch
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"

    if [[ "${os}" != "linux" ]]; then
        echo "This build helper only supports Linux."
        exit 1
    fi

    case "${arch}" in
        aarch64|arm64) ;;
        *)
            echo "Unsupported architecture for this helper: ${arch}"
            exit 1
            ;;
    esac
}

ensure_command() {
    local name="$1"
    if ! command -v "${name}" >/dev/null 2>&1; then
        return 1
    fi
}

ensure_dependencies() {
    local required=(
        gcc
        g++
        make
        cmake
        pkg-config
        git
        curl
    )
    local missing=0

    for cmd in "${required[@]}"; do
        if ! command -v "${cmd}" >/dev/null 2>&1; then
            missing=1
            break
        fi
    done

    if [[ "${missing}" -eq 1 ]]; then
        install_packages
    fi

    ensure_protoc
}

install_packages() {
    local apt_packages=(
        build-essential
        pkg-config
        cmake
        clang
        git
        curl
        protobuf-compiler
    )

    local rpm_packages=(
        gcc
        gcc-c++
        make
        cmake
        pkgconf-pkg-config
        clang
        git
        curl
        protobuf-compiler
    )

    if command -v apt-get >/dev/null 2>&1; then
        if command -v sudo >/dev/null 2>&1; then
            sudo apt-get update
            sudo apt-get install -y "${apt_packages[@]}"
        else
            apt-get update
            apt-get install -y "${apt_packages[@]}"
        fi
        return
    fi

    if command -v dnf >/dev/null 2>&1; then
        if command -v sudo >/dev/null 2>&1; then
            sudo dnf install -y "${rpm_packages[@]}"
        else
            dnf install -y "${rpm_packages[@]}"
        fi
        return
    fi

    if command -v yum >/dev/null 2>&1; then
        if command -v sudo >/dev/null 2>&1; then
            sudo yum install -y "${rpm_packages[@]}"
        else
            yum install -y "${rpm_packages[@]}"
        fi
        return
    fi

    if command -v apk >/dev/null 2>&1; then
        local apk_packages=(
            build-base
            pkgconfig
            cmake
            clang
            git
            curl
            protobuf
        )
        if command -v sudo >/dev/null 2>&1; then
            sudo apk add --no-cache "${apk_packages[@]}"
        else
            apk add --no-cache "${apk_packages[@]}"
        fi
        return
    fi

    echo "No supported package manager found. Install build dependencies manually:"
    echo "  build tools (gcc/clang, make), cmake, pkg-config, git, curl, protobuf-compiler"
    exit 1
}

ensure_rust() {
    if ensure_command cargo && ensure_command rustc; then
        return
    fi

    if ensure_command rustup; then
        rustup toolchain install stable
        rustup default stable
        return
    fi

    if ! ensure_command curl; then
        install_packages
    fi

    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
    # shellcheck source=/dev/null
    source "${HOME}/.cargo/env"
}

ensure_protoc() {
    if ensure_command protoc; then
        return
    fi

    if ! ensure_command curl || ! ensure_command python3; then
        install_packages
    fi

    local version="${PROTOC_VERSION}"
    if [[ -z "${version}" ]]; then
        version="$(curl -sSfL https://api.github.com/repos/protocolbuffers/protobuf/releases/latest | python3 - <<'PY'
import json, sys
data = json.load(sys.stdin)
print(data.get("tag_name", "").strip())
PY
)" || true
        version="${version#v}"
    fi

    if [[ -z "${version}" ]]; then
        version="25.3"
    fi

    local cache_dir="${PROTOC_CACHE_DIR}/${version}"
    local archive="${cache_dir}/protoc-${version}-linux-aarch_64.zip"
    local url="https://github.com/protocolbuffers/protobuf/releases/download/v${version}/protoc-${version}-linux-aarch_64.zip"

    mkdir -p "${cache_dir}"
    curl -sSfL "${url}" -o "${archive}"

    python3 - <<PY
import os, zipfile
archive = "${archive}"
dest = "${cache_dir}"
with zipfile.ZipFile(archive, "r") as zf:
    zf.extractall(dest)
PY

    mkdir -p "${HOME}/.local/bin"
    install -m 755 "${cache_dir}/bin/protoc" "${HOME}/.local/bin/protoc"
    export PATH="${HOME}/.local/bin:${PATH}"
}

resolve_latest_tag() {
    if [[ -n "${LANCEDB_GO_TAG}" ]]; then
        echo "${LANCEDB_GO_TAG}"
        return
    fi

    if ensure_command curl && ensure_command python3; then
        local tag
        tag="$(curl -sSfL https://api.github.com/repos/lancedb/lancedb-go/releases/latest | python3 - <<'PY'
import json, sys
data = json.load(sys.stdin)
print(data.get("tag_name", "").strip())
PY
)"
        if [[ -n "${tag}" ]]; then
            echo "${tag}"
            return
        fi
    fi

    if ensure_command git; then
        git ls-remote --tags "${LANCEDB_GO_REPO}" | awk -F/ '{print $3}' | grep -v '{}' | sort -V | tail -n 1
        return
    fi

    echo "Failed to resolve latest lancedb-go release tag."
    exit 1
}

clone_or_update_repo() {
    local tag="$1"
    if [[ ! -d "${CACHE_DIR}/.git" ]]; then
        rm -rf "${CACHE_DIR}"
        git clone "${LANCEDB_GO_REPO}" "${CACHE_DIR}"
    else
        git -C "${CACHE_DIR}" fetch --tags --prune
    fi

    git -C "${CACHE_DIR}" checkout --force "${tag}"
    git -C "${CACHE_DIR}" clean -fdx
}

run_build() {
    local rust_dir="${CACHE_DIR}/rust"

    if [[ -x "${CACHE_DIR}/scripts/build.sh" ]]; then
        "${CACHE_DIR}/scripts/build.sh"
        return
    fi

    if [[ -x "${CACHE_DIR}/scripts/build-lancedb.sh" ]]; then
        "${CACHE_DIR}/scripts/build-lancedb.sh"
        return
    fi

    if [[ -d "${rust_dir}" ]]; then
        (cd "${rust_dir}" && cargo build --release)
        return
    fi

    echo "Unable to locate build instructions in ${CACHE_DIR}."
    exit 1
}

find_artifacts() {
    local static_lib
    local shared_lib

    static_lib="$(find "${CACHE_DIR}" -type f -name "liblancedb_go.a" | head -n 1 || true)"
    shared_lib="$(find "${CACHE_DIR}" -type f -name "liblancedb_go.so" | head -n 1 || true)"

    if [[ -z "${static_lib}" || -z "${shared_lib}" ]]; then
        echo "Failed to locate built liblancedb_go artifacts in ${CACHE_DIR}."
        exit 1
    fi

    echo "${static_lib}::${shared_lib}"
}

validate_shared_lib() {
    local shared_lib="$1"
    if ! ensure_command readelf; then
        echo "readelf not available; skipping architecture validation for ${shared_lib}"
        return
    fi

    local machine
    machine="$(readelf -h "${shared_lib}" | awk -F: '/Machine:/ {gsub(/^[[:space:]]+/, "", $2); print $2}')"
    if [[ "${machine}" != *"AArch64"* ]]; then
        echo "Unexpected architecture for ${shared_lib}: ${machine}"
        exit 1
    fi
}

copy_artifacts() {
    local static_lib="$1"
    local shared_lib="$2"

    mkdir -p "${LIB_DIR}"
    install -m 644 "${static_lib}" "${LIB_DIR}/liblancedb_go.a"
    install -m 644 "${shared_lib}" "${LIB_DIR}/liblancedb_go.so"

    if ensure_command ranlib; then
        ranlib "${LIB_DIR}/liblancedb_go.a"
    fi
}

main() {
    require_linux_arm64
    ensure_dependencies
    ensure_rust

    local tag
    tag="$(resolve_latest_tag)"
    if [[ -z "${tag}" ]]; then
        echo "Unable to resolve lancedb-go tag."
        exit 1
    fi

    echo "Building lancedb-go native libraries from ${tag}"
    clone_or_update_repo "${tag}"
    run_build

    local artifacts
    artifacts="$(find_artifacts)"
    local static_lib="${artifacts%%::*}"
    local shared_lib="${artifacts##*::}"

    validate_shared_lib "${shared_lib}"
    copy_artifacts "${static_lib}" "${shared_lib}"

    echo "✓ Installed LanceDB native libraries to ${LIB_DIR}"
}

main "$@"
