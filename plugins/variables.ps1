Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

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

function _stibs_docker_path {
    return $script:DM_STIBS_DOCKER_PATH
}

function _stibs_compose_file {
    return Join-Path (_stibs_docker_path) "docker-compose.dev.yml"
}

function _stibs_db_config {
    return [pscustomobject]@{
        Container = $script:DM_STIBS_DB_CONTAINER
        User      = $script:DM_STIBS_DB_USER
        Password  = $script:DM_STIBS_DB_PASSWORD
        Database  = $script:DM_STIBS_DB_NAME
    }
}
