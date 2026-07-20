Set-StrictMode -Version Latest

$script:SeedEnvironmentNames = @(
    "ENTCOIN_SEED_DOMAIN",
    "ENTCOIN_ACME_EMAIL",
    "ENTCOIN_INSTALL_DIR",
    "ENTCOIN_DATA_DIR",
    "ENTCOIN_CADDY_DATA_DIR",
    "ENTCOIN_LOG_DIR",
    "ENTCOIN_LISTEN_ADDRESS",
    "ENTCOIN_BOOTSTRAP_PEER",
    "ENTCOIN_BOOTSTRAP_MANIFEST_URLS",
    "ENTCOIN_PRUNE_DEPTH",
    "ENTCOIN_DISABLE_DISCOVERY",
    "ENTCOIN_TRUST_LOOPBACK_PROXY"
)

function Read-SeedEnvironment {
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$Path)

    $resolved = (Resolve-Path -LiteralPath $Path -ErrorAction Stop).Path
    $values = [ordered]@{}
    $lineNumber = 0
    foreach ($line in [System.IO.File]::ReadAllLines($resolved)) {
        $lineNumber++
        $trimmed = $line.Trim()
        if ($trimmed.Length -eq 0 -or $trimmed.StartsWith("#")) {
            continue
        }
        $separator = $trimmed.IndexOf("=")
        if ($separator -le 0) {
            throw "Invalid environment entry at ${resolved}:$lineNumber"
        }
        $name = $trimmed.Substring(0, $separator).Trim()
        if ($script:SeedEnvironmentNames -notcontains $name) {
            throw "Unknown seed environment variable '$name' at ${resolved}:$lineNumber"
        }
        if ($values.Contains($name)) {
            throw "Duplicate seed environment variable '$name' at ${resolved}:$lineNumber"
        }
        $value = $trimmed.Substring($separator + 1).Trim()
        if ($value.Length -ge 2) {
            $first = $value[0]
            $last = $value[$value.Length - 1]
            if (($first -eq '"' -and $last -eq '"') -or ($first -eq "'" -and $last -eq "'")) {
                $value = $value.Substring(1, $value.Length - 2)
            }
        }
        $values[$name] = $value
    }
    return $values
}

function Test-SeedHttpsBaseUrl {
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$Value)

    $uri = $null
    if (-not [System.Uri]::TryCreate($Value, [System.UriKind]::Absolute, [ref]$uri)) {
        return $false
    }
    return $uri.Scheme -eq "https" -and
        [string]::IsNullOrEmpty($uri.UserInfo) -and
        $uri.AbsolutePath -eq "/" -and
        [string]::IsNullOrEmpty($uri.Query) -and
        [string]::IsNullOrEmpty($uri.Fragment)
}

function Test-SeedHttpsUrl {
    [CmdletBinding()]
    param([Parameter(Mandatory)][string]$Value)

    $uri = $null
    if (-not [System.Uri]::TryCreate($Value, [System.UriKind]::Absolute, [ref]$uri)) {
        return $false
    }
    return $uri.Scheme -eq "https" -and
        [string]::IsNullOrEmpty($uri.UserInfo) -and
        [string]::IsNullOrEmpty($uri.Query) -and
        [string]::IsNullOrEmpty($uri.Fragment)
}

function Test-SeedPathWithin {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$Child,
        [Parameter(Mandatory)][string]$Parent
    )

    $separator = [System.IO.Path]::DirectorySeparatorChar
    $childPath = [System.IO.Path]::GetFullPath($Child).TrimEnd($separator)
    $parentPath = [System.IO.Path]::GetFullPath($Parent).TrimEnd($separator)
    return $childPath.Equals($parentPath, [System.StringComparison]::OrdinalIgnoreCase) -or
        $childPath.StartsWith($parentPath + $separator, [System.StringComparison]::OrdinalIgnoreCase)
}

