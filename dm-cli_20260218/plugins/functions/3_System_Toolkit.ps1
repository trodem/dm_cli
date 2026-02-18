# =============================================================================
# DM SYS TOOLKIT – Local System & Network operational layer
# Production-safe Windows system helpers for local environments
# Non-destructive defaults, deterministic behavior, no admin requirements
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
#   sys_pgrep
#   sys_pkill
#   sys_version_git
#   sys_version_go
#   sys_version_node
#   sys_which
#   sys_events
#	sys_wifi_dev
#	sys_wifi_switch
# =============================================================================


Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$devWifi = "IBS_Labor"
$stadlerwlan = "STADLERWLAN"

# =========================
# INTERNAL HELPERS
# =========================

<#
.SYNOPSIS
Ensures a command exists in PATH.
.DESCRIPTION
Validates that the specified executable is available on the system.
Throws a terminating error if not found.
.PARAMETER Name
Command name to validate.
#>
function _assert_command_available {
    param(
        [Parameter(Mandatory)]
        [string]$Name
    )

    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' is not available in PATH."
    }
}

<#
.SYNOPSIS
Ensures a path exists.
.DESCRIPTION
Validates that the specified path exists on disk.
Throws a terminating error if not found.
.PARAMETER Path
Path to validate.
#>
function _assert_path_exists {
    param(
        [Parameter(Mandatory)]
        [string]$Path
    )

    if (-not (Test-Path -Path $Path)) {
        throw "Path '$Path' does not exist."
    }
}

<#
.SYNOPSIS
Get saved Wi-Fi profile names.
.DESCRIPTION
Parses `netsh wlan show profiles` and returns profile names.
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

# =========================
# SYSTEM
# =========================

<#
.SYNOPSIS
Show system uptime.
.DESCRIPTION
Prints elapsed time since last OS boot.
.EXAMPLE
sys_uptime
#>
function sys_uptime {
    $os = Get-CimInstance Win32_OperatingSystem
    $boot = $os.LastBootUpTime
    $uptime = (Get-Date) - $boot
    "{0}d {1}h {2}m" -f $uptime.Days, $uptime.Hours, $uptime.Minutes
}

<#
.SYNOPSIS
Show operating system information.
.DESCRIPTION
Prints Windows caption, version, build, and architecture.
.EXAMPLE
sys_os
#>
function sys_os {
    $os = Get-CimInstance Win32_OperatingSystem
    [pscustomobject]@{
        Caption      = $os.Caption
        Version      = $os.Version
        Build        = $os.BuildNumber
        Architecture = $os.OSArchitecture
    } | Format-List
}

<#
.SYNOPSIS
Show PATH entries.
.DESCRIPTION
Prints PATH entries one per line.
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
Number of processes to show.
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
                    return [math]::Round($_.CPU.TotalSeconds,3)
                }
                return [math]::Round([double]$_.CPU,3)
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
Number of processes to show.
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
                return [math]::Round($_.WorkingSet64/1MB,1)
            }
        } |
        Sort-Object RAM_MB -Descending |
        Select-Object -First $Count
}

<#
.SYNOPSIS
Show Windows service status.
.DESCRIPTION
Prints service status by service name.
.PARAMETER Name
Service name.
.EXAMPLE
sys_service_status -Name Spooler
#>
function sys_service_status {
    param([Parameter(Mandatory)][string]$Name)

    Get-Service -Name $Name |
        Select-Object Name, DisplayName, Status, StartType
}

<#
.SYNOPSIS
Restart a Windows service.
.DESCRIPTION
Restarts a service and optionally asks for confirmation.
.PARAMETER Name
Service name.
.PARAMETER Confirm
Skip confirmation prompt when provided.
.EXAMPLE
sys_service_restart -Name Spooler -Confirm
#>
function sys_service_restart {
    param(
        [Parameter(Mandatory)][string]$Name,
        [switch]$Confirm
    )

    if (-not $Confirm) {
        $answer = Read-Host "Restart service '$Name'? (y/N)"
        if ($answer -notin @("y","Y","yes","YES")) {
            Write-Host "Canceled."
            return
        }
    }

    Restart-Service -Name $Name -ErrorAction Stop
    Get-Service -Name $Name | Select-Object Name, Status
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
        @{Name="UsedGB";Expression={[math]::Round($_.Used/1GB,2)}},
        @{Name="FreeGB";Expression={[math]::Round($_.Free/1GB,2)}}
}

<#
.SYNOPSIS
List largest files in a directory tree.
.DESCRIPTION
Finds biggest files recursively.
.PARAMETER Path
Base directory path.
.PARAMETER Top
Number of files to return.
.EXAMPLE
sys_big -Path . -Top 20
#>
function sys_big {
    param(
        [Parameter(Mandatory)][string]$Path,
        [int]$Top = 20
    )

    _assert_path_exists -Path $Path

    Get-ChildItem -Path $Path -Recurse -File -ErrorAction SilentlyContinue |
        Sort-Object Length -Descending |
        Select-Object -First $Top FullName,
        @{Name="SizeMB";Expression={[math]::Round($_.Length/1MB,2)}}
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
    param([Parameter(Mandatory)][string]$Path)

    _assert_path_exists -Path $Path

    $sum = (Get-ChildItem -Path $Path -Recurse -File -ErrorAction SilentlyContinue |
        Measure-Object Length -Sum).Sum

    if ($null -eq $sum) { $sum = 0 }

    [pscustomobject]@{
        Path   = (Resolve-Path $Path).Path
        Bytes  = $sum
        SizeMB = [math]::Round($sum/1MB,2)
        SizeGB = [math]::Round($sum/1GB,2)
    }
}

