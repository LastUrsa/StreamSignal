#!/usr/bin/env bash
set -euo pipefail

launch=false
launch_env=""
nsis=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        --launch)
            launch=true
            shift
            ;;
        --nsis)
            nsis=true
            shift
            ;;
        --env)
            if [[ -z "${2:-}" ]]; then
                echo "--env requires a value" >&2
                exit 2
            fi
            launch_env="$2"
            shift 2
            ;;
        *)
            echo "unknown argument: $1" >&2
            exit 2
            ;;
    esac
done

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
if [[ -n "$launch_env" ]]; then
    args+=(-LaunchEnvironment "$launch_env")
fi
if [[ "$nsis" == true ]]; then
    args+=(-Nsis)
fi

powershell.exe "${args[@]}"
