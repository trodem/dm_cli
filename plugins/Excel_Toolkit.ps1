# =============================================================================
# EXCEL TOOLKIT – Auto-generated toolkit (standalone)
# Safety: Review generated functions before use.
# Entry point: ods_count_*
#
# FUNCTIONS
#   ods_count_sheets
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal helpers
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
_assert_path_exists -Path C:\Data
#>
function _assert_path_exists {
    param([Parameter(Mandatory = $true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Required path '$Path' does not exist."
    }
}

# -----------------------------------------------------------------------------
# Public functions
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Counts the number of sheets in an ODS file.
.DESCRIPTION
This function takes the file path of an ODS file as input and returns the number of sheets it contains.
.PARAMETER FilePath
The full path to the ODS file.
.EXAMPLE
ods_count_sheets -FilePath "C:\Users\Demtro\Downloads\test_file.ods"
#>
function ods_count_sheets {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath
    )

    _assert_path_exists -Path $FilePath

    # Load the ODS file and count sheets
    try {
        $doc = [System.IO.Packaging.Package]::Open($FilePath, [System.IO.FileMode]::Open)
        $sheets = $doc.GetParts() | Where-Object { $_.Uri.OriginalString -like "*content.xml" }
        $sheetCount = $sheets.Count
        $doc.Close()
    } catch {
        throw "Failed to load ODS file: $_"
    }

    return [pscustomobject]@{
        SheetCount = $sheetCount
    }
}
