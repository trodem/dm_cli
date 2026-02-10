Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

<#
.SYNOPSIS
Change directory to the configured development root.
.DESCRIPTION
Uses $varDevPath from plugins/variables.ps1 and moves the current shell location there.
.EXAMPLE
dm dev_path
#>
function dev_path {
    _assert_path_exists -Path $varDevPath
    Set-Location "$varDevPath"
}


<#
.SYNOPSIS
Change directory to the dm CLI repository.
.DESCRIPTION
Moves to the local knowledge/CLI repository rooted under $varSynologyDrivePath.
.EXAMPLE
dm dm_cli_path
#>
function dm_cli_path {
    $target = "$varSynologyDrivePath\5_Demtrodev_Knowledge"
    _assert_path_exists -Path $target
    Set-Location $target
}