<#
.SYNOPSIS
Show current timestamp.
.DESCRIPTION
Prints current date and time in ISO-like format.
.EXAMPLE
sys_now
#>
function sys_now {
    Get-Date -Format "yyyy-MM-dd HH:mm:ss"
}

<#
.SYNOPSIS
Write timestamped log message.
.DESCRIPTION
Prints a message prefixed with current timestamp.
.PARAMETER Message
Message to log.
.EXAMPLE
sys_log -Message "Deployment started"
#>
function sys_log {
    param([Parameter(Mandatory)][string]$Message)
    Write-Host "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') | $Message"
}

<#
.SYNOPSIS
Copy text to clipboard.
.DESCRIPTION
Copies provided text to system clipboard.
.PARAMETER Text
Text to copy.
.EXAMPLE
sys_clip -Text "hello"
#>
function sys_clip {
    param([Parameter(Mandatory)][string]$Text)
    $Text | Set-Clipboard
}

<#
.SYNOPSIS
Read clipboard content.
.DESCRIPTION
Returns current clipboard text.
.EXAMPLE
sys_paste
#>
function sys_paste {
    Get-Clipboard
}

# =========================
# NETWORK
# =========================

<#
.SYNOPSIS
Show local IPv4 addresses.
.DESCRIPTION
Lists active IPv4 addresses per interface.
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
Lists listening TCP ports with owning process ID.
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
.DESCRIPTION
Resolves host name and prints DNS result.
.PARAMETER Host
Host name to resolve.
.EXAMPLE
sys_dns -Host openai.com
#>
function sys_dns {
    param([Parameter(Mandatory)][string]$Host)
    Resolve-DnsName -Name $Host
}

