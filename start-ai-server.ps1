$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$serverRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$mainEntry = Join-Path $serverRoot 'cmd\ai-server\main.go'
$logsDir = Join-Path $serverRoot 'logs'
$stdoutLog = Join-Path $logsDir 'ai-server-stdout.log'
$stderrLog = Join-Path $logsDir 'ai-server-stderr.log'
$pidFile = Join-Path $serverRoot '.ai-server.pid'
$httpHealthUrl = 'http://127.0.0.1:18080/healthz'
$wsHealthUrl = 'http://127.0.0.1:18443/healthz'
$listenPorts = @(18080, 18443)

function Test-HealthEndpoint {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Url
  )

  $handler = $null
  $client = $null
  try {
    $handler = [System.Net.Http.HttpClientHandler]::new()
    $handler.UseProxy = $false
    $client = [System.Net.Http.HttpClient]::new($handler)
    $client.Timeout = [TimeSpan]::FromSeconds(2)
    $response = $client.GetAsync($Url).GetAwaiter().GetResult()
    return [int]$response.StatusCode -eq 200
  } catch {
    return $false
  } finally {
    if ($client -ne $null) {
      $client.Dispose()
    }
    if ($handler -ne $null) {
      $handler.Dispose()
    }
  }
}

function Get-ListeningProcessIds {
  param(
    [Parameter(Mandatory = $true)]
    [int[]]$Ports
  )

  $connections = Get-NetTCPConnection -State Listen -ErrorAction SilentlyContinue |
    Where-Object { $_.LocalPort -in $Ports }
  if (-not $connections) {
    return @()
  }

  return @($connections | Select-Object -ExpandProperty OwningProcess -Unique)
}

if (-not (Test-Path $mainEntry)) {
  throw "Server entry not found: $mainEntry"
}

$goCommand = Get-Command go -ErrorAction Stop

if (-not (Test-Path $logsDir)) {
  New-Item -ItemType Directory -Path $logsDir | Out-Null
}

if (Test-Path $pidFile) {
  $existingPidText = (Get-Content $pidFile -ErrorAction SilentlyContinue | Select-Object -First 1).Trim()
  if ($existingPidText) {
    $existingPid = 0
    if ([int]::TryParse($existingPidText, [ref]$existingPid)) {
      $existingProcess = Get-Process -Id $existingPid -ErrorAction SilentlyContinue
      if ($existingProcess) {
        if ((Test-HealthEndpoint -Url $httpHealthUrl) -and (Test-HealthEndpoint -Url $wsHealthUrl)) {
          Write-Host "ai-server is already running. PID: $existingPid"
          Write-Host "HTTP health: $httpHealthUrl"
          Write-Host "WS health: $wsHealthUrl"
          Write-Host "Log file: $stdoutLog"
          exit 0
        }
      } else {
        Remove-Item $pidFile -ErrorAction SilentlyContinue
      }
    }
  }
}

$listeningPids = @(Get-ListeningProcessIds -Ports $listenPorts)
if ($listeningPids.Count -gt 0) {
  throw "Ports $($listenPorts -join ', ') are already in use by PID: $($listeningPids -join ', ')."
}

$process = Start-Process `
  -FilePath $goCommand.Source `
  -ArgumentList @('run', './cmd/ai-server') `
  -WorkingDirectory $serverRoot `
  -RedirectStandardOutput $stdoutLog `
  -RedirectStandardError $stderrLog `
  -WindowStyle Hidden `
  -PassThru

Set-Content -Path $pidFile -Value $process.Id -Encoding ascii

$deadline = (Get-Date).AddSeconds(20)
$started = $false

while ((Get-Date) -lt $deadline) {
  Start-Sleep -Milliseconds 500

  $runningProcess = Get-Process -Id $process.Id -ErrorAction SilentlyContinue
  if (-not $runningProcess) {
    break
  }

  if ((Test-HealthEndpoint -Url $httpHealthUrl) -and (Test-HealthEndpoint -Url $wsHealthUrl)) {
    $started = $true
    break
  }
}

if (-not $started) {
  Remove-Item $pidFile -ErrorAction SilentlyContinue
  $stdoutTail = if (Test-Path $stdoutLog) { Get-Content $stdoutLog -Tail 20 } else { @() }
  $stderrTail = if (Test-Path $stderrLog) { Get-Content $stderrLog -Tail 20 } else { @() }

  if (Get-Process -Id $process.Id -ErrorAction SilentlyContinue) {
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  }

  Write-Host 'ai-server failed to start. Recent logs:'
  if (@($stdoutTail).Count -gt 0) {
    Write-Host '--- stdout ---'
    $stdoutTail | ForEach-Object { Write-Host $_ }
  }
  if (@($stderrTail).Count -gt 0) {
    Write-Host '--- stderr ---'
    $stderrTail | ForEach-Object { Write-Host $_ }
  }

  exit 1
}

Write-Host "ai-server started. PID: $($process.Id)"
Write-Host "HTTP health: $httpHealthUrl"
Write-Host "WS health: $wsHealthUrl"
Write-Host "stdout log: $stdoutLog"
Write-Host "stderr log: $stderrLog"
Write-Host "pid file: $pidFile"
