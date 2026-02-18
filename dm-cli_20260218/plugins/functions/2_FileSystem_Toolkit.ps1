# =============================================================================
# DM FILESYSTEM PATH TOOLKIT – Windows Special Paths Layer
# Production-safe helpers for resolving and navigating known system paths
# Non-destructive defaults, deterministic behavior, no admin requirements
# Entry point: fs_path_*
#
# FUNCTIONS
#   fs_path_list
#   fs_path_show
#   fs_path_open
#   fs_path_pick
#   fs_path_cd
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

<#
.SYNOPSIS
Get known special paths map.
.DESCRIPTION
Builds a hashtable of common user and system paths on Windows.
.EXAMPLE
_fs_path_map
#>
function _fs_path_map {
    $userStartup = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\Startup"
    $allStartup  = Join-Path $env:ProgramData "Microsoft\Windows\Start Menu\Programs\Startup"
    $psProfileDir = Join-Path $env:USERPROFILE "Documents\PowerShell"

    return @{
        appdata                 = $env:APPDATA
        localappdata            = $env:LOCALAPPDATA
        programdata             = $env:ProgramData
        temp                    = $env:TEMP
        userprofile             = $env:USERPROFILE
        desktop                 = [Environment]::GetFolderPath("Desktop")
        documents               = [Environment]::GetFolderPath("MyDocuments")
        downloads               = Join-Path $env:USERPROFILE "Downloads"
        pictures                = [Environment]::GetFolderPath("MyPictures")
        startup_user            = $userStartup
        startup_all             = $allStartup
        powershell_profile_dir  = $psProfileDir
    }
}

<#
.SYNOPSIS
List known special paths.
.DESCRIPTION
Prints key and resolved path for each known location.
.EXAMPLE
fs_path_list
#>
function fs_path_list {
    $map = _fs_path_map
    foreach ($k in ($map.Keys | Sort-Object)) {
        "{0,-22} {1}" -f $k, $map[$k]
    }
}

<#
.SYNOPSIS
Show one known special path.
.DESCRIPTION
Prints the resolved path for a selected key.
.PARAMETER Name
Path key from `fs_path_list`.
.EXAMPLE
fs_path_show -Name appdata
#>
function fs_path_show {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $map = _fs_path_map
    $key = $Name.ToLowerInvariant()

    if (-not $map.ContainsKey($key)) {
        throw "Unknown path key '$Name'. Use fs_path_list."
    }

    $map[$key]
}

<#
.SYNOPSIS
Open a known special path.
.DESCRIPTION
Opens selected path in the system file browser.
.PARAMETER Name
Path key from `fs_path_list`.
.EXAMPLE
fs_path_open -Name localappdata
#>
function fs_path_open {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $map = _fs_path_map
    $key = $Name.ToLowerInvariant()

    if (-not $map.ContainsKey($key)) {
        throw "Unknown path key '$Name'. Use fs_path_list."
    }

    $path = $map[$key]

    if (-not (Test-Path -LiteralPath $path)) {
        throw "Path not found: $path"
    }

    Start-Process -FilePath explorer.exe -ArgumentList $path
}

<#
.SYNOPSIS
Interactively pick and open a special path.
.DESCRIPTION
Shows indexed list from `fs_path_list`, asks selection, then opens the selected path.
.EXAMPLE
fs_path_pick
#>
function fs_path_pick {
    $map  = _fs_path_map
    $keys = @($map.Keys | Sort-Object)

    if ($keys.Count -eq 0) {
        Write-Host "No paths configured."
        return
    }

    for ($i = 0; $i -lt $keys.Count; $i++) {
        "{0}. {1}" -f ($i + 1), $keys[$i]
    }

    $raw = Read-Host "Select path number"

    $idx = 0
    if (-not [int]::TryParse($raw, [ref]$idx)) {
        Write-Host "Invalid selection."
        return
    }

    if ($idx -lt 1 -or $idx -gt $keys.Count) {
        Write-Host "Selection out of range."
        return
    }

    $selected = $keys[$idx - 1]
    fs_path_open -Name $selected
}

<#
.SYNOPSIS
Change current location to a special path.
.DESCRIPTION
Resolves path key and executes `Set-Location`.
.PARAMETER Name
Path key from `fs_path_list`.
.EXAMPLE
fs_path_cd -Name documents
#>
function fs_path_cd {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $map = _fs_path_map
    $key = $Name.ToLowerInvariant()

    if (-not $map.ContainsKey($key)) {
        throw "Unknown path key '$Name'. Use fs_path_list."
    }

    $path = $map[$key]

    if (-not (Test-Path -LiteralPath $path)) {
        throw "Path not found: $path"
    }

    Set-Location -LiteralPath $path
    Get-Location
}
