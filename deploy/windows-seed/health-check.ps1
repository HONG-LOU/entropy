[CmdletBinding()]
param(
    [string]$ConfigPath = (Join-Path $PSScriptRoot "seed.env"),
    [ValidateRange(1, 60)][int]$TimeoutSeconds = 10,
    [switch]$LocalOnly,
    [switch]$SkipWebSocket
)

$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "seed-config.ps1")

function Assert-EntcoinStatus {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)]$Status,
        [Parameter(Mandatory)][string]$Source
    )

    if ($Status.protocol -ne "entropy-mainnet-v1" -or $Status.name -ne "Entropy" -or $Status.symbol -ne "ENT") {
        throw "$Source returned an incompatible Entcoin network identity"
    }
    if ([uint64]$Status.height -lt 0 -or [string]$Status.tip_hash -notmatch '^[0-9a-f]{64}$') {
        throw "$Source returned malformed chain status"
    }
    $work = [System.Numerics.BigInteger]::Zero
    if (-not [System.Numerics.BigInteger]::TryParse([string]$Status.chain_work, [ref]$work) -or $work.Sign -lt 0) {
        throw "$Source returned malformed cumulative work"
    }
}

function Receive-EntcoinHello {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][System.Uri]$Uri,
        [Parameter(Mandatory)][int]$Timeout
    )

    $socket = [System.Net.WebSockets.ClientWebSocket]::new()
    $cancellation = [System.Threading.CancellationTokenSource]::new([TimeSpan]::FromSeconds($Timeout))
    try {
        $socket.ConnectAsync($Uri, $cancellation.Token).GetAwaiter().GetResult()
        $buffer = [byte[]]::new(65536)
        $segment = [System.ArraySegment[byte]]::new($buffer)
        $result = $socket.ReceiveAsync($segment, $cancellation.Token).GetAwaiter().GetResult()
        if ($result.MessageType -ne [System.Net.WebSockets.WebSocketMessageType]::Text -or $result.Count -le 0) {
            throw "WSS endpoint did not return an Entcoin hello message"
        }
        $message = [System.Text.Encoding]::UTF8.GetString($buffer, 0, $result.Count) | ConvertFrom-Json
        if ($message.type -ne "hello" -or $message.protocol -ne "entropy-mainnet-v1") {
            throw "WSS endpoint returned an incompatible hello message"
        }
        $socket.CloseAsync(
            [System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure,
            "health check",
            [System.Threading.CancellationToken]::None
        ).GetAwaiter().GetResult()
    }
    finally {
        $cancellation.Dispose()
        $socket.Dispose()
    }
}

try {
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls12
    $config = Read-SeedEnvironment -Path $ConfigPath
    Assert-SeedEnvironment -Values $config

    $localUri = "http://$($config['ENTCOIN_LISTEN_ADDRESS'])/v2/status"
    $local = Invoke-RestMethod -Uri $localUri -TimeoutSec $TimeoutSeconds
    Assert-EntcoinStatus -Status $local -Source $localUri

    if (-not $LocalOnly) {
        $publicBase = "https://$($config['ENTCOIN_SEED_DOMAIN'])"
        $publicUri = "$publicBase/v2/status"
        $public = Invoke-RestMethod -Uri $publicUri -TimeoutSec $TimeoutSeconds
        Assert-EntcoinStatus -Status $public -Source $publicUri
        if ([uint64]$public.height -ne [uint64]$local.height -or [string]$public.tip_hash -ne [string]$local.tip_hash) {
            throw "Public status does not match the loopback node"
        }
        if (-not $SkipWebSocket) {
            Receive-EntcoinHello -Uri ([System.Uri]("wss://$($config['ENTCOIN_SEED_DOMAIN'])/v2/p2p")) -Timeout $TimeoutSeconds
        }
    }

    [PSCustomObject]@{
        Healthy = $true
        Domain = [string]$config["ENTCOIN_SEED_DOMAIN"]
        Height = [uint64]$local.height
        TipHash = [string]$local.tip_hash
        PublicChecked = -not $LocalOnly
        WebSocketChecked = -not $LocalOnly -and -not $SkipWebSocket
    }
    exit 0
}
catch {
    Write-Error $_
    exit 1
}
