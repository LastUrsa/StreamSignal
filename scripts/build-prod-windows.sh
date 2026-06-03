#!/usr/bin/env bash
set -euo pipefail

launch=false
if [[ "${1:-}" == "--launch" ]]; then
    launch=true
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
archive="/tmp/streamsignal-prod-build.tar"

cd "$repo_root"
tar -cf "$archive" \
    --exclude=.git \
    --exclude=build/bin \
    --exclude=coverage \
    --exclude=frontend/coverage \
    --exclude=frontend/dist \
    --exclude=frontend/node_modules \
    --exclude=node_modules \
    .

archive_windows="$(wslpath -w "$archive")"
script_windows="$(wslpath -w "$repo_root/scripts/build-prod-windows.ps1")"

args=(-NoProfile -ExecutionPolicy Bypass -File "$script_windows" -SourceArchive "$archive_windows")
if [[ "$launch" == true ]]; then
    args+=(-Launch)
fi

powershell.exe "${args[@]}"
