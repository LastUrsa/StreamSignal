param(
    [Parameter(Mandatory = $true)]
    [string]$ExePath,

    [string]$EnvironmentName = "service-smoke"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $ExePath)) {
    throw "StreamSignal executable not found: $ExePath"
}

$resolvedExePath = (Resolve-Path $ExePath).Path
$startedProcesses = New-Object System.Collections.Generic.List[System.Diagnostics.Process]
$previousEnvironment = $env:STREAMSIGNAL_ENV

function Write-Step {
    param([string]$Message)
    Write-Host "[service-smoke] $Message"
}

function Get-TestProcesses {
    Get-Process -Name "StreamSignal" -ErrorAction SilentlyContinue | Where-Object {
        try {
            $_.Path -eq $resolvedExePath
        } catch {
            $false
        }
    }
}

function Stop-TestProcesses {
    Get-TestProcesses | ForEach-Object {
        try {
            Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
            $_.WaitForExit(5000)
        } catch {
        }
    }
}

function Get-SipAppUrls {
    $urls = @()
    foreach ($port in 47020..47029) {
        $url = "http://127.0.0.1:$port/api/v1/app"
        try {
            $response = Invoke-RestMethod -Method Get -Uri $url -TimeoutSec 1
            if ($response.appId -eq "streamsignal") {
                $urls += $url
            }
        } catch {
        }
    }
    return $urls
}

function Wait-ForSipApp {
    param(
        [string]$ExpectedMode,
        [string[]]$ExcludeUrls = @(),
        [int]$TimeoutSeconds = 20
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        foreach ($port in 47020..47029) {
            $url = "http://127.0.0.1:$port/api/v1/app"
            if ($ExcludeUrls -contains $url) {
                continue
            }
            try {
                $response = Invoke-RestMethod -Method Get -Uri $url -TimeoutSec 1
                if ($response.appId -eq "streamsignal" -and $response.mode -eq $ExpectedMode) {
                    return @{ Url = $url; App = $response }
                }
            } catch {
            }
        }
        Start-Sleep -Milliseconds 250
    }

    throw "Timed out waiting for SIP /api/v1/app mode '$ExpectedMode'"
}

function Wait-ForSipUrlUnavailable {
    param(
        [string]$Url,
        [int]$TimeoutSeconds = 10
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        try {
            Invoke-RestMethod -Method Get -Uri $Url -TimeoutSec 1 | Out-Null
        } catch {
            return
        }
        Start-Sleep -Milliseconds 250
    }

    throw "Timed out waiting for SIP URL to stop responding: $Url"
}

function Wait-ForProcessCount {
    param(
        [int]$ExpectedCount,
        [int]$TimeoutSeconds = 10
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        $processes = @(Get-TestProcesses)
        if ($processes.Count -eq $ExpectedCount) {
            return $processes
        }
        Start-Sleep -Milliseconds 250
    }

    $actual = @(Get-TestProcesses).Count
    throw "Expected $ExpectedCount StreamSignal test process(es), found $actual"
}

function Wait-ForMainWindowHandle {
    param(
        [System.Diagnostics.Process]$Process,
        [bool]$ShouldExist,
        [int]$TimeoutSeconds = 10
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        $Process.Refresh()
        $hasWindow = $Process.MainWindowHandle -ne [IntPtr]::Zero
        if ($hasWindow -eq $ShouldExist) {
            return
        }
        Start-Sleep -Milliseconds 250
    }

    $Process.Refresh()
    throw "Window handle expectation failed for process $($Process.Id): expected exists=$ShouldExist, actual handle=$($Process.MainWindowHandle)"
}

function Start-TestStreamSignal {
    param([string[]]$Arguments = @())

    $startArgs = @{
        FilePath = $resolvedExePath
        PassThru = $true
    }
    if ($Arguments.Count -gt 0) {
        $startArgs.ArgumentList = $Arguments
    }

    $process = Start-Process @startArgs
    $startedProcesses.Add($process) | Out-Null
    return $process
}

try {
    $env:STREAMSIGNAL_ENV = $EnvironmentName
    Stop-TestProcesses

    Write-Step "Testing --service hidden startup, SIP service mode, --show activation, and duplicate prevention"
    $baselineUrls = @(Get-SipAppUrls)
    $serviceProcess = Start-TestStreamSignal -Arguments @("--service")
    Wait-ForProcessCount -ExpectedCount 1 | Out-Null
    $serviceEndpoint = Wait-ForSipApp -ExpectedMode "service" -ExcludeUrls $baselineUrls
    Wait-ForMainWindowHandle -Process $serviceProcess -ShouldExist $false -TimeoutSeconds 5

    Start-TestStreamSignal -Arguments @("--show") | Out-Null
    $processes = @(Wait-ForProcessCount -ExpectedCount 1)
    $serviceProcess = $processes[0]
    Wait-ForMainWindowHandle -Process $serviceProcess -ShouldExist $true

    Stop-TestProcesses
    Wait-ForSipUrlUnavailable -Url $serviceEndpoint.Url

    Write-Step "Testing standalone startup and duplicate --service suppression"
    $baselineUrls = @(Get-SipAppUrls)
    $standaloneProcess = Start-TestStreamSignal
    Wait-ForProcessCount -ExpectedCount 1 | Out-Null
    Wait-ForSipApp -ExpectedMode "standalone" -ExcludeUrls $baselineUrls | Out-Null
    Wait-ForMainWindowHandle -Process $standaloneProcess -ShouldExist $true

    Start-TestStreamSignal -Arguments @("--service") | Out-Null
    Wait-ForProcessCount -ExpectedCount 1 | Out-Null

    Write-Step "Service mode smoke test passed"
} finally {
    Stop-TestProcesses
    if ($null -eq $previousEnvironment) {
        Remove-Item Env:\STREAMSIGNAL_ENV -ErrorAction SilentlyContinue
    } else {
        $env:STREAMSIGNAL_ENV = $previousEnvironment
    }
}
