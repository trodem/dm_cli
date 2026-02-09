Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

<#
.SYNOPSIS
Get saved Wi-Fi profile names.
.DESCRIPTION
Parses `netsh wlan show profiles` and returns profile names.
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

<#
.SYNOPSIS
Show system uptime.
.DESCRIPTION
Prints elapsed time since last OS boot.
.EXAMPLE
dm s_uptime
#>
function s_uptime {
    $os = Get-CimInstance Win32_OperatingSystem
    $boot = $os.LastBootUpTime
    $uptime = (Get-Date) - $boot
    "{0}d {1}h {2}m" -f $uptime.Days, $uptime.Hours, $uptime.Minutes
}

<#
.SYNOPSIS
Show operating system info.
.DESCRIPTION
Prints Windows caption, version, build, and architecture.
.EXAMPLE
dm s_os
#>
function s_os {
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
dm s_path
#>
function s_path {
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
dm s_top_cpu -Count 20
#>
function s_top_cpu {
    param(
        [int]$Count = 15
    )
    Get-Process |
        Select-Object Name, Id, @{
            Name       = "CPUSeconds"
            Expression = {
                if ($null -eq $_.CPU) {
                    return 0.0
                }
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
Number of processes to show.
.EXAMPLE
dm s_top_mem -Count 20
#>
function s_top_mem {
    param(
        [int]$Count = 15
    )
    Get-Process |
        Select-Object Name, Id, @{
            Name       = "RAM_MB"
            Expression = {
                if ($null -eq $_.WorkingSet64) {
                    return 0.0
                }
                return [math]::Round([double]$_.WorkingSet64 / 1MB, 1)
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
dm s_svc -Name Spooler
#>
function s_svc {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )
    Get-Service -Name $Name | Select-Object Name, DisplayName, Status, StartType
}

<#
.SYNOPSIS
Restart a Windows service.
.DESCRIPTION
Restarts a service and optionally asks for confirmation.
.PARAMETER Name
Service name.
.PARAMETER Confirm
Skip prompt when provided.
.EXAMPLE
dm s_svc_restart -Name Spooler -Confirm
#>
function s_svc_restart {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [switch]$Confirm
    )
    if (-not $Confirm) {
        $answer = Read-Host "Restart service '$Name'? (y/N)"
        if ($answer -notin @("y", "Y", "yes", "YES")) {
            Write-Host "Canceled."
            return
        }
    }
    Restart-Service -Name $Name -ErrorAction Stop
    Get-Service -Name $Name | Select-Object Name, Status
}

<#
.SYNOPSIS
Show local IP addresses.
.DESCRIPTION
Lists active IPv4 addresses per interface.
.EXAMPLE
dm s_ip
#>
function s_ip {
    Get-NetIPAddress -AddressFamily IPv4 | Where-Object { $_.IPAddress -ne "127.0.0.1" } | Select-Object InterfaceAlias, IPAddress, PrefixLength
}

<#
.SYNOPSIS
Show listening TCP ports.
.DESCRIPTION
Lists listening TCP ports with owning process ID.
.EXAMPLE
dm s_ports
#>
function s_ports {
    Get-NetTCPConnection -State Listen | Select-Object LocalAddress, LocalPort, OwningProcess | Sort-Object LocalPort
}

<#
.SYNOPSIS
Test DNS resolution.
.DESCRIPTION
Resolves host name and prints DNS result.
.PARAMETER Host
Host name to resolve.
.EXAMPLE
dm s_dns -Host openai.com
#>
function s_dns {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Host
    )
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
dm s_ping -Host 8.8.8.8 -Count 4
#>
function s_ping {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Host,
        [int]$Count = 4
    )
    Test-Connection -ComputerName $Host -Count $Count
}

<#
.SYNOPSIS
Show disk usage.
.DESCRIPTION
Lists local disks with size and free space.
.EXAMPLE
dm s_disk
#>
function s_disk {
    Get-PSDrive -PSProvider FileSystem | Select-Object Name, @{Name = "UsedGB"; Expression = { [math]::Round(($_.Used / 1GB), 2) }}, @{Name = "FreeGB"; Expression = { [math]::Round(($_.Free / 1GB), 2) }}
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
dm s_big -Path . -Top 20
#>
function s_big {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path,
        [int]$Top = 20
    )
    _assert_path_exists -Path $Path
    Get-ChildItem -Path $Path -Recurse -File -ErrorAction SilentlyContinue | Sort-Object Length -Descending | Select-Object -First $Top FullName, @{Name = "SizeMB"; Expression = { [math]::Round($_.Length / 1MB, 2) }}
}

<#
.SYNOPSIS
Show total directory size.
.DESCRIPTION
Computes recursive sum of file sizes in a directory.
.PARAMETER Path
Directory path.
.EXAMPLE
dm s_size -Path .
#>
function s_size {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )
    _assert_path_exists -Path $Path
    $sum = (Get-ChildItem -Path $Path -Recurse -File -ErrorAction SilentlyContinue | Measure-Object -Property Length -Sum).Sum
    if ($null -eq $sum) { $sum = 0 }
    [pscustomobject]@{
        Path   = (Resolve-Path $Path).Path
        Bytes  = $sum
        SizeMB = [math]::Round($sum / 1MB, 2)
        SizeGB = [math]::Round($sum / 1GB, 2)
    }
}

<#
.SYNOPSIS
Find a command in PATH.
.DESCRIPTION
Shows command resolution path.
.PARAMETER Name
Command name.
.EXAMPLE
dm s_which -Name git
#>
function s_which {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )
    Get-Command -Name $Name | Select-Object Name, Source, CommandType
}

<#
.SYNOPSIS
Show Git version.
.DESCRIPTION
Runs `git --version`.
.EXAMPLE
dm s_v_git
#>
function s_v_git {
    _assert_command_available -Name git
    git --version
}

<#
.SYNOPSIS
Show Go version.
.DESCRIPTION
Runs `go version`.
.EXAMPLE
dm s_v_go
#>
function s_v_go {
    _assert_command_available -Name go
    go version
}

<#
.SYNOPSIS
Show Node.js version.
.DESCRIPTION
Runs `node --version`.
.EXAMPLE
dm s_v_node
#>
function s_v_node {
    _assert_command_available -Name node
    node --version
}

<#
.SYNOPSIS
Show recent Windows events.
.DESCRIPTION
Reads recent events from Application/System log.
.PARAMETER LogName
Event log name.
.PARAMETER Top
Number of entries to show.
.EXAMPLE
dm s_events -LogName System -Top 30
#>
function s_events {
    param(
        [string]$LogName = "Application",
        [int]$Top = 50
    )
    Get-WinEvent -LogName $LogName -MaxEvents $Top | Select-Object TimeCreated, Id, LevelDisplayName, ProviderName, Message
}

<#
.SYNOPSIS
Show network connection summary.
.DESCRIPTION
Groups TCP connections by state.
.EXAMPLE
dm s_net
#>
function s_net {
    Get-NetTCPConnection | Group-Object -Property State | Sort-Object Name | Select-Object Name, Count
}

<#
.SYNOPSIS
List saved Wi-Fi profiles.
.DESCRIPTION
Shows all saved Wi-Fi profile names.
.EXAMPLE
dm w_list
#>
function w_list {
    _wifi_profiles
}

<#
.SYNOPSIS
Show current Wi-Fi connection.
.DESCRIPTION
Prints current SSID/state/interface from `netsh wlan show interfaces`.
.EXAMPLE
dm w_cur
#>
function w_cur {
    _assert_command_available -Name netsh
    netsh wlan show interfaces
}

<#
.SYNOPSIS
Connect to a saved Wi-Fi profile.
.DESCRIPTION
Runs `netsh wlan connect name=<profile>`.
.PARAMETER Name
Saved Wi-Fi profile name.
.EXAMPLE
dm w_conn -Name HomeWifi
#>
function w_conn {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )
    _assert_command_available -Name netsh
    netsh wlan connect name="$Name"
    netsh wlan show interfaces
}

<#
.SYNOPSIS
Pick a Wi-Fi profile interactively.
.DESCRIPTION
Shows indexed profile list, asks for selection, and connects.
.EXAMPLE
dm w_pick
#>
function w_pick {
    $profiles = @(_wifi_profiles)
    if ($profiles.Count -eq 0) {
        Write-Host "No saved Wi-Fi profiles found."
        return
    }
    for ($i = 0; $i -lt $profiles.Count; $i++) {
        "{0}. {1}" -f ($i + 1), $profiles[$i]
    }
    $raw = Read-Host "Select profile number"
    $index = 0
    if (-not [int]::TryParse($raw, [ref]$index)) {
        Write-Host "Invalid selection."
        return
    }
    if ($index -lt 1 -or $index -gt $profiles.Count) {
        Write-Host "Selection out of range."
        return
    }
    $name = $profiles[$index - 1]
    netsh wlan connect name="$name"
    netsh wlan show interfaces
}

<#
.SYNOPSIS
Disconnect current Wi-Fi.
.DESCRIPTION
Runs `netsh wlan disconnect`.
.EXAMPLE
dm w_off
#>
function w_off {
    _assert_command_available -Name netsh
    netsh wlan disconnect
    netsh wlan show interfaces
}

<#
.SYNOPSIS
Show Wi-Fi signal details.
.DESCRIPTION
Displays signal and radio details from current Wi-Fi interface.
.EXAMPLE
dm w_sig
#>
function w_sig {
    _assert_command_available -Name netsh
    $lines = netsh wlan show interfaces
    $lines | Select-String -Pattern "^\s*(Name|Description|State|SSID|Signal|Radio type|Receive rate|Transmit rate)\s*:"
}
