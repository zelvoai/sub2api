param(
    [ValidateSet("up", "build", "rebuild", "down", "restart", "logs", "ps")]
    [string]$Action = "up"
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $ScriptDir
$DeployDir = Join-Path $RepoRoot "deploy"
$EnvFile = Join-Path $DeployDir ".env"
$EnvExample = Join-Path $DeployDir ".env.example"
$ComposeFile = Join-Path $DeployDir "docker-compose.dev.yml"

function Assert-Command {
    param([string]$Name)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Missing command: $Name"
    }
}

function New-HexSecret {
    $bytes = New-Object byte[] 32
    [System.Security.Cryptography.RandomNumberGenerator]::Fill($bytes)
    return -join ($bytes | ForEach-Object { $_.ToString("x2") })
}

function Set-EnvValue {
    param(
        [string]$Path,
        [string]$Key,
        [string]$Value
    )

    $line = "$Key=$Value"
    if (-not (Test-Path $Path)) {
        Set-Content -Path $Path -Value $line -Encoding utf8
        return
    }

    $content = Get-Content -Path $Path
    $updated = $false
    $next = foreach ($item in $content) {
        if ($item -like "$Key=*") {
            $updated = $true
            $line
        } else {
            $item
        }
    }
    if (-not $updated) {
        $next += $line
    }
    Set-Content -Path $Path -Value $next -Encoding utf8
}

function Ensure-Env {
    if (Test-Path $EnvFile) {
        return
    }
    if (-not (Test-Path $EnvExample)) {
        throw "Missing env example: $EnvExample"
    }

    Copy-Item $EnvExample $EnvFile
    Set-EnvValue $EnvFile "POSTGRES_PASSWORD" (New-HexSecret)
    Set-EnvValue $EnvFile "JWT_SECRET" (New-HexSecret)
    Set-EnvValue $EnvFile "TOTP_ENCRYPTION_KEY" (New-HexSecret)
    Set-EnvValue $EnvFile "SERVER_PORT" "8080"
    Write-Host "Initialized $EnvFile"
}

function Ensure-Dirs {
    @(
        (Join-Path $DeployDir "data"),
        (Join-Path $DeployDir "postgres_data"),
        (Join-Path $DeployDir "redis_data")
    ) | ForEach-Object {
        New-Item -ItemType Directory -Force -Path $_ | Out-Null
    }
}

function Invoke-Compose {
    param([string[]]$ComposeArgs)
    docker compose --env-file $EnvFile -f $ComposeFile @ComposeArgs
}

function Get-EnvValue {
    param(
        [string]$Path,
        [string]$Key,
        [string]$DefaultValue
    )

    if (-not (Test-Path $Path)) {
        return $DefaultValue
    }
    $line = Get-Content -Path $Path | Where-Object { $_ -like "$Key=*" } | Select-Object -Last 1
    if (-not $line) {
        return $DefaultValue
    }
    $value = $line.Substring($Key.Length + 1).Trim()
    if ($value) {
        return $value
    }
    return $DefaultValue
}

Assert-Command "docker"
Ensure-Env
Ensure-Dirs

switch ($Action) {
    "up" {
        Invoke-Compose @("up", "-d")
    }
    "build" {
        Invoke-Compose @("up", "--build", "-d")
    }
    "rebuild" {
        Invoke-Compose @("down")
        Invoke-Compose @("up", "--build", "-d")
    }
    "down" {
        Invoke-Compose @("down")
    }
    "restart" {
        Invoke-Compose @("down")
        Invoke-Compose @("up", "-d")
    }
    "logs" {
        Invoke-Compose @("logs", "-f", "sub2api")
    }
    "ps" {
        Invoke-Compose @("ps")
    }
}

if ($Action -in @("up", "build", "rebuild", "restart", "ps")) {
    $serverPort = Get-EnvValue $EnvFile "SERVER_PORT" "8080"
    Write-Host ""
    Write-Host "Web: http://localhost:$serverPort"
    Write-Host "Logs: .\scripts\dev-docker.ps1 logs"
    Write-Host "Postgres data: $DeployDir\postgres_data"
}
