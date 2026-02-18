# =============================================================================
# START DEV TOOLKIT – Local Application Launch Helpers
# Production-safe helpers to start local development applications
# Non-destructive defaults, deterministic behavior, no admin requirements
# Entry point: start_*
#
# FUNCTIONS
#   start_heididb
#   start_docker_desktop
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

<#
.SYNOPSIS
Ensures a filesystem path exists.
.DESCRIPTION
Validates that the specified file or directory exists.
Throws a terminating error if not found.
.PARAMETER Path
Filesystem path to validate.
.EXAMPLE
_assert_path_exists -Path "C:\Program Files\App\app.exe"
#>
function _assert_path_exists {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    if (-not (Test-Path -Path $Path)) {
        throw "Required path '$Path' does not exist."
    }
}

<#
.SYNOPSIS
Start HeidiSQL.
.DESCRIPTION
Opens the installed HeidiSQL executable using the default installation path.
.EXAMPLE
start_heididb
#>
function start_heididb {
    $heidi = "C:\Program Files\HeidiSQL\heidisql.exe"
    _assert_path_exists -Path $heidi
    Start-Process -FilePath $heidi
}

<#
.SYNOPSIS
Start Docker Desktop.
.DESCRIPTION
Opens Docker Desktop using the default installation path.
.EXAMPLE
start_docker_desktop
#>
function start_docker_desktop {
    $exe = "C:\Program Files\Docker\Docker\Docker Desktop.exe"
    _assert_path_exists -Path $exe
    Start-Process -FilePath $exe
}
