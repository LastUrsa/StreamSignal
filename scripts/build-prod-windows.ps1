param(
    [Parameter(Mandatory = $true)]
    [string]$SourceArchive,

    [string]$BuildRoot = (Join-Path $env:TEMP "StreamSignal-prod-build"),

    [switch]$Launch,

    [string]$LaunchEnvironment = "",

    [switch]$Nsis,

    [switch]$ServiceSmoke
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
    $buildArgs = @("build")
    if ($Nsis) {
        foreach ($nsisPath in @("${env:ProgramFiles(x86)}\NSIS", "${env:ProgramFiles}\NSIS")) {
            if ((Test-Path (Join-Path $nsisPath "makensis.exe")) -and ($env:Path -notlike "*$nsisPath*")) {
                $env:Path = "$nsisPath;$env:Path"
            }
        }
        $buildArgs += @("-platform", "windows/amd64", "-nsis")
    }
    & wails @buildArgs
    $exe = Join-Path $BuildRoot "build\bin\StreamSignal.exe"
    if ($ServiceSmoke) {
        $smokeScript = Join-Path $BuildRoot "scripts\test-service-mode-windows.ps1"
        $smokeEnvironment = "service-smoke"
        if (-not [string]::IsNullOrWhiteSpace($LaunchEnvironment)) {
            $smokeEnvironment = $LaunchEnvironment
        }
        & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $smokeScript -ExePath $exe -EnvironmentName $smokeEnvironment
    }
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
