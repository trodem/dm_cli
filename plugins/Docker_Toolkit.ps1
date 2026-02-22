# =============================================================================
# DOCKER TOOLKIT – Docker Compose orchestration layer (standalone)
# Manage any docker-compose development stack from the current directory.
# Compose file resolved from DM_DOCKER_COMPOSE_FILE or auto-discovered in CWD.
# Safety: Non-destructive defaults. Down stops containers but keeps volumes.
# Entry point: dc_*
#
# FUNCTIONS
#   dc_ps
#   dc_file
#   dc_services
#   dc_up
#   dc_down
#   dc_status
#   dc_logs
#   dc_restart
#   dc_build
#   dc_build_all
#   dc_exec
#   dc_shell
#   dc_kill
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
Resolve the docker-compose file to use.
.DESCRIPTION
Checks DM_DOCKER_COMPOSE_FILE environment variable first.
Falls back to auto-discovery in the current directory, trying common
compose file names in priority order.
.EXAMPLE
_dc_resolve_compose_file
#>
function _dc_resolve_compose_file {
    $envFile = [Environment]::GetEnvironmentVariable("DM_DOCKER_COMPOSE_FILE")
    if (-not [string]::IsNullOrWhiteSpace($envFile)) {
        _assert_path_exists -Path $envFile
        return $envFile
    }

    $candidates = @(
        "docker-compose.yml",
        "docker-compose.yaml",
        "compose.yml",
        "compose.yaml",
        "docker-compose.dev.yml",
        "docker-compose.dev.yaml"
    )

    foreach ($name in $candidates) {
        $path = Join-Path (Get-Location).Path $name
        if (Test-Path -LiteralPath $path) {
            return $path
        }
    }

    throw "No docker-compose file found in '$((Get-Location).Path)'. Set DM_DOCKER_COMPOSE_FILE or place a compose file in the current directory."
}

<#
.SYNOPSIS
Run docker compose against the resolved compose file.
.DESCRIPTION
Prefixes every docker compose call with the resolved compose file path.
Passes all remaining arguments through.
.EXAMPLE
_dc up -d
#>
function _dc {
    $composeFile = _dc_resolve_compose_file
    docker compose -f $composeFile @args
}

<#
.SYNOPSIS
Ask user for confirmation before a destructive action.
.PARAMETER Message
Prompt text to display.
.EXAMPLE
_confirm_action -Message "Stop all containers?"
#>
function _confirm_action {
    param([Parameter(Mandatory = $true)][string]$Message)
    $answer = Read-Host "$Message [y/N]"
    if ($answer -notin @("y", "Y", "yes", "Yes")) {
        throw "Canceled by user."
    }
}

# -----------------------------------------------------------------------------
# Info
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
List all running Docker containers on the system.
.DESCRIPTION
Runs docker ps to show every active container regardless of compose file
or project. Useful as a quick global health check.
.EXAMPLE
dc_ps
#>
function dc_ps {
    _assert_command_available -Name docker
    docker ps
}

<#
.SYNOPSIS
Show the compose file used in the current directory.
.DESCRIPTION
Displays which compose file will be used based on the current
environment variable or auto-discovery in the current directory.
.EXAMPLE
dc_file
#>
function dc_file {
    _assert_command_available -Name docker
    $file = _dc_resolve_compose_file

    return [pscustomobject]@{
        ComposeFile = $file
    }
}

<#
.SYNOPSIS
List services defined in the current directory compose file.
.DESCRIPTION
Parses the compose file in the current directory and returns all service names.
.EXAMPLE
dc_services
#>
function dc_services {
    _assert_command_available -Name docker
    _dc config --services
}

# -----------------------------------------------------------------------------
# Stack
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Start the docker-compose stack in the current directory.
.DESCRIPTION
Runs docker compose up -d using the compose file found in the current directory.
Containers are started in detached mode.
.EXAMPLE
dc_up
#>
function dc_up {
    _assert_command_available -Name docker
    _dc up -d
}

<#
.SYNOPSIS
Stop the docker-compose stack in the current directory.
.DESCRIPTION
Runs docker compose down for the current directory stack. Does not remove volumes or images.
.EXAMPLE
dc_down
#>
function dc_down {
    _assert_command_available -Name docker
    _dc down
}