<#
.SYNOPSIS
Ping a host.
.DESCRIPTION
Runs ICMP echo requests.
.PARAMETER Host
Host to ping.
.PARAMETER Count
Number of packets.
.EXAMPLE
sys_ping -Host 8.8.8.8 -Count 4
#>
function sys_ping {
    param(
        [Parameter(Mandatory)][string]$Host,
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
.DESCRIPTION
Verifies TCP connectivity to remote host and port.
.PARAMETER ComputerName
Remote host.
.PARAMETER Port
Remote port.
.EXAMPLE
sys_test_port -ComputerName google.com -Port 443
#>
function sys_test_port {
    param(
        [Parameter(Mandatory)][string]$ComputerName,
        [Parameter(Mandatory)][int]$Port
    )
    Test-NetConnection -ComputerName $ComputerName -Port $Port
}

# =========================
# WIFI
# =========================

<#
.SYNOPSIS
List saved Wi-Fi profiles.
.DESCRIPTION
Shows all saved Wi-Fi profile names.
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
Prints current SSID, state and interface details.
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
Connects to specified saved Wi-Fi profile.
.PARAMETER Name
Saved Wi-Fi profile name.
.EXAMPLE
sys_wifi_connect -Name HomeWifi
#>
function sys_wifi_connect {
    param([Parameter(Mandatory)][string]$Name)

    _assert_command_available -Name netsh
    netsh wlan connect name="$Name"
    netsh wlan show interfaces
}

<#
.SYNOPSIS
Interactively pick a Wi-Fi profile.
.DESCRIPTION
Displays indexed list and connects to selected profile.
.EXAMPLE
sys_wifi_pick
#>
function sys_wifi_pick {
    $profiles = @(_wifi_profiles)

    if ($profiles.Count -eq 0) {
        Write-Host "No saved Wi-Fi profiles found."
        return
    }

    for ($i=0; $i -lt $profiles.Count; $i++) {
        "{0}. {1}" -f ($i+1), $profiles[$i]
    }

    $raw = Read-Host "Select profile number"
    $index = 0

    if (-not [int]::TryParse($raw,[ref]$index)) {
        Write-Host "Invalid selection."
        return
    }

    if ($index -lt 1 -or $index -gt $profiles.Count) {
        Write-Host "Selection out of range."
        return
    }

    $name = $profiles[$index-1]
    netsh wlan connect name="$name"
    netsh wlan show interfaces
}

<#
.SYNOPSIS
Disconnect current Wi-Fi.
.DESCRIPTION
Disconnects from active Wi-Fi network.
.EXAMPLE
sys_wifi_disconnect
#>
function sys_wifi_disconnect {
    _assert_command_available -Name netsh
    netsh wlan disconnect
    netsh wlan show interfaces
}

<#
.SYNOPSIS
Show Wi-Fi signal details.
.DESCRIPTION
Displays signal strength and radio details.
.EXAMPLE
sys_wifi_signal
#>
function sys_wifi_signal {
    _assert_command_available -Name netsh
    $lines = netsh wlan show interfaces
    $lines | Select-String -Pattern "^\s*(Name|Description|State|SSID|Signal|Radio type|Receive rate|Transmit rate)\s*:"
}

# =========================
# PROCESS
# =========================

<#
.SYNOPSIS
Find processes by name.
.DESCRIPTION
Searches active processes by partial name match.
.PARAMETER Name
Process name fragment.
.EXAMPLE
sys_pgrep -Name chrome
#>
function sys_pgrep {
    param([Parameter(Mandatory)][string]$Name)

    Get-Process *$Name* |
        Select-Object Name, Id, CPU, WorkingSet
}

<#
.SYNOPSIS
Terminate processes by name.
.DESCRIPTION
Stops processes matching provided name.
.PARAMETER Name
Process name fragment.
.EXAMPLE
sys_pkill -Name chrome
#>
function sys_pkill {
    param([Parameter(Mandatory)][string]$Name)

    Get-Process *$Name* -ErrorAction SilentlyContinue |
        Stop-Process -Force
}

<#
.SYNOPSIS
Show Git version.
.DESCRIPTION
Runs `git --version`.
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
.DESCRIPTION
Runs `go version`.
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
.DESCRIPTION
Runs `node --version`.
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
.DESCRIPTION
Shows command resolution path.
.PARAMETER Name
Command name.
.EXAMPLE
sys_which -Name git
#>
function sys_which {
    param([Parameter(Mandatory)][string]$Name)
    Get-Command -Name $Name |
        Select-Object Name, Source, CommandType
}

<#
.SYNOPSIS
Show recent Windows events.
.DESCRIPTION
Reads recent events from Application or System log.
.PARAMETER LogName
Event log name.
.PARAMETER Top
Number of entries to show.
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


<#
.SYNOPSIS
Invoke sys_wifi_dev.
.DESCRIPTION
Helper/command function for sys_wifi_dev.
.EXAMPLE
dm sys_wifi_dev
#>
function sys_wifi_dev {
    param(
        [Parameter(Position = 0)]
        [string]$ProfileName = $devWifi
    )

    # Check if netsh exists
    if (-not (Get-Command netsh -ErrorAction SilentlyContinue)) {
        throw "netsh command not available."
    }

    # Check if profile exists
    $profiles = netsh wlan show profiles

    if (-not ($profiles -match $ProfileName)) {
        throw "Wi-Fi profile '$ProfileName' not found."
    }

    # Connect
    netsh wlan connect name="$ProfileName" | Out-Null

    Start-Sleep -Seconds 2

    # Check connection status
    $status = netsh wlan show interfaces

    [pscustomobject]@{
        Action      = "Connect"
        ProfileName = $ProfileName
        Status      = if ($status -match "State\s*:\s*connected") { "Connected" } else { "Unknown" }
    }
}

<#
.SYNOPSIS
Invoke sys_wifi_stadlerwlan.
.DESCRIPTION
Helper/command function for sys_wifi_stadlerwlan.
.EXAMPLE
dm sys_wifi_stadlerwlan
#>
function sys_wifi_stadlerwlan {
    param(
        [Parameter(Position = 0)]
        [string]$ProfileName = $stadlerwlan
    )

    # Check if netsh exists
    if (-not (Get-Command netsh -ErrorAction SilentlyContinue)) {
        throw "netsh command not available."
    }

    # Check if profile exists
    $profiles = netsh wlan show profiles

    if (-not ($profiles -match $ProfileName)) {
        throw "Wi-Fi profile '$ProfileName' not found."
    }

    # Connect
    netsh wlan connect name="$ProfileName" | Out-Null

    Start-Sleep -Seconds 2

    # Check connection status
    $status = netsh wlan show interfaces

    [pscustomobject]@{
        Action      = "Connect"
        ProfileName = $ProfileName
        Status      = if ($status -match "State\s*:\s*connected") { "Connected" } else { "Unknown" }
    }
}


<#
.SYNOPSIS
Invoke sys_wifi_switch.
.DESCRIPTION
Helper/command function for sys_wifi_switch.
.EXAMPLE
dm sys_wifi_switch
#>
function sys_wifi_switch {
    param(
        [Parameter(Mandatory = $true, Position = 0)]
        [string]$ProfileName,

        [Parameter(Position = 1)]
        [int]$TimeoutSeconds = 15
    )

    # Validate netsh availability
    if (-not (Get-Command netsh -ErrorAction SilentlyContinue)) {
        throw "netsh command not available."
    }

    # Check profile existence
    $profiles = netsh wlan show profiles

    if (-not ($profiles -match $ProfileName)) {
        throw "Wi-Fi profile '$ProfileName' not found."
    }

    # Disconnect first (clean switch)
    netsh wlan disconnect | Out-Null
    Start-Sleep -Seconds 2

    # Attempt connection
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

    [pscustomobject]@{
        Action      = "Switch"
        ProfileName = $ProfileName
        Status      = "Connected"
        TimeoutUsed = $elapsed
    }
}
