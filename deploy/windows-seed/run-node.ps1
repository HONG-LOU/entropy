[CmdletBinding()]
param([string]$ConfigPath = (Join-Path $PSScriptRoot "seed.env"))

$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "seed-config.ps1")

try {
    $config = Read-SeedEnvironment -Path $ConfigPath
    Assert-SeedEnvironment -Values $config
    Set-SeedProcessEnvironment -Values $config

    $executable = Join-Path ([string]$config["ENTCOIN_INSTALL_DIR"]) "entcoin-cli.exe"
    if (-not (Test-Path -LiteralPath $executable -PathType Leaf)) {
        throw "Entcoin CLI is missing at $executable"
    }
    $logDirectory = [string]$config["ENTCOIN_LOG_DIR"]
    New-Item -ItemType Directory -Path $logDirectory -Force | Out-Null
    Get-ChildItem -LiteralPath $logDirectory -Filter "node-*.log" -File -ErrorAction SilentlyContinue |
        Where-Object LastWriteTime -lt (Get-Date).AddDays(-30) |
        Remove-Item -Force
    $logPath = Join-Path $logDirectory ("node-{0:yyyyMMdd-HHmmss}.log" -f (Get-Date))

    $arguments = @(
        "node",
        "--seed-mode",
        "--data", [string]$config["ENTCOIN_DATA_DIR"],
        "--listen", [string]$config["ENTCOIN_LISTEN_ADDRESS"],
        "--prune-depth", [string]$config["ENTCOIN_PRUNE_DEPTH"],
        "--no-discovery",
        "--trust-loopback-proxy"
    )
    foreach ($manifestURL in ([string]$config["ENTCOIN_BOOTSTRAP_MANIFEST_URLS"]).Split(",")) {
        $arguments += @("--bootstrap-manifest", $manifestURL.Trim())
    }
    $peer = [string]$config["ENTCOIN_BOOTSTRAP_PEER"]
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
