Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

param(
    [Parameter(Mandatory = $true)]
    [string]$FunctionName,
    [string]$ProfilePathsJson = "[]",
    [string]$ArgsJson = "[]"
)

<#
.SYNOPSIS
Invoke _to_array.
.DESCRIPTION
Helper/command function for _to_array.
.EXAMPLE
dm _to_array
#>
function _to_array {
    param([object]$Value)
    if ($null -eq $Value) { return @() }
    if ($Value -is [System.Array]) { return $Value }
    return @($Value)
}

$profilePaths = @()
if (-not [string]::IsNullOrWhiteSpace($ProfilePathsJson)) {
    try {
        $profilePaths = _to_array (ConvertFrom-Json -InputObject $ProfilePathsJson -ErrorAction Stop)
    } catch {
        throw "Invalid ProfilePathsJson payload."
    }
}

$invokeArgs = @()
if (-not [string]::IsNullOrWhiteSpace($ArgsJson)) {
    try {
        $invokeArgs = _to_array (ConvertFrom-Json -InputObject $ArgsJson -ErrorAction Stop)
    } catch {
        throw "Invalid ArgsJson payload."
    }
}

foreach ($profilePath in $profilePaths) {
    if (-not [string]::IsNullOrWhiteSpace([string]$profilePath) -and (Test-Path -LiteralPath $profilePath)) {
        . $profilePath
    }
}

if (-not (Get-Command -Name $FunctionName -CommandType Function -ErrorAction SilentlyContinue)) {
    throw "Function '$FunctionName' was not loaded from plugin sources."
}

& $FunctionName @invokeArgs
