[CmdletBinding(SupportsShouldProcess)]
param(
    [string]$ConfigPath = (Join-Path $PSScriptRoot "seed.env"),
    [string]$EntcoinCliPath = (Join-Path $PSScriptRoot "entcoin-cli.exe"),
    [string]$CaddyPath = (Join-Path $PSScriptRoot "caddy.exe"),
    [System.Management.Automation.PSCredential]$NodeCredential,
    [switch]$SkipFirewall,
    [switch]$SkipStart
)

$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "seed-config.ps1")

$nodeTaskName = "EntcoinSeedNode"
$proxyTaskName = "EntcoinSeedProxy"
$firewallRuleName = "Entcoin Seed HTTPS"
$windowsPowerShell = "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe"

function Assert-Administrator {
    $identity = [System.Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [System.Security.Principal.WindowsPrincipal]::new($identity)
    if (-not $principal.IsInRole([System.Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw "Run install.ps1 from an elevated PowerShell session"
    }
}

function Stop-AndRemoveTask {
    param([Parameter(Mandatory)][string]$Name)

    $task = Get-ScheduledTask -TaskName $Name -ErrorAction SilentlyContinue
    if ($null -ne $task) {
        Stop-ScheduledTask -TaskName $Name -ErrorAction SilentlyContinue
        $deadline = (Get-Date).AddSeconds(20)
        do {
            $task = Get-ScheduledTask -TaskName $Name -ErrorAction SilentlyContinue
            if ($null -eq $task -or $task.State -ne "Running") {
                break
            }
            Start-Sleep -Milliseconds 250
        } while ((Get-Date) -lt $deadline)
        if ($null -ne $task -and $task.State -eq "Running") {
            throw "Scheduled task $Name did not stop within 20 seconds"
        }
        Unregister-ScheduledTask -TaskName $Name -Confirm:$false
    }
}

function Invoke-Icacls {
    param(
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)][string[]]$Arguments
    )

    & icacls.exe $Path @Arguments | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "icacls failed for $Path with exit code $LASTEXITCODE"
    }
}

function Wait-ForLocalNode {
    param([Parameter(Mandatory)][string]$Address)

    $deadline = (Get-Date).AddSeconds(45)
    do {
        try {
            $status = Invoke-RestMethod -Uri "http://$Address/v2/status" -TimeoutSec 3
            if ($status.protocol -eq "entropy-mainnet-v1") {
                return
            }
        }
        catch {
        }
        Start-Sleep -Seconds 1
    } while ((Get-Date) -lt $deadline)
    throw "Entcoin seed node did not become healthy within 45 seconds"
}

Assert-Administrator
$config = Read-SeedEnvironment -Path $ConfigPath
Assert-SeedEnvironment -Values $config
if ([string]$config["ENTCOIN_SEED_DOMAIN"] -like "*.example.com" -or
    [string]$config["ENTCOIN_ACME_EMAIL"] -like "*@example.com") {
    throw "Replace the example domain and ACME email before installation"
}

foreach ($path in @($EntcoinCliPath, $CaddyPath)) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required executable is missing: $path"
    }
}
if ($null -eq $NodeCredential) {
    $NodeCredential = Get-Credential -Message "Dedicated Windows account for the walletless Entcoin seed service"
}
$nodeAccount = $NodeCredential.UserName
try {
    $nodeSid = ([System.Security.Principal.NTAccount]$nodeAccount).Translate([System.Security.Principal.SecurityIdentifier]).Value
}
catch {
    throw "Cannot resolve the node service account '$nodeAccount'"
}

$installDirectory = [string]$config["ENTCOIN_INSTALL_DIR"]
$dataDirectory = [string]$config["ENTCOIN_DATA_DIR"]
$caddyDataDirectory = [string]$config["ENTCOIN_CADDY_DATA_DIR"]
$logDirectory = [string]$config["ENTCOIN_LOG_DIR"]
if (-not $PSCmdlet.ShouldProcess($installDirectory, "Install Entcoin public seed scheduled tasks")) {
    return
}

