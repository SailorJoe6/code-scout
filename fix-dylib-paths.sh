#!/bin/bash
# Fix hardcoded install_name paths in LanceDB dylib files
# This is needed after downloading native libraries from the LanceDB CI server

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="${SCRIPT_DIR}/lib"

echo "Fixing dylib install_name paths..."

for platform_dir in "${LIB_DIR}"/darwin_*; do
    if [[ ! -d "${platform_dir}" ]]; then
        continue
    fi

    dylib="${platform_dir}/liblancedb_go.dylib"
    if [[ ! -f "${dylib}" ]]; then
        echo "⚠ Skipping ${platform_dir}: dylib not found"
        continue
    fi

    platform=$(basename "${platform_dir}")
    echo "Fixing ${platform}..."
    install_name_tool -id @rpath/liblancedb_go.dylib "${dylib}"
    echo "✓ Fixed ${platform}/liblancedb_go.dylib"
done

echo ""
echo "All macOS dylibs fixed!"
echo "Note: Linux .so files don't need fixing (they use RPATH correctly)"
