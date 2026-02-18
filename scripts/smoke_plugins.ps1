param(
    [switch]$RunStibs,
    [switch]$Strict
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $repoRoot

function Test-Step {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][scriptblock]$Action,
        [switch]$Optional
    )

    Write-Host "[SMOKE] $Name"
    try {
        & $Action
        Write-Host "[OK] $Name" -ForegroundColor Green
        return $true
    }
    catch {
        Write-Host "[FAIL] $Name -> $($_.Exception.Message)" -ForegroundColor Red
        if (-not $Optional -or $Strict) {
            throw
        }
        return $false
    }
}

function Assert-FunctionExists {
    param([Parameter(Mandatory = $true)][string]$Name)
    if (-not (Get-Command -Name $Name -CommandType Function -ErrorAction SilentlyContinue)) {
        throw "Function '$Name' was not loaded."
    }
}

$allOk = $true

$allOk = (Test-Step -Name "Plugin help checker" -Action {
    go run ./scripts/check_plugin_help.go | Out-Host
}) -and $allOk

$allOk = (Test-Step -Name "Load plugin files and check key commands" -Action {
    . ./plugins/utils.ps1
    if (Test-Path -LiteralPath ./plugins/variables.ps1) {
        . ./plugins/variables.ps1
    }

    $pluginFiles = Get-ChildItem -Path ./plugins/functions -Recurse -File -Filter *.ps1 | Sort-Object FullName
    foreach ($file in $pluginFiles) {
        . $file.FullName
    }

    $required = @(
        "git_status",
        "notes_read",
        "translate_text",
        "stibs_db_status",
        "stibs_docker_status"
    )

    foreach ($name in $required) {
        Assert-FunctionExists -Name $name
    }
}) -and $allOk

if ($RunStibs) {
    $allOk = (Test-Step -Name "STIBS docker status smoke" -Action {
        _assert_command_available -Name docker
        stibs_docker_status | Out-Host
    } -Optional) -and $allOk

    $allOk = (Test-Step -Name "STIBS db status smoke" -Action {
        _assert_command_available -Name docker
        stibs_db_status | Out-Host
    } -Optional) -and $allOk
}

if (-not $allOk) {
    throw "Smoke checks completed with optional failures."
}

Write-Host "[DONE] smoke_plugins completed successfully." -ForegroundColor Cyan
