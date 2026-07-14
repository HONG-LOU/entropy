[CmdletBinding()]
param([string]$ConfigPath = (Join-Path $PSScriptRoot "seed.env"))

$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "seed-config.ps1")

try {
    $config = Read-SeedEnvironment -Path $ConfigPath
    Assert-SeedEnvironment -Values $config
    Set-SeedProcessEnvironment -Values $config

    $installDirectory = [string]$config["ENTROPY_INSTALL_DIR"]
    $executable = Join-Path $installDirectory "caddy.exe"
    $caddyfile = Join-Path $installDirectory "Caddyfile"
    if (-not (Test-Path -LiteralPath $executable -PathType Leaf)) {
        throw "Caddy is missing at $executable"
    }
    $logDirectory = [string]$config["ENTROPY_LOG_DIR"]
    New-Item -ItemType Directory -Path $logDirectory -Force | Out-Null
    Get-ChildItem -LiteralPath $logDirectory -Filter "caddy-console-*.log" -File -ErrorAction SilentlyContinue |
        Where-Object LastWriteTime -lt (Get-Date).AddDays(-30) |
        Remove-Item -Force
    $logPath = Join-Path $logDirectory ("caddy-console-{0:yyyyMMdd-HHmmss}.log" -f (Get-Date))

    & $executable run --config $caddyfile --adapter caddyfile *>> $logPath
    exit $LASTEXITCODE
}
catch {
    Write-Error $_
    exit 1
}
