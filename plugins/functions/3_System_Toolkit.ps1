# =============================================================================
# SYSTEM TOOLKIT – Local system & network operational layer (standalone)
# Windows system helpers for local environments.
# Safety: Non-destructive defaults. Kill/restart require -Force or confirmation.
# Entry point: sys_*
#
# FUNCTIONS
#   sys_uptime
#   sys_os
#   sys_path
#   sys_top_cpu
#   sys_top_mem
#   sys_service_status
#   sys_service_restart
#   sys_disk
#   sys_big
#   sys_size
#   sys_now
#   sys_log
#   sys_clip
#   sys_paste
#   sys_ip
#   sys_ports
#   sys_dns
#   sys_ping
#   sys_net_summary
#   sys_test_port
#   sys_wifi_list
#   sys_wifi_current
#   sys_wifi_connect
#   sys_wifi_pick
#   sys_wifi_disconnect
#   sys_wifi_signal
#   sys_wifi_switch
#   sys_pgrep
#   sys_pkill
#   sys_version_git
#   sys_version_go
#   sys_version_node
#   sys_which
#   sys_events
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal helpers — guards
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Ensure a command is available in PATH.
.PARAMETER Name
Command name to validate.
.EXAMPLE
_assert_command_available -Name docker
#>
function _assert_command_available {
    param([Parameter(Mandatory = $true)][string]$Name)
    if (-not (Get-Command -Name $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' was not found in PATH."
    }
}

<#
.SYNOPSIS
Ensure a filesystem path exists.
.PARAMETER Path
Path to validate.
.EXAMPLE
_assert_path_exists -Path "C:\Data"
#>
function _assert_path_exists {
    param([Parameter(Mandatory = $true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Required path '$Path' does not exist."
    }
}

<#
.SYNOPSIS
Ask for yes/no confirmation before a risky action.
.PARAMETER Prompt
Message shown to the user.
.EXAMPLE
if (-not (_confirm_action -Prompt "Continue?")) { return }
#>
function _confirm_action {
    param([Parameter(Mandatory = $true)][string]$Prompt)
    $answer = Read-Host "$Prompt [y/N]"
    if ([string]::IsNullOrWhiteSpace($answer)) { return $false }
    return $answer.Trim().ToLowerInvariant() -in @("y", "yes")
}

# -----------------------------------------------------------------------------
# Internal helpers — Wi-Fi
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Get saved Wi-Fi profile names.
.DESCRIPTION
Parses netsh wlan show profiles and returns an array of profile names.
.EXAMPLE
_wifi_profiles
#>
function _wifi_profiles {
    _assert_command_available -Name netsh
    $lines = netsh wlan show profiles
    $profiles = @()
    foreach ($line in $lines) {
        if ($line -match "All User Profile\s*:\s*(.+)$") {
            $profiles += $Matches[1].Trim()
        }
    }
    return $profiles
}

# -----------------------------------------------------------------------------
# System
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Show system uptime.
.DESCRIPTION
Returns elapsed time since last OS boot.
.EXAMPLE
sys_uptime
#>
function sys_uptime {
    $os = Get-CimInstance Win32_OperatingSystem
    $boot = $os.LastBootUpTime
    $uptime = (Get-Date) - $boot

    return [pscustomobject]@{
        Days    = $uptime.Days
        Hours   = $uptime.Hours
        Minutes = $uptime.Minutes
        Boot    = $boot
    }
}

<#
.SYNOPSIS
Show operating system information.
.DESCRIPTION
Returns Windows caption, version, build, and architecture.
.EXAMPLE
sys_os
#>
function sys_os {
    $os = Get-CimInstance Win32_OperatingSystem

    return [pscustomobject]@{
        Caption      = $os.Caption
        Version      = $os.Version
        Build        = $os.BuildNumber
        Architecture = $os.OSArchitecture
    }
}

<#
.SYNOPSIS
Show PATH entries.
.DESCRIPTION
Returns PATH entries one per line.
.EXAMPLE
sys_path
#>
function sys_path {
    $env:Path -split ";" | Where-Object { $_ -and $_.Trim() -ne "" }
}

<#
.SYNOPSIS
Show top processes by CPU.
.DESCRIPTION
Lists top N processes sorted by CPU time.
.PARAMETER Count
Number of processes to show (default 15).
.EXAMPLE
sys_top_cpu
.EXAMPLE
sys_top_cpu -Count 20
#>
function sys_top_cpu {
    param([int]$Count = 15)

    Get-Process |
        Select-Object Name, Id, @{
            Name="CPUSeconds"
            Expression={
                if ($null -eq $_.CPU) { return 0 }
                if ($_.CPU -is [TimeSpan]) {
                    return [math]::Round($_.CPU.TotalSeconds, 3)
                }
                return [math]::Round([double]$_.CPU, 3)
            }
        } |
        Sort-Object CPUSeconds -Descending |
        Select-Object -First $Count
}

<#
.SYNOPSIS
Show top processes by memory.
.DESCRIPTION
Lists top N processes sorted by working set.
.PARAMETER Count
Number of processes to show (default 15).
.EXAMPLE
sys_top_mem
.EXAMPLE
sys_top_mem -Count 20
#>
function sys_top_mem {
    param([int]$Count = 15)

    Get-Process |
        Select-Object Name, Id, @{
            Name="RAM_MB"
            Expression={
                if ($null -eq $_.WorkingSet64) { return 0 }
                return [math]::Round($_.WorkingSet64 / 1MB, 1)
            }
        } |
        Sort-Object RAM_MB -Descending |
        Select-Object -First $Count
}

<#
.SYNOPSIS
Show Windows service status.
.PARAMETER Name
Service name.
.EXAMPLE
sys_service_status -Name Spooler
#>
function sys_service_status {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    Get-Service -Name $Name |
        Select-Object Name, DisplayName, Status, StartType
}

<#
.SYNOPSIS
Restart a Windows service.
.DESCRIPTION
Restarts a service. Requires -Force or interactive confirmation.
.PARAMETER Name
Service name.
.PARAMETER Force
Skip interactive confirmation.
.EXAMPLE
sys_service_restart -Name Spooler -Force
#>
function sys_service_restart {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [switch]$Force
    )

    if (-not $Force) {
        if (-not (_confirm_action -Prompt "Restart service '$Name'?")) {
            return [pscustomobject]@{ Service = $Name; Status = "Cancelled" }
        }
    }

    Restart-Service -Name $Name -ErrorAction Stop
    $svc = Get-Service -Name $Name

    return [pscustomobject]@{
        Service = $svc.Name
        Status  = $svc.Status.ToString()
    }
}

<#
.SYNOPSIS
Show disk usage.
.DESCRIPTION
Lists local disks with used and free space in GB.
.EXAMPLE
sys_disk
#>
function sys_disk {
    Get-PSDrive -PSProvider FileSystem |
        Select-Object Name,
        @{Name = "UsedGB"; Expression = { [math]::Round($_.Used / 1GB, 2) } },
        @{Name = "FreeGB"; Expression = { [math]::Round($_.Free / 1GB, 2) } }
}

<#
.SYNOPSIS
List largest files in a directory tree.
.PARAMETER Path
Base directory path.
.PARAMETER Top
Number of files to return (default 20).
.EXAMPLE
sys_big -Path .
.EXAMPLE
sys_big -Path C:\Data -Top 10
#>
function sys_big {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path,
        [int]$Top = 20
    )

    _assert_path_exists -Path $Path

    Get-ChildItem -Path $Path -Recurse -File -ErrorAction SilentlyContinue |
        Sort-Object Length -Descending |
        Select-Object -First $Top FullName,
        @{Name = "SizeMB"; Expression = { [math]::Round($_.Length / 1MB, 2) } }
}

<#
.SYNOPSIS
Show total directory size.
.DESCRIPTION
Computes recursive sum of file sizes in a directory.
.PARAMETER Path
Directory path.
.EXAMPLE
sys_size -Path .
#>
function sys_size {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    _assert_path_exists -Path $Path

    $sum = (Get-ChildItem -Path $Path -Recurse -File -ErrorAction SilentlyContinue |
        Measure-Object Length -Sum).Sum

    if ($null -eq $sum) { $sum = 0 }

    return [pscustomobject]@{
        Path   = (Resolve-Path $Path).Path
        Bytes  = $sum
        SizeMB = [math]::Round($sum / 1MB, 2)
        SizeGB = [math]::Round($sum / 1GB, 2)
    }
}

<#
.SYNOPSIS
Show current timestamp.
.EXAMPLE
sys_now
#>
function sys_now {
    Get-Date -Format "yyyy-MM-dd HH:mm:ss"
}

<#
.SYNOPSIS
Write timestamped log message.
.PARAMETER Message
Message to log.
.EXAMPLE
sys_log -Message "Deployment started"
#>
function sys_log {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Message
    )

    return [pscustomobject]@{
        Timestamp = (Get-Date -Format "yyyy-MM-dd HH:mm:ss")
        Message   = $Message
    }
}

