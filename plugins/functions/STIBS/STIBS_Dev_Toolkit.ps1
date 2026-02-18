# =============================================================================
# STIBS DOCKER TOOLKIT – Container orchestration layer
# Generic helpers for docker-compose based STIBS development stack
# Safe operations only (no destructive actions)
# Entry point: dm stibs_docker_*
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

function _dc {
    $composeFile = _stibs_compose_file
    docker compose -f $composeFile @args
}

# ------------------------------------------------------------
# STACK
# ------------------------------------------------------------

<#
.SYNOPSIS
Start the STIBS docker development stack.
.DESCRIPTION
Runs docker compose up -d using docker-compose.dev.yml.
Ensures containers are started in detached mode.
.EXAMPLE
dm stibs_docker_up
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
Runs docker compose down using docker-compose.dev.yml.
Does not remove volumes or images.
.EXAMPLE
dm stibs_docker_down
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
dm stibs_docker_status
#>
function stibs_docker_status {
    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile
    _dc ps
}

# ------------------------------------------------------------
# LOGS
# ------------------------------------------------------------

<#
.SYNOPSIS
Show logs for a specific service.
.DESCRIPTION
Prints the last 50 lines of logs for the selected container.
.EXAMPLE
dm stibs_docker_logs backend
.EXAMPLE
dm stibs_docker_logs mariadb
#>
function stibs_docker_logs {
    param(
        [Parameter(Mandatory)]
        [ValidateSet("backend","frontend","mariadb","redis")]
        [string]$Service
    )

    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile
    _dc logs --tail 50 $Service
}

# ------------------------------------------------------------
# RESTART
# ------------------------------------------------------------

<#
.SYNOPSIS
Restart a specific service container.
.DESCRIPTION
Runs docker compose restart for the selected service.
.EXAMPLE
dm stibs_docker_restart backend
#>
function stibs_docker_restart {
    param(
        [Parameter(Mandatory)]
        [ValidateSet("backend","frontend","mariadb","redis")]
        [string]$Service
    )

    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile
    _dc restart $Service
}

# ------------------------------------------------------------
# BUILD
# ------------------------------------------------------------

<#
.SYNOPSIS
Build docker image for a specific service.
.DESCRIPTION
Runs docker compose build for the selected service.
Does not start containers.
.EXAMPLE
dm stibs_docker_build backend
#>
function stibs_docker_build {
    param(
        [Parameter(Mandatory)]
        [ValidateSet("backend","frontend")]
        [string]$Service
    )

    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile

    Write-Host "Building image for $Service..."
    _dc build $Service
}

<#
.SYNOPSIS
Build docker images for all services.
.DESCRIPTION
Runs docker compose build for the entire stack.
.EXAMPLE
dm stibs_docker_build_all
#>
function stibs_docker_build_all {
    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile

    Write-Host "Building all services..."
    _dc build
}

# ------------------------------------------------------------
# EXEC
# ------------------------------------------------------------

<#
.SYNOPSIS
Execute a command inside a service container.
.DESCRIPTION
Runs docker compose exec for the selected service with a custom command.
Used for migrations, tests, scripts, and tooling.
.EXAMPLE
dm stibs_docker_exec backend npm run test
.EXAMPLE
dm stibs_docker_exec backend npm run migration:run
.EXAMPLE
dm stibs_docker_exec frontend ng generate component user-card
#>
function stibs_docker_exec {
    param(
        [Parameter(Mandatory)]
        [ValidateSet("backend","frontend","mariadb","redis")]
        [string]$Service,

        [Parameter(Mandatory, ValueFromRemainingArguments=$true)]
        [string[]]$Command
    )

    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile

    _dc exec $Service @Command
}

# ------------------------------------------------------------
# SHELL
# ------------------------------------------------------------

<#
.SYNOPSIS
Open interactive shell inside a service container.
.DESCRIPTION
Runs docker compose exec <service> sh.
Used for manual debugging and inspection.
.EXAMPLE
dm stibs_docker_shell backend
#>
function stibs_docker_shell {
    param(
        [Parameter(Mandatory)]
        [ValidateSet("backend","frontend","mariadb","redis")]
        [string]$Service
    )

    _assert_command_available -Name docker
    $composeFile = _stibs_compose_file
    _assert_path_exists -Path $composeFile

    _dc exec $Service sh
}
