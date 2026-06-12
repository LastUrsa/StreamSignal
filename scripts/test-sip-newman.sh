#!/usr/bin/env bash
set -euo pipefail

launch=true
launch_env="dev"
base_url=""
collection="docs/streamsignal-sip-newman-smoke.postman_collection.json"
timeout_seconds=90
newman_package="newman@6.2.1"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --no-launch)
            launch=false
            shift
            ;;
        --base-url)
            if [[ -z "${2:-}" ]]; then
                echo "--base-url requires a value" >&2
                exit 2
            fi
            base_url="$2"
            shift 2
            ;;
        --collection)
            if [[ -z "${2:-}" ]]; then
                echo "--collection requires a value" >&2
                exit 2
            fi
            collection="$2"
            shift 2
            ;;
        --env)
            if [[ -z "${2:-}" ]]; then
                echo "--env requires a value" >&2
                exit 2
            fi
            launch_env="$2"
            shift 2
            ;;
        --timeout)
            if [[ -z "${2:-}" ]]; then
                echo "--timeout requires a value" >&2
                exit 2
            fi
            timeout_seconds="$2"
            shift 2
            ;;
        *)
            echo "unknown argument: $1" >&2
            exit 2
            ;;
    esac
done

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if [[ ! "$launch_env" =~ ^[A-Za-z0-9_-]+$ ]]; then
    echo "--env must contain only letters, numbers, underscores, or hyphens" >&2
    exit 2
fi

if [[ "$launch" == true ]]; then
    ./scripts/build-prod-windows.sh --env "$launch_env"
    exe_windows="C:\\Users\\Don\\AppData\\Local\\Temp\\StreamSignal-prod-build\\build\\bin\\StreamSignal.exe"
    powershell.exe -NoProfile -Command "\$ErrorActionPreference = 'Stop'; \$env:STREAMSIGNAL_ENV = '${launch_env}'; Start-Process -FilePath '${exe_windows}' -ArgumentList '--service'"
fi

run_newman_on_windows=false
if [[ -z "$base_url" ]]; then
    deadline=$((SECONDS + timeout_seconds))
    while [[ $SECONDS -lt $deadline ]]; do
        for port in {47020..47029}; do
            candidate="http://127.0.0.1:${port}"
            if curl --silent --fail --max-time 1 "${candidate}/api/v1/app" >/dev/null; then
                base_url="$candidate"
                break 2
            fi
        done
        discovered="$(
            powershell.exe -NoProfile -Command '$ErrorActionPreference = "SilentlyContinue"; foreach ($p in 47020..47029) { try { Invoke-RestMethod -Uri "http://127.0.0.1:$p/api/v1/app" -TimeoutSec 1 | Out-Null; Write-Output "http://127.0.0.1:$p"; exit 0 } catch {} }; exit 1' 2>/dev/null |
                tr -d '\r' |
                head -n 1
        )"
        if [[ -n "$discovered" ]]; then
            base_url="$discovered"
            run_newman_on_windows=true
            break
        fi
        sleep 1
    done
fi

if [[ -z "$base_url" ]]; then
    echo "Timed out waiting for StreamSignal SIP on 127.0.0.1:47020-47029" >&2
    exit 1
fi

echo "Running SIP Newman smoke checks against ${base_url}"
if [[ "$run_newman_on_windows" == true ]]; then
    collection_windows="$(wslpath -w "$collection")"
    powershell.exe -NoProfile -Command "\$ErrorActionPreference = 'Stop'; Set-Location \$env:TEMP; & npx.cmd --yes '${newman_package}' run '${collection_windows}' --env-var 'baseUrl=${base_url}'"
elif command -v newman >/dev/null 2>&1; then
    newman_cmd=(newman)
elif command -v npx >/dev/null 2>&1; then
    newman_cmd=(npx --yes "$newman_package")
else
    echo "Newman is not installed, and npx is unavailable. Install with: npm install -g newman" >&2
    exit 1
fi
if [[ "$run_newman_on_windows" != true ]]; then
    "${newman_cmd[@]}" run "$collection" --env-var "baseUrl=${base_url}"
fi