<#
.SYNOPSIS
Copy text to clipboard.
.PARAMETER Text
Text to copy.
.EXAMPLE
sys_clip -Text "hello"
#>
function sys_clip {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Text
    )
    $Text | Set-Clipboard
}

<#
.SYNOPSIS
Read clipboard content.
.EXAMPLE
sys_paste
#>
function sys_paste {
    Get-Clipboard
}

# -----------------------------------------------------------------------------
# Network
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Show local IPv4 addresses.
.DESCRIPTION
Lists active IPv4 addresses per interface, excluding loopback.
.EXAMPLE
sys_ip
#>
function sys_ip {
    Get-NetIPAddress -AddressFamily IPv4 |
        Where-Object { $_.IPAddress -ne "127.0.0.1" } |
        Select-Object InterfaceAlias, IPAddress, PrefixLength
}

<#
.SYNOPSIS
Show listening TCP ports.
.DESCRIPTION
Lists listening TCP ports sorted by port number.
.EXAMPLE
sys_ports
#>
function sys_ports {
    Get-NetTCPConnection -State Listen |
        Select-Object LocalAddress, LocalPort, OwningProcess |
        Sort-Object LocalPort
}

<#
.SYNOPSIS
Test DNS resolution.
.PARAMETER Host
Host name to resolve.
.EXAMPLE
sys_dns -Host openai.com
#>
function sys_dns {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Host
    )
    Resolve-DnsName -Name $Host
}