$registeredTasks = @()
$firewallCreated = $false
try {
    Stop-AndRemoveTask -Name $nodeTaskName
    Stop-AndRemoveTask -Name $proxyTaskName

    foreach ($directory in @($installDirectory, $dataDirectory, $caddyDataDirectory, $logDirectory)) {
        New-Item -ItemType Directory -Path $directory -Force | Out-Null
    }
    Copy-Item -LiteralPath $EntcoinCliPath -Destination (Join-Path $installDirectory "entcoin-cli.exe") -Force
    Copy-Item -LiteralPath $CaddyPath -Destination (Join-Path $installDirectory "caddy.exe") -Force
    foreach ($name in @("Caddyfile", "seed-config.ps1", "run-node.ps1", "run-proxy.ps1", "health-check.ps1")) {
        Copy-Item -LiteralPath (Join-Path $PSScriptRoot $name) -Destination (Join-Path $installDirectory $name) -Force
    }
    Copy-Item -LiteralPath $ConfigPath -Destination (Join-Path $installDirectory "seed.env") -Force

    Invoke-Icacls -Path $installDirectory -Arguments @(
        "/inheritance:r",
        "/grant:r",
        "*S-1-5-18:(OI)(CI)(F)",
        "*S-1-5-32-544:(OI)(CI)(F)",
        "*S-1-5-19:(OI)(CI)(RX)",
        "*${nodeSid}:(OI)(CI)(RX)"
    )
    Invoke-Icacls -Path $dataDirectory -Arguments @(
        "/inheritance:r",
        "/grant:r",
        "*S-1-5-18:(OI)(CI)(F)",
        "*S-1-5-32-544:(OI)(CI)(F)",
        "*${nodeSid}:(OI)(CI)(M)"
    )
    Invoke-Icacls -Path $caddyDataDirectory -Arguments @(
        "/inheritance:r",
        "/grant:r",
        "*S-1-5-18:(OI)(CI)(F)",
        "*S-1-5-32-544:(OI)(CI)(F)",
        "*S-1-5-19:(OI)(CI)(M)"
    )
    Invoke-Icacls -Path $logDirectory -Arguments @(
        "/inheritance:r",
        "/grant:r",
        "*S-1-5-18:(OI)(CI)(F)",
        "*S-1-5-32-544:(OI)(CI)(F)",
        "*S-1-5-19:(OI)(CI)(M)",
        "*${nodeSid}:(OI)(CI)(M)"
    )

    Set-SeedProcessEnvironment -Values $config
    & (Join-Path $installDirectory "caddy.exe") validate --config (Join-Path $installDirectory "Caddyfile") --adapter caddyfile
    if ($LASTEXITCODE -ne 0) {
        throw "Caddy configuration validation failed with exit code $LASTEXITCODE"
    }

    $settings = New-ScheduledTaskSettingsSet `
        -AllowStartIfOnBatteries `
        -DontStopIfGoingOnBatteries `
        -StartWhenAvailable `
        -ExecutionTimeLimit ([TimeSpan]::Zero) `
        -MultipleInstances IgnoreNew `
        -RestartCount 999 `
        -RestartInterval (New-TimeSpan -Minutes 1)
    $trigger = New-ScheduledTaskTrigger -AtStartup
    $nodeArguments = '-NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "{0}" -ConfigPath "{1}"' -f `
        (Join-Path $installDirectory "run-node.ps1"), (Join-Path $installDirectory "seed.env")
    $nodeAction = New-ScheduledTaskAction -Execute $windowsPowerShell -Argument $nodeArguments -WorkingDirectory $installDirectory
    $plainPassword = $NodeCredential.GetNetworkCredential().Password
    Register-ScheduledTask `
        -TaskName $nodeTaskName `
        -Action $nodeAction `
        -Trigger $trigger `
        -Settings $settings `
        -User $nodeAccount `
        -Password $plainPassword `
        -RunLevel Limited `
        -Description "Entcoin walletless archive seed node under its dedicated service account" `
        -Force | Out-Null
    $plainPassword = $null
    $registeredTasks += $nodeTaskName

    $proxyArguments = '-NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "{0}" -ConfigPath "{1}"' -f `
        (Join-Path $installDirectory "run-proxy.ps1"), (Join-Path $installDirectory "seed.env")
    $proxyAction = New-ScheduledTaskAction -Execute $windowsPowerShell -Argument $proxyArguments -WorkingDirectory $installDirectory
    $proxyPrincipal = New-ScheduledTaskPrincipal -UserId "NT AUTHORITY\LOCAL SERVICE" -LogonType ServiceAccount -RunLevel Limited
    Register-ScheduledTask `
        -TaskName $proxyTaskName `
        -Action $proxyAction `
        -Trigger $trigger `
        -Settings $settings `
        -Principal $proxyPrincipal `
        -Description "Caddy HTTPS and WSS proxy for the Entcoin archive seed" `
        -Force | Out-Null
    $registeredTasks += $proxyTaskName

    if (-not $SkipFirewall) {
        Get-NetFirewallRule -DisplayName $firewallRuleName -ErrorAction SilentlyContinue | Remove-NetFirewallRule
        New-NetFirewallRule `
            -DisplayName $firewallRuleName `
            -Direction Inbound `
            -Action Allow `
            -Protocol TCP `
            -LocalPort 443 `
            -Profile Any | Out-Null
        $firewallCreated = $true
    }

    if (-not $SkipStart) {
        Start-ScheduledTask -TaskName $nodeTaskName
        Wait-ForLocalNode -Address ([string]$config["ENTCOIN_LISTEN_ADDRESS"])
        Start-ScheduledTask -TaskName $proxyTaskName
        Start-Sleep -Seconds 2
        $proxyTask = Get-ScheduledTask -TaskName $proxyTaskName
        if ($proxyTask.State -ne "Running") {
            throw "Caddy proxy task exited during startup; inspect the Caddy console log"
        }
    }

    Write-Host "Installed Entcoin seed tasks: $nodeTaskName, $proxyTaskName"
    Write-Host "Public endpoint: https://$($config['ENTCOIN_SEED_DOMAIN'])"
    Write-Host "Run health-check.ps1 after DNS and ACME certificate issuance are ready."
}
catch {
    foreach ($taskName in $registeredTasks) {
        Stop-AndRemoveTask -Name $taskName
    }
    if ($firewallCreated) {
        Get-NetFirewallRule -DisplayName $firewallRuleName -ErrorAction SilentlyContinue | Remove-NetFirewallRule
    }
    throw
}
