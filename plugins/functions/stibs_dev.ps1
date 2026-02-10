Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$varStibsDockerPath = "C:\Users\trodem\Documents\01_Development\50_STIBS\stibs-mono\stibs\docker"
$varStibsMonoPath   = "C:\Users\trodem\Documents\01_Development\50_STIBS\stibs-mono"

$composeFile = Join-Path $varStibsDockerPath "docker-compose.dev.yml"

function _dc {
    docker compose -f $composeFile @args
}

<#
.SYNOPSIS
Show backend container logs.
.DESCRIPTION
Prints the last 50 lines from the backend service.
.EXAMPLE
dm stibs_logs_backend
#>
function stibs_logs_backend {
    _assert_command_available -Name docker
    _assert_path_exists -Path $composeFile
    _dc logs --tail 50 backend
}

<#
.SYNOPSIS
Show frontend container logs.
.DESCRIPTION
Prints the last 50 lines from the frontend service.
.EXAMPLE
dm stibs_logs_frontend
#>
function stibs_logs_frontend {
    _assert_command_available -Name docker
    _assert_path_exists -Path $composeFile
    _dc logs --tail 50 frontend
}

<#
.SYNOPSIS
Start STIBS development stack.
.DESCRIPTION
Runs docker compose up using docker-compose.dev.yml from $varStibsDockerPath.
.EXAMPLE
dm stibs_dev_up
#>
function stibs_dev_up {
    _assert_command_available -Name docker
    _assert_path_exists -Path $composeFile
    _dc up -d
}

<#
.SYNOPSIS
Stop STIBS development stack.
.DESCRIPTION
Runs docker compose down using docker-compose.dev.yml from $varStibsDockerPath.
.EXAMPLE
dm stibs_dev_down
#>
function stibs_dev_down {
    _assert_command_available -Name docker
    _assert_path_exists -Path $composeFile
    _dc down
}

<#
.SYNOPSIS
Show MariaDB connection command.
.DESCRIPTION
Prints the local DB connection command for the STIBS dev database.
.EXAMPLE
dm stibs_db
#>
function stibs_db {
    Write-Host "mariadb -h 127.0.0.1 -P 13306 -u stibs -pstibs stibs"
}

<#
.SYNOPSIS
Open a MariaDB shell in the dev container.
.DESCRIPTION
Runs docker compose exec mariadb with STIBS credentials.
.EXAMPLE
dm stibs_db_shell
#>
function stibs_db_shell {
    _assert_command_available -Name docker
    _assert_path_exists -Path $composeFile
    _dc exec mariadb mariadb -u stibs -pstibs stibs
}

<#
.SYNOPSIS
Restart backend container in the dev stack.
.DESCRIPTION
Runs docker compose restart backend using docker-compose.dev.yml.
.EXAMPLE
dm stibs_dev_restart_backend
#>
function stibs_dev_restart_backend {
    _assert_command_available -Name docker
    _assert_path_exists -Path $composeFile
    _dc restart backend
}

<#
.SYNOPSIS
Rebuild and start backend container in the dev stack.
.DESCRIPTION
Runs docker compose up -d --build backend using docker-compose.dev.yml.
.EXAMPLE
dm stibs_dev_rebuild_backend
#>
function stibs_dev_rebuild_backend {
    _assert_command_available -Name docker
    _assert_path_exists -Path $composeFile
    _dc up -d --build backend
}

<#
.SYNOPSIS
Show status of STIBS development stack.
.DESCRIPTION
Displays docker compose ps for the dev stack.
.EXAMPLE
dm stibs_status
#>
function stibs_status {
    _assert_command_available -Name docker
    _assert_path_exists -Path $composeFile
    _dc ps
}

<#
.SYNOPSIS
Change directory to STIBS local path.
.DESCRIPTION
Moves shell location to $varStibsMonoPath.
.EXAMPLE
dm stibs
#>
function stibs {
    _assert_path_exists -Path $varStibsMonoPath
    Set-Location $varStibsMonoPath
}