<#
.SYNOPSIS
Ping a host.
.PARAMETER Host
Host to ping.
.PARAMETER Count
Number of packets (default 4).
.EXAMPLE
sys_ping -Host 8.8.8.8
.EXAMPLE
sys_ping -Host google.com -Count 10
#>
function sys_ping {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Host,
        [int]$Count = 4
    )
    Test-Connection -ComputerName $Host -Count $Count
}

<#
.SYNOPSIS
Show network connection summary.
.DESCRIPTION
Groups TCP connections by state.
.EXAMPLE
sys_net_summary
#>
function sys_net_summary {
    Get-NetTCPConnection |
        Group-Object State |
        Sort-Object Name |
        Select-Object Name, Count
}

<#
.SYNOPSIS
Test remote TCP port.
.PARAMETER ComputerName
Remote host.
.PARAMETER Port
Remote port.
.EXAMPLE
sys_test_port -ComputerName google.com -Port 443
#>
function sys_test_port {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ComputerName,
        [Parameter(Mandatory = $true)]
        [int]$Port
    )
    Test-NetConnection -ComputerName $ComputerName -Port $Port
}

# -----------------------------------------------------------------------------
# Wi-Fi
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
List saved Wi-Fi profiles.
.EXAMPLE
sys_wifi_list
#>
function sys_wifi_list {
    _wifi_profiles
}

<#
.SYNOPSIS
Show current Wi-Fi connection.
.DESCRIPTION
Returns current SSID, state and interface details.
.EXAMPLE
sys_wifi_current
#>
function sys_wifi_current {
    _assert_command_available -Name netsh
    netsh wlan show interfaces
}

<#
.SYNOPSIS
Connect to a saved Wi-Fi profile.
.DESCRIPTION
Validates the profile exists, connects, waits briefly and returns status.
.PARAMETER Name
Saved Wi-Fi profile name.
.EXAMPLE
sys_wifi_connect -Name HomeWifi
#>
function sys_wifi_connect {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    _assert_command_available -Name netsh

    $profiles = _wifi_profiles
    if ($Name -notin $profiles) {
        throw "Wi-Fi profile '$Name' not found."
    }

    netsh wlan connect name="$Name" | Out-Null
    Start-Sleep -Seconds 2

    $status = netsh wlan show interfaces

    return [pscustomobject]@{
        Action  = "Connect"
        Profile = $Name
        Status  = if ($status -match "State\s*:\s*connected") { "Connected" } else { "Unknown" }
    }
}

<#
.SYNOPSIS
Interactively pick a Wi-Fi profile and connect.
.DESCRIPTION
Displays an indexed list and connects to the selected profile.
.EXAMPLE
sys_wifi_pick
#>
function sys_wifi_pick {
    $profiles = @(_wifi_profiles)

    if ($profiles.Count -eq 0) {
        throw "No saved Wi-Fi profiles found."
    }

    for ($i = 0; $i -lt $profiles.Count; $i++) {
        "{0}. {1}" -f ($i + 1), $profiles[$i]
    }

    $raw = Read-Host "Select profile number"
    $index = 0

    if (-not [int]::TryParse($raw, [ref]$index)) {
        throw "Invalid selection."
    }

    if ($index -lt 1 -or $index -gt $profiles.Count) {
        throw "Selection out of range."
    }

    $name = $profiles[$index - 1]
    sys_wifi_connect -Name $name
}

<#
.SYNOPSIS
Disconnect current Wi-Fi.
.EXAMPLE
sys_wifi_disconnect
#>
function sys_wifi_disconnect {
    _assert_command_available -Name netsh
    netsh wlan disconnect | Out-Null

    return [pscustomobject]@{
        Action = "Disconnect"
        Status = "Disconnected"
    }
}

