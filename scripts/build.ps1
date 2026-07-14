$ErrorActionPreference = "Stop"

$project = Split-Path -Parent $PSScriptRoot
$nsis = Join-Path ${env:ProgramFiles(x86)} "NSIS\makensis.exe"
if (-not (Test-Path -LiteralPath $nsis)) {
    throw "NSIS 3.x is required at $nsis"
}
$env:PATH = "$(Split-Path -Parent $nsis);$env:PATH"
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go 1.26.5 is required"
}
$goVersion = (& go version 2>&1 | Out-String)
if ($LASTEXITCODE -ne 0 -or $goVersion -notmatch 'go1\.26\.5') {
    throw "Go 1.26.5 is required; got: $($goVersion.Trim())"
}
if (-not (Get-Command wails -ErrorAction SilentlyContinue)) {
    throw "Wails v2.13.0 is required: go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0"
}
$wailsVersion = (& wails version 2>&1 | Out-String)
if ($LASTEXITCODE -ne 0 -or $wailsVersion -notmatch 'v2\.13\.0') {
    throw "Wails v2.13.0 is required; got: $($wailsVersion.Trim())"
}
Push-Location $project
try {
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "Go tests failed with exit code $LASTEXITCODE" }
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "Go vet failed with exit code $LASTEXITCODE" }
    wails build -clean -trimpath -platform windows/amd64 -nsis -installscope user -webview2 download
    if ($LASTEXITCODE -ne 0) { throw "Wails build failed with exit code $LASTEXITCODE" }
    $bin = Join-Path $project "build\bin"
    $portable = Join-Path $bin "Entropy.exe"
    $installer = Join-Path $bin "entropy-amd64-installer.exe"
    $cli = Join-Path $bin "entropy-cli.exe"
    go build -trimpath -o $cli .\cmd\entropy
    if ($LASTEXITCODE -ne 0) { throw "CLI build failed with exit code $LASTEXITCODE" }

    $seedStage = Join-Path $project "build\seed-package"
    $seedPackage = Join-Path $bin "entropy-windows-seed-deploy.zip"
    if (Test-Path -LiteralPath $seedStage) {
        Remove-Item -LiteralPath $seedStage -Recurse -Force
    }
    New-Item -ItemType Directory -Path $seedStage | Out-Null
    try {
        Copy-Item -Path (Join-Path $project "deploy\windows-seed\*") -Destination $seedStage -Recurse
        Copy-Item -LiteralPath $cli -Destination (Join-Path $seedStage "entropy-cli.exe")
        Copy-Item -LiteralPath (Join-Path $project "docs\public-seed.md") -Destination (Join-Path $seedStage "PUBLIC-SEED.md")
        Compress-Archive -Path (Join-Path $seedStage "*") -DestinationPath $seedPackage -CompressionLevel Optimal -Force
    }
    finally {
        Remove-Item -LiteralPath $seedStage -Recurse -Force -ErrorAction SilentlyContinue
    }

    foreach ($expected in @($portable, $installer, $cli, $seedPackage)) {
        if (-not (Test-Path -LiteralPath $expected -PathType Leaf)) {
            throw "Expected release artifact was not produced: $expected"
        }
    }
    $artifacts = Get-Item -LiteralPath $portable, $installer, $cli, $seedPackage | Sort-Object Name
    $checksums = foreach ($artifact in $artifacts) {
        $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $artifact.FullName).Hash.ToLowerInvariant()
        "$hash  $($artifact.Name)"
    }
    Set-Content -LiteralPath (Join-Path $bin "SHA256SUMS.txt") -Value $checksums -Encoding ascii
    $artifacts | ForEach-Object { Write-Host "Built: $($_.FullName)" }
}
finally {
    Pop-Location
}
