param(
    [Parameter(Mandatory = $true)]
    [string]$SourceArchive,

    [string]$BuildRoot = (Join-Path $env:TEMP "StreamSignal-prod-build"),

    [switch]$Launch
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $SourceArchive)) {
    throw "Source archive not found: $SourceArchive"
}

if (Test-Path $BuildRoot) {
    Remove-Item $BuildRoot -Recurse -Force
}

New-Item -ItemType Directory -Path $BuildRoot | Out-Null
tar -xf $SourceArchive -C $BuildRoot

Push-Location $BuildRoot
try {
    wails build
    $exe = Join-Path $BuildRoot "build\bin\StreamSignal.exe"
    if ($Launch) {
        Start-Process -FilePath $exe
    }
    Write-Host "Built $exe"
} finally {
    Pop-Location
}
