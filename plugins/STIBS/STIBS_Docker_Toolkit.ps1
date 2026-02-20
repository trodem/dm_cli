# =============================================================================
# STIBS DOCKER TOOLKIT – Container orchestration layer (standalone)
# Manage the docker-compose STIBS development stack.
# Safety: Non-destructive defaults. Down stops containers but keeps volumes.
# Entry point: stibs_docker_*
#
# FUNCTIONS
#   stibs_docker_up
#   stibs_docker_down
#   stibs_docker_status
#   stibs_docker_logs
#   stibs_docker_restart
#   stibs_docker_build
#   stibs_docker_build_all
#   stibs_docker_exec
#   stibs_docker_shell
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal helpers — guards and config
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
Read an environment variable with a fallback default.
.PARAMETER Name
Environment variable name.
.PARAMETER Default
Value to return if the variable is unset or empty.
.EXAMPLE
_env_or_default -Name "DM_STIBS_DOCKER_PATH" -Default "C:\docker"
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
Return the STIBS docker-compose.dev.yml file path.
.DESCRIPTION
Builds the path from the DM_STIBS_DOCKER_PATH environment variable.
.EXAMPLE
_stibs_compose_file
#>
function _stibs_compose_file {
    $dockerPath = _env_or_default `
        -Name "DM_STIBS_DOCKER_PATH" `
        -Default "C:\Users\trodem\Documents\01_Development\50_STIBS\stibs-mono\stibs\docker"
    return Join-Path $dockerPath "docker-compose.dev.yml"
}

<#
.SYNOPSIS
Run docker compose against the STIBS dev compose file.
.DESCRIPTION
Prefixes every docker compose call with the STIBS compose file path.
Passes all remaining arguments through.
.EXAMPLE
_dc up -d
#>
function _dc {
    $composeFile = _stibs_compose_file
    docker compose -f $composeFile @args
}

# -----------------------------------------------------------------------------
# Stack
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Start the STIBS docker development stack.
.DESCRIPTION
Runs docker compose up -d using docker-compose.dev.yml.
Ensures containers are started in detached mode.
.EXAMPLE
stibs_docker_up
#>
function stibs_docker_up {
    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile
    _dc up -d
}

<#
.SYNOPSIS
Stop the STIBS docker development stack.
.DESCRIPTION
Runs docker compose down. Does not remove volumes or images.
.EXAMPLE
stibs_docker_down
#>
function stibs_docker_down {
    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile
    _dc down
}

<#
.SYNOPSIS
Show status of running containers.
.DESCRIPTION
Displays docker compose ps for the STIBS development stack.
.EXAMPLE
stibs_docker_status
#>
function stibs_docker_status {
    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile
    _dc ps
}

# -----------------------------------------------------------------------------
# Logs
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Show logs for a specific service.
.DESCRIPTION
Prints the last 50 lines of logs for the selected container.
.PARAMETER Service
Service name to show logs for.
.EXAMPLE
stibs_docker_logs -Service backend
.EXAMPLE
stibs_docker_logs -Service mariadb
#>
function stibs_docker_logs {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet("backend","frontend","mariadb","redis")]
        [string]$Service
    )

    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile
    _dc logs --tail 50 $Service
}

# -----------------------------------------------------------------------------
# Restart
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Restart a specific service container.
.DESCRIPTION
Runs docker compose restart for the selected service.
.PARAMETER Service
Service name to restart.
.EXAMPLE
stibs_docker_restart -Service backend
#>
function stibs_docker_restart {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet("backend","frontend","mariadb","redis")]
        [string]$Service
    )

    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile
    _dc restart $Service
}

# -----------------------------------------------------------------------------
# Build
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Build docker image for a specific service.
.DESCRIPTION
Runs docker compose build for the selected service. Does not start containers.
.PARAMETER Service
Service name to build.
.EXAMPLE
stibs_docker_build -Service backend
#>
function stibs_docker_build {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet("backend","frontend")]
        [string]$Service
    )

    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile
    _dc build $Service
}

<#
.SYNOPSIS
Build docker images for all services.
.DESCRIPTION
Runs docker compose build for the entire stack.
.EXAMPLE
stibs_docker_build_all
#>
function stibs_docker_build_all {
    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile
    _dc build
}

# -----------------------------------------------------------------------------
# Exec
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Execute a command inside a service container.
.DESCRIPTION
Runs docker compose exec for the selected service with a custom command.
Used for migrations, tests, scripts, and tooling.
.PARAMETER Service
Service name to exec into.
.PARAMETER Command
Command and arguments to execute.
.EXAMPLE
stibs_docker_exec -Service backend npm run test
.EXAMPLE
stibs_docker_exec -Service frontend ng generate component user-card
#>
function stibs_docker_exec {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet("backend","frontend","mariadb","redis")]
        [string]$Service,

        [Parameter(Mandatory = $true, ValueFromRemainingArguments = $true)]
        [string[]]$Command
    )

    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile
    _dc exec $Service @Command
}

# -----------------------------------------------------------------------------
# Shell
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Open interactive shell inside a service container.
.DESCRIPTION
Runs docker compose exec <service> sh for manual debugging and inspection.
.PARAMETER Service
Service name to open a shell into.
.EXAMPLE
stibs_docker_shell -Service backend
#>
function stibs_docker_shell {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet("backend","frontend","mariadb","redis")]
        [string]$Service
    )

    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile
    _dc exec $Service sh
}
