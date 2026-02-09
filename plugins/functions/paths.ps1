Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

<#
.SYNOPSIS
Get known special paths map.
.DESCRIPTION
Builds a hashtable of common user/system paths on Windows.
.EXAMPLE
_paths_map
#>
function _paths_map {
    $userStartup = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\Startup"
    $allStartup = Join-Path $env:ProgramData "Microsoft\Windows\Start Menu\Programs\Startup"
    $psProfileDir = Join-Path $env:USERPROFILE "Documents\PowerShell"

    return @{
        appdata              = $env:APPDATA
        localappdata         = $env:LOCALAPPDATA
        programdata          = $env:ProgramData
        temp                 = $env:TEMP
        userprofile          = $env:USERPROFILE
        desktop              = [Environment]::GetFolderPath("Desktop")
        documents            = [Environment]::GetFolderPath("MyDocuments")
        downloads            = (Join-Path $env:USERPROFILE "Downloads")
        pictures             = [Environment]::GetFolderPath("MyPictures")
        startup_user         = $userStartup
        startup_all          = $allStartup
        powershell_profile_dir = $psProfileDir
    }
}

<#
.SYNOPSIS
List known special paths.
.DESCRIPTION
Prints key and resolved path for each known location.
.EXAMPLE
dm p_list
#>
function p_list {
    $map = _paths_map
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
Path key from `p_list`.
.EXAMPLE
dm p_show -Name appdata
#>
function p_show {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $map = _paths_map
    $key = $Name.ToLowerInvariant()
    if (-not $map.ContainsKey($key)) {
        throw "Unknown path key '$Name'. Use p_list."
    }
    $map[$key]
}

<#
.SYNOPSIS
Open a known special path.
.DESCRIPTION
Opens selected path in system file browser.
.PARAMETER Name
Path key from `p_list`.
.EXAMPLE
dm p_open -Name localappdata
#>
function p_open {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $map = _paths_map
    $key = $Name.ToLowerInvariant()
    if (-not $map.ContainsKey($key)) {
        throw "Unknown path key '$Name'. Use p_list."
    }
    $path = $map[$key]
    if (-not (Test-Path -LiteralPath $path)) {
        throw "Path not found: $path"
    }
    Start-Process explorer.exe $path
}

<#
.SYNOPSIS
Interactively pick and open a special path.
.DESCRIPTION
Shows indexed list from `p_list`, asks selection, then opens path.
.EXAMPLE
dm p_pick
#>
function p_pick {
    $map = _paths_map
    $keys = @($map.Keys | Sort-Object)
    if ($keys.Count -eq 0) {
        Write-Host "No paths configured."
        return
    }

    for ($i = 0; $i -lt $keys.Count; $i++) {
        $k = $keys[$i]
        "{0}. {1}" -f ($i + 1), $k
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
    p_open -Name $selected
}

<#
.SYNOPSIS
Change current location to a special path.
.DESCRIPTION
Resolves path key and executes `Set-Location`.
.PARAMETER Name
Path key from `p_list`.
.EXAMPLE
dm p_cd -Name documents
#>
function p_cd {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $map = _paths_map
    $key = $Name.ToLowerInvariant()
    if (-not $map.ContainsKey($key)) {
        throw "Unknown path key '$Name'. Use p_list."
    }
    $path = $map[$key]
    if (-not (Test-Path -LiteralPath $path)) {
        throw "Path not found: $path"
    }
    Set-Location -LiteralPath $path
    Get-Location
}
