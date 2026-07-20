[CmdletBinding(SupportsShouldProcess, ConfirmImpact = "High")]
param(
    [string]$ConfigPath = (Join-Path $PSScriptRoot "seed.env"),
    [switch]$RemoveRuntimeData
)

$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "seed-config.ps1")

$identity = [System.Security.Principal.WindowsIdentity]::GetCurrent()
$principal = [System.Security.Principal.WindowsPrincipal]::new($identity)
if (-not $principal.IsInRole([System.Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw "Run uninstall.ps1 from an elevated PowerShell session"
}

$config = Read-SeedEnvironment -Path $ConfigPath
Assert-SeedEnvironment -Values $config
foreach ($taskName in @("EntcoinSeedNode", "EntcoinSeedProxy")) {
    if ($PSCmdlet.ShouldProcess($taskName, "Stop and unregister scheduled task")) {
        Stop-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
        $deadline = (Get-Date).AddSeconds(20)
        do {
            $task = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
            if ($null -eq $task -or $task.State -ne "Running") {
                break
            }
            Start-Sleep -Milliseconds 250
        } while ((Get-Date) -lt $deadline)
        if ($null -ne $task -and $task.State -eq "Running") {
            throw "Scheduled task $taskName did not stop within 20 seconds"
        }
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    }
}
if ($PSCmdlet.ShouldProcess("Entcoin Seed HTTPS", "Remove TCP 443 firewall rule")) {
    Get-NetFirewallRule -DisplayName "Entcoin Seed HTTPS" -ErrorAction SilentlyContinue | Remove-NetFirewallRule
}

$installDirectory = [string]$config["ENTCOIN_INSTALL_DIR"]
if ($PSCmdlet.ShouldProcess($installDirectory, "Remove installed seed binaries and configuration")) {
    Remove-Item -LiteralPath $installDirectory -Recurse -Force -ErrorAction SilentlyContinue
}

if ($RemoveRuntimeData) {
    foreach ($directory in @(
        [string]$config["ENTCOIN_DATA_DIR"],
        [string]$config["ENTCOIN_CADDY_DATA_DIR"],
        [string]$config["ENTCOIN_LOG_DIR"]
    )) {
        if ($PSCmdlet.ShouldProcess($directory, "Permanently remove seed runtime data")) {
            Remove-Item -LiteralPath $directory -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}
else {
    Write-Host "Runtime data was retained. Use -RemoveRuntimeData only after verifying wallet recovery."
}
