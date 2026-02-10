Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"


<#
.SYNOPSIS
Ensure a command is available in the current shell.
.DESCRIPTION
Throws a descriptive error if the command is not found in PATH.
.PARAMETER Name
Command name to validate.
.EXAMPLE
Assert-CommandAvailable -Name docker
#>
function _assert_command_available {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    if (-not (Get-Command -Name $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' was not found in PATH."
    }
}

<#
.SYNOPSIS
Ensure a filesystem path exists.
.DESCRIPTION
Throws a descriptive error if the provided path does not exist.
.PARAMETER Path
Filesystem path to validate.
.EXAMPLE
Assert-PathExists -Path $varStibsMonoPath
#>
function _assert_path_exists {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Required path '$Path' does not exist."
    }
}
