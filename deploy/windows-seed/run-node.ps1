[CmdletBinding()]
param([string]$ConfigPath = (Join-Path $PSScriptRoot "seed.env"))

$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "seed-config.ps1")

try {
    $config = Read-SeedEnvironment -Path $ConfigPath
    Assert-SeedEnvironment -Values $config
    Set-SeedProcessEnvironment -Values $config

    $executable = Join-Path ([string]$config["ENTROPY_INSTALL_DIR"]) "entropy-cli.exe"
    if (-not (Test-Path -LiteralPath $executable -PathType Leaf)) {
        throw "Entropy CLI is missing at $executable"
    }
    $logDirectory = [string]$config["ENTROPY_LOG_DIR"]
    New-Item -ItemType Directory -Path $logDirectory -Force | Out-Null
    Get-ChildItem -LiteralPath $logDirectory -Filter "node-*.log" -File -ErrorAction SilentlyContinue |
        Where-Object LastWriteTime -lt (Get-Date).AddDays(-30) |
        Remove-Item -Force
    $logPath = Join-Path $logDirectory ("node-{0:yyyyMMdd-HHmmss}.log" -f (Get-Date))

    $arguments = @(
        "node",
        "--data", [string]$config["ENTROPY_DATA_DIR"],
        "--listen", [string]$config["ENTROPY_LISTEN_ADDRESS"],
        "--prune-depth", [string]$config["ENTROPY_PRUNE_DEPTH"],
        "--no-discovery",
        "--trust-loopback-proxy"
    )
    foreach ($manifestURL in ([string]$config["ENTROPY_BOOTSTRAP_MANIFEST_URLS"]).Split(",")) {
        $arguments += @("--bootstrap-manifest", $manifestURL.Trim())
    }
    $peer = [string]$config["ENTROPY_BOOTSTRAP_PEER"]
    if (-not [string]::IsNullOrWhiteSpace($peer)) {
        $arguments += @("--peer", $peer)
    }

    & $executable @arguments *>> $logPath
    exit $LASTEXITCODE
}
catch {
    Write-Error $_
    exit 1
}