function Assert-SeedEnvironment {
    [CmdletBinding()]
    param([Parameter(Mandatory)][System.Collections.IDictionary]$Values)

    foreach ($name in $script:SeedEnvironmentNames) {
        if (-not $Values.Contains($name)) {
            throw "Missing required seed environment variable '$name'"
        }
    }

    $domain = [string]$Values["ENTCOIN_SEED_DOMAIN"]
    if ([System.Uri]::CheckHostName($domain) -ne [System.UriHostNameType]::Dns -or $domain -notmatch '^[A-Za-z0-9.-]+$') {
        throw "ENTCOIN_SEED_DOMAIN must be one ASCII DNS hostname without a scheme or path"
    }

    try {
        $mail = [System.Net.Mail.MailAddress]::new([string]$Values["ENTCOIN_ACME_EMAIL"])
    }
    catch {
        throw "ENTCOIN_ACME_EMAIL must be a valid email address"
    }
    if ($mail.Address -ne [string]$Values["ENTCOIN_ACME_EMAIL"]) {
        throw "ENTCOIN_ACME_EMAIL must not contain a display name"
    }

    foreach ($name in @("ENTCOIN_INSTALL_DIR", "ENTCOIN_DATA_DIR", "ENTCOIN_CADDY_DATA_DIR", "ENTCOIN_LOG_DIR")) {
        $value = [string]$Values[$name]
        if ([string]::IsNullOrWhiteSpace($value) -or -not [System.IO.Path]::IsPathRooted($value)) {
            throw "$name must be an absolute Windows path"
        }
        $fullPath = [System.IO.Path]::GetFullPath($value).TrimEnd([System.IO.Path]::DirectorySeparatorChar)
        $rootPath = [System.IO.Path]::GetPathRoot($fullPath).TrimEnd([System.IO.Path]::DirectorySeparatorChar)
        if ($fullPath.Equals($rootPath, [System.StringComparison]::OrdinalIgnoreCase)) {
            throw "$name must not be a filesystem root"
        }
    }
    $installDirectory = [string]$Values["ENTCOIN_INSTALL_DIR"]
    $runtimeNames = @("ENTCOIN_DATA_DIR", "ENTCOIN_CADDY_DATA_DIR", "ENTCOIN_LOG_DIR")
    foreach ($name in $runtimeNames) {
        $runtimeDirectory = [string]$Values[$name]
        if ((Test-SeedPathWithin -Child $runtimeDirectory -Parent $installDirectory) -or
            (Test-SeedPathWithin -Child $installDirectory -Parent $runtimeDirectory)) {
            throw "$name must be outside ENTCOIN_INSTALL_DIR so upgrades cannot remove runtime data"
        }
    }
    for ($left = 0; $left -lt $runtimeNames.Count; $left++) {
        for ($right = $left + 1; $right -lt $runtimeNames.Count; $right++) {
            $leftPath = [string]$Values[$runtimeNames[$left]]
            $rightPath = [string]$Values[$runtimeNames[$right]]
            if ((Test-SeedPathWithin -Child $leftPath -Parent $rightPath) -or
                (Test-SeedPathWithin -Child $rightPath -Parent $leftPath)) {
                throw "$($runtimeNames[$left]) and $($runtimeNames[$right]) must not overlap"
            }
        }
    }

    $listen = [string]$Values["ENTCOIN_LISTEN_ADDRESS"]
    if ($listen -notmatch '^127\.0\.0\.1:([0-9]{1,5})$') {
        throw "ENTCOIN_LISTEN_ADDRESS must use the loopback form 127.0.0.1:PORT"
    }
    $port = [int]$Matches[1]
    if ($port -lt 1 -or $port -gt 65535) {
        throw "ENTCOIN_LISTEN_ADDRESS port must be between 1 and 65535"
    }
    if ($port -ne 47821) {
        throw "The public seed deployment requires ENTCOIN_LISTEN_ADDRESS=127.0.0.1:47821"
    }

    if ([string]$Values["ENTCOIN_PRUNE_DEPTH"] -ne "0") {
        throw "A public bootstrap seed must use ENTCOIN_PRUNE_DEPTH=0 (archive mode)"
    }
    if ([string]$Values["ENTCOIN_DISABLE_DISCOVERY"] -ne "true") {
        throw "A public seed must use ENTCOIN_DISABLE_DISCOVERY=true"
    }
    if ([string]$Values["ENTCOIN_TRUST_LOOPBACK_PROXY"] -ne "true") {
        throw "Caddy requires ENTCOIN_TRUST_LOOPBACK_PROXY=true"
    }

    $peer = [string]$Values["ENTCOIN_BOOTSTRAP_PEER"]
    if (-not [string]::IsNullOrWhiteSpace($peer) -and -not (Test-SeedHttpsBaseUrl -Value $peer)) {
        throw "ENTCOIN_BOOTSTRAP_PEER must be an HTTPS base URL without a path, query, or fragment"
    }
    $manifestURLs = @(([string]$Values["ENTCOIN_BOOTSTRAP_MANIFEST_URLS"]).Split(",") | ForEach-Object { $_.Trim() })
    if ($manifestURLs.Count -eq 0 -or $manifestURLs.Count -gt 8 -or $manifestURLs -contains "") {
        throw "ENTCOIN_BOOTSTRAP_MANIFEST_URLS must contain between 1 and 8 comma-separated URLs"
    }
    foreach ($manifestURL in $manifestURLs) {
        if (-not (Test-SeedHttpsUrl -Value $manifestURL)) {
            throw "ENTCOIN_BOOTSTRAP_MANIFEST_URLS contains an invalid HTTPS URL"
        }
    }
}

function Set-SeedProcessEnvironment {
    [CmdletBinding()]
    param([Parameter(Mandatory)][System.Collections.IDictionary]$Values)

    foreach ($name in $script:SeedEnvironmentNames) {
        [System.Environment]::SetEnvironmentVariable($name, [string]$Values[$name], "Process")
    }
}