<#
.SYNOPSIS
Show container status for the current directory compose stack.
.DESCRIPTION
Displays docker compose ps for the compose file in the current directory.
.EXAMPLE
dc_status
#>
function dc_status {
    _assert_command_available -Name docker
    _dc ps
}

# -----------------------------------------------------------------------------
# Logs
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Show logs for a service in the current directory compose stack.
.DESCRIPTION
Prints the last 50 lines of logs for the selected container
from the compose file in the current directory.
.PARAMETER Service
Service name as defined in the compose file.
.EXAMPLE
dc_logs -Service backend
.EXAMPLE
dc_logs -Service mariadb
#>
function dc_logs {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Service
    )

    _assert_command_available -Name docker
    _dc logs --tail 50 $Service
}

# -----------------------------------------------------------------------------
# Restart
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Restart a service in the current directory compose stack.
.DESCRIPTION
Runs docker compose restart for the selected service
from the compose file in the current directory.
.PARAMETER Service
Service name as defined in the compose file.
.EXAMPLE
dc_restart -Service backend
#>
function dc_restart {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Service
    )

    _assert_command_available -Name docker
    _dc restart $Service
}

# -----------------------------------------------------------------------------
# Build
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Build docker image for a service in the current directory compose stack.
.DESCRIPTION
Runs docker compose build for the selected service using the compose file
in the current directory. Does not start containers.
.PARAMETER Service
Service name as defined in the compose file.
.EXAMPLE
dc_build -Service backend
#>
function dc_build {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Service
    )

    _assert_command_available -Name docker
    _dc build $Service
}

<#
.SYNOPSIS
Build all docker images for the current directory compose stack.
.DESCRIPTION
Runs docker compose build for the entire stack in the current directory.
.EXAMPLE
dc_build_all
#>
function dc_build_all {
    _assert_command_available -Name docker
    _dc build
}

# -----------------------------------------------------------------------------
# Exec
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Execute a command in a service of the current directory compose stack.
.DESCRIPTION
Runs docker compose exec for the selected service with a custom command
using the compose file in the current directory.
.PARAMETER Service
Service name as defined in the compose file.
.PARAMETER Command
Command and arguments to execute.
.EXAMPLE
dc_exec -Service backend npm run test
.EXAMPLE
dc_exec -Service frontend ng generate component user-card
#>
function dc_exec {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Service,

        [Parameter(Mandatory = $true, ValueFromRemainingArguments = $true)]
        [string[]]$Command
    )

    _assert_command_available -Name docker
    _dc exec $Service @Command
}

# -----------------------------------------------------------------------------
# Shell
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Open shell in a service of the current directory compose stack.
.DESCRIPTION
Runs docker compose exec <service> sh for manual debugging using
the compose file in the current directory.
.PARAMETER Service
Service name as defined in the compose file.
.EXAMPLE
dc_shell -Service backend
#>
function dc_shell {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Service
    )

    _assert_command_available -Name docker
    _dc exec $Service sh
}

# -----------------------------------------------------------------------------
# Kill
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Stop and remove all running Docker containers on the system.
.DESCRIPTION
Lists every running container, stops them, then removes them.
Requires -Force or interactive confirmation because it affects
ALL containers, not just the current compose stack.
Volumes and images are NOT removed.
.PARAMETER Force
Skip confirmation prompt.
.EXAMPLE
dc_kill
.EXAMPLE
dc_kill -Force
#>
function dc_kill {
    param(
        [switch]$Force
    )

    _assert_command_available -Name docker

    $ids = docker ps -q
    if ($null -eq $ids -or ($ids | Measure-Object).Count -eq 0) {
        return [pscustomobject]@{
            Status     = "clean"
            Message    = "No running containers found."
            Stopped    = 0
        }
    }

    $containers = docker ps --format "{{.ID}}  {{.Names}}  {{.Image}}"
    $count = ($ids | Measure-Object).Count

    if (-not $Force) {
        Write-Host "Running containers ($count):"
        Write-Host $containers
        _confirm_action -Message "Stop and remove all $count containers?"
    }

    docker stop $ids | Out-Null
    docker rm $ids | Out-Null

    return [pscustomobject]@{
        Status     = "done"
        Message    = "Stopped and removed $count containers."
        Stopped    = $count
    }
}