<#
.SYNOPSIS
Show Wi-Fi signal details.
.DESCRIPTION
Returns signal strength, radio type and rates for the current connection.
.EXAMPLE
sys_wifi_signal
#>
function sys_wifi_signal {
    _assert_command_available -Name netsh
    $lines = netsh wlan show interfaces
    $lines | Select-String -Pattern "^\s*(Name|Description|State|SSID|Signal|Radio type|Receive rate|Transmit rate)\s*:"
}

<#
.SYNOPSIS
Disconnect current Wi-Fi and connect to another profile.
.DESCRIPTION
Performs a clean Wi-Fi switch: disconnects, then connects to the target profile
with a configurable timeout.
.PARAMETER ProfileName
Target Wi-Fi profile name.
.PARAMETER TimeoutSeconds
Maximum seconds to wait for connection (default 15).
.EXAMPLE
sys_wifi_switch -ProfileName "HomeWifi"
.EXAMPLE
sys_wifi_switch -ProfileName "OfficeWifi" -TimeoutSeconds 30
#>
function sys_wifi_switch {
    param(
        [Parameter(Mandatory = $true, Position = 0)]
        [string]$ProfileName,

        [Parameter(Position = 1)]
        [int]$TimeoutSeconds = 15
    )

    _assert_command_available -Name netsh

    $profiles = _wifi_profiles
    if ($ProfileName -notin $profiles) {
        throw "Wi-Fi profile '$ProfileName' not found."
    }

    netsh wlan disconnect | Out-Null
    Start-Sleep -Seconds 2

    netsh wlan connect name="$ProfileName" | Out-Null

    $elapsed = 0
    $connected = $false

    while ($elapsed -lt $TimeoutSeconds) {
        $status = netsh wlan show interfaces

        if ($status -match "State\s*:\s*connected" -and
            $status -match "SSID\s*:\s*$ProfileName") {
            $connected = $true
            break
        }

        Start-Sleep -Seconds 1
        $elapsed++
    }

    if (-not $connected) {
        throw "Failed to connect to '$ProfileName' within $TimeoutSeconds seconds."
    }

    return [pscustomobject]@{
        Action  = "Switch"
        Profile = $ProfileName
        Status  = "Connected"
        Elapsed = $elapsed
    }
}

# -----------------------------------------------------------------------------
# Process
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Find processes by name.
.PARAMETER Name
Process name fragment (wildcard match).
.EXAMPLE
sys_pgrep -Name chrome
#>
function sys_pgrep {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    Get-Process *$Name* |
        Select-Object Name, Id, CPU, WorkingSet
}

<#
.SYNOPSIS
Terminate processes by name.
.DESCRIPTION
Stops all processes matching the provided name. Requires -Force or confirmation.
.PARAMETER Name
Process name fragment (wildcard match).
.PARAMETER Force
Skip interactive confirmation.
.EXAMPLE
sys_pkill -Name notepad -Force
#>
function sys_pkill {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [switch]$Force
    )

    $procs = Get-Process *$Name* -ErrorAction SilentlyContinue

    if (-not $procs) {
        return [pscustomobject]@{ Name = $Name; Killed = 0; Status = "No matching processes" }
    }

    if (-not $Force) {
        $count = $procs.Count
        if (-not (_confirm_action -Prompt "Kill $count process(es) matching '$Name'?")) {
            return [pscustomobject]@{ Name = $Name; Killed = 0; Status = "Cancelled" }
        }
    }

    $killed = $procs.Count
    $procs | Stop-Process -Force

    return [pscustomobject]@{ Name = $Name; Killed = $killed; Status = "Terminated" }
}

# -----------------------------------------------------------------------------
# Dev tools
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Show Git version.
.EXAMPLE
sys_version_git
#>
function sys_version_git {
    _assert_command_available -Name git
    git --version
}

<#
.SYNOPSIS
Show Go version.
.EXAMPLE
sys_version_go
#>
function sys_version_go {
    _assert_command_available -Name go
    go version
}

<#
.SYNOPSIS
Show Node.js version.
.EXAMPLE
sys_version_node
#>
function sys_version_node {
    _assert_command_available -Name node
    node --version
}

<#
.SYNOPSIS
Find a command in PATH.
.PARAMETER Name
Command name.
.EXAMPLE
sys_which -Name git
#>
function sys_which {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )
    Get-Command -Name $Name |
        Select-Object Name, Source, CommandType
}

<#
.SYNOPSIS
Show recent Windows events.
.PARAMETER LogName
Event log name (default: Application).
.PARAMETER Top
Number of entries to show (default 50).
.EXAMPLE
sys_events
.EXAMPLE
sys_events -LogName System -Top 30
#>
function sys_events {
    param(
        [string]$LogName = "Application",
        [int]$Top = 50
    )

    Get-WinEvent -LogName $LogName -MaxEvents $Top |
        Select-Object TimeCreated, Id, LevelDisplayName, ProviderName, Message
}
