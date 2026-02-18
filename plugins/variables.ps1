Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

<#
.SYNOPSIS
Invoke _env_or_default.
.DESCRIPTION
Helper/command function for _env_or_default.
.EXAMPLE
dm _env_or_default
#>
function _env_or_default {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [Parameter(Mandatory = $true)]
        [string]$Default
    )

    $value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($value)) {
        return $Default
    }

    return $value
}

$script:DM_STIBS_DOCKER_PATH = _env_or_default `
    -Name "DM_STIBS_DOCKER_PATH" `
    -Default "C:\Users\trodem\Documents\01_Development\50_STIBS\stibs-mono\stibs\docker"

$script:DM_STIBS_DB_CONTAINER = _env_or_default `
    -Name "DM_STIBS_DB_CONTAINER" `
    -Default "docker-mariadb-1"

$script:DM_STIBS_DB_USER = _env_or_default `
    -Name "DM_STIBS_DB_USER" `
    -Default "stibs"

$script:DM_STIBS_DB_PASSWORD = _env_or_default `
    -Name "DM_STIBS_DB_PASSWORD" `
    -Default "stibs"

$script:DM_STIBS_DB_NAME = _env_or_default `
    -Name "DM_STIBS_DB_NAME" `
    -Default "stibs"

<#
.SYNOPSIS
Invoke _stibs_docker_path.
.DESCRIPTION
Helper/command function for _stibs_docker_path.
.EXAMPLE
dm _stibs_docker_path
#>
function _stibs_docker_path {
    return $script:DM_STIBS_DOCKER_PATH
}

<#
.SYNOPSIS
Invoke _stibs_compose_file.
.DESCRIPTION
Helper/command function for _stibs_compose_file.
.EXAMPLE
dm _stibs_compose_file
#>
function _stibs_compose_file {
    return Join-Path (_stibs_docker_path) "docker-compose.dev.yml"
}

<#
.SYNOPSIS
Invoke _stibs_db_config.
.DESCRIPTION
Helper/command function for _stibs_db_config.
.EXAMPLE
dm _stibs_db_config
#>
function _stibs_db_config {
    return [pscustomobject]@{
        Container = $script:DM_STIBS_DB_CONTAINER
        User      = $script:DM_STIBS_DB_USER
        Password  = $script:DM_STIBS_DB_PASSWORD
        Database  = $script:DM_STIBS_DB_NAME
    }
}
