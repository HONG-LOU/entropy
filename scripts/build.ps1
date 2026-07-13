$ErrorActionPreference = "Stop"

$project = Split-Path -Parent $PSScriptRoot
Push-Location $project
try {
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "Go tests failed with exit code $LASTEXITCODE" }
    Push-Location (Join-Path $project "frontend")
    try {
        npm install
        if ($LASTEXITCODE -ne 0) { throw "Frontend dependency install failed with exit code $LASTEXITCODE" }
    }
    finally {
        Pop-Location
    }
    wails build -clean -platform windows/amd64
    if ($LASTEXITCODE -ne 0) { throw "Wails build failed with exit code $LASTEXITCODE" }
    $binary = Join-Path $project "build\bin\Entropy.exe"
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $binary).Hash.ToLowerInvariant()
    Set-Content -LiteralPath (Join-Path $project "build\bin\SHA256SUMS.txt") -Value "$hash  Entropy.exe" -Encoding ascii
    Write-Host "Built: $project\build\bin\Entropy.exe"
}
finally {
    Pop-Location
}
