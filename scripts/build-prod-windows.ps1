param(
    [Parameter(Mandatory = $true)]
    [string]$SourceArchive,

    [string]$BuildRoot = (Join-Path $env:TEMP "StreamSignal-prod-build"),

    [switch]$Launch,

    [string]$LaunchEnvironment = ""
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
        $previousEnvironment = $env:STREAMSIGNAL_ENV
        try {
            if (-not [string]::IsNullOrWhiteSpace($LaunchEnvironment)) {
                $env:STREAMSIGNAL_ENV = $LaunchEnvironment
            }
            Start-Process -FilePath $exe
        } finally {
            if ($null -eq $previousEnvironment) {
                Remove-Item Env:\STREAMSIGNAL_ENV -ErrorAction SilentlyContinue
            } else {
                $env:STREAMSIGNAL_ENV = $previousEnvironment
            }
        }
    }
    Write-Host "Built $exe"
} finally {
    Pop-Location
}
