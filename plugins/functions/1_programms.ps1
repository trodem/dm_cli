Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

<#
.SYNOPSIS
Show running docker containers.
.DESCRIPTION
Runs `docker ps` in the current shell.
.EXAMPLE
dm dps
#>
function dps {
    _assert_command_available -Name docker
    docker ps
}

<#
.SYNOPSIS
Start HeidiSQL.
.DESCRIPTION
Opens the installed HeidiSQL executable.
.EXAMPLE
dm heidi
#>
function heidi {
    $heidi = "C:\Program Files\HeidiSQL\heidisql.exe"
    _assert_path_exists -Path $heidi
    Start-Process $heidi
}

<#
.SYNOPSIS
Start Docker Desktop.
.DESCRIPTION
Opens Docker Desktop using the default installation path.
.EXAMPLE
dm run_docker_desktop
#>
function run_docker_desktop {
    $dockerDesktop = "C:\Program Files\Docker\Docker\Docker Desktop.exe"
    _assert_path_exists -Path $dockerDesktop
    Start-Process $dockerDesktop
}

<#
.SYNOPSIS
Print a docker readiness message.
.DESCRIPTION
Outputs a status line used as placeholder helper.
.EXAMPLE
dm get_name
#>
function get_name {
    Write-Host ">> Attendo che Docker sia pronto..."
}
