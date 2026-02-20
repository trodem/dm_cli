# =============================================================================
# START DEV TOOLKIT – Local application launch helpers (standalone)
# Launch development tools from their default or configured paths.
# Safety: Non-destructive — only starts processes, never stops or modifies.
# Entry point: start_*
#
# FUNCTIONS
#   start_heididb
#   start_docker_desktop
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal helpers
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Read an environment variable with a fallback default.
.PARAMETER Name
Environment variable name.
.PARAMETER Default
Value to return if the variable is unset or empty.
.EXAMPLE
_env_or_default -Name "DM_HEIDI_PATH" -Default "C:\Program Files\HeidiSQL\heidisql.exe"
#>
function _env_or_default {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Default
    )
    $value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($value)) { return $Default }
    return $value
}

<#
.SYNOPSIS
Ensure a filesystem path exists.
.PARAMETER Path
Path to validate.
.EXAMPLE
_assert_path_exists -Path "C:\Program Files\App\app.exe"
#>
function _assert_path_exists {
    param([Parameter(Mandatory = $true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Required path '$Path' does not exist."
    }
}

# -----------------------------------------------------------------------------
# Public functions
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Start HeidiSQL.
.DESCRIPTION
Opens HeidiSQL from the default or DM_HEIDI_PATH-configured location.
.EXAMPLE
start_heididb
#>
function start_heididb {
    $heidi = _env_or_default -Name "DM_HEIDI_PATH" -Default "C:\Program Files\HeidiSQL\heidisql.exe"
    _assert_path_exists -Path $heidi
    Start-Process -FilePath $heidi
}

<#
.SYNOPSIS
Start Docker Desktop.
.DESCRIPTION
Opens Docker Desktop from the default or DM_DOCKER_DESKTOP_PATH-configured location.
.EXAMPLE
start_docker_desktop
#>
function start_docker_desktop {
    $exe = _env_or_default -Name "DM_DOCKER_DESKTOP_PATH" -Default "C:\Program Files\Docker\Docker\Docker Desktop.exe"
    _assert_path_exists -Path $exe
    Start-Process -FilePath $exe
}
