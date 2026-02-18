# =============================================================================
# DM HELP TOOLKIT – AI Introspection & Discovery Layer
# Runtime inspection helpers for DM toolkits
# Non-destructive, read-only, deterministic behavior
# Entry point: help_*
#
# FUNCTIONS
#   help_list_all
#   help_list_prefix
#   help_toolkit_map
#   help_function
#   help_parameters
#   help_examples
#   help_where
#   help_source
#   help_exists
#   help_count
#   help_export_index
#   help_builtin_list
#   help_builtin_info
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# =========================
# INTERNAL HELPERS
# =========================

<#
.SYNOPSIS
Returns all toolkit functions.
.DESCRIPTION
Returns all functions following toolkit naming convention (prefix_action).
#>
function _help_get_toolkit_functions {
    Get-Command -CommandType Function |
        Where-Object { $_.Name -match "^[a-zA-Z0-9]+_" }
}

<#
.SYNOPSIS
Extracts prefix from function name.
.DESCRIPTION
Returns prefix portion before first underscore.
.PARAMETER Name
Target function name.
#>
function _help_get_prefix {
    param(
        [Parameter(Mandatory)]
        [string]$Name
    )

    return ($Name -split "_")[0]
}

# =========================
# DISCOVERY
# =========================

<#
.SYNOPSIS
List all toolkit functions.
.DESCRIPTION
Returns all loaded toolkit functions sorted by name.
.EXAMPLE
help_list_all
#>
function help_list_all {
    _help_get_toolkit_functions |
        Sort-Object Name |
        Select-Object Name
}

<#
.SYNOPSIS
List toolkit functions by prefix.
.DESCRIPTION
Filters toolkit functions by prefix.
.PARAMETER Prefix
Toolkit prefix.
.EXAMPLE
help_list_prefix -Prefix sys
#>
function help_list_prefix {
    param(
        [Parameter(Mandatory)]
        [string]$Prefix
    )

    _help_get_toolkit_functions |
        Where-Object { $_.Name -like "$Prefix*" } |
        Sort-Object Name |
        Select-Object Name
}

<#
.SYNOPSIS
Group toolkit functions by prefix.
.DESCRIPTION
Returns grouped view of toolkit functions by prefix.
.EXAMPLE
help_toolkit_map
#>
function help_toolkit_map {
    _help_get_toolkit_functions |
        Group-Object { _help_get_prefix $_.Name } |
        Sort-Object Name |
        Select-Object Name, Count
}

<#
.SYNOPSIS
Count toolkit functions.
.DESCRIPTION
Returns total number of toolkit functions loaded.
.EXAMPLE
help_count
#>
function help_count {
    (_help_get_toolkit_functions).Count
}

<#
.SYNOPSIS
Check if toolkit function exists.
.DESCRIPTION
Returns true if function exists and follows toolkit naming convention.
.PARAMETER Name
Target function name.
.EXAMPLE
help_exists -Name sys_uptime
#>
function help_exists {
    param(
        [Parameter(Mandatory)]
        [string]$Name
    )

    $cmd = Get-Command -Name $Name -CommandType Function -ErrorAction SilentlyContinue

    if (-not $cmd) { return $false }
    if ($Name -notmatch "^[a-zA-Z0-9]+_") { return $false }

    return $true
}

# =========================
# DOCUMENTATION
# =========================

<#
.SYNOPSIS
Show help for a toolkit function.
.DESCRIPTION
Displays comment-based help for specified toolkit function.
.PARAMETER Name
Target function name.
.EXAMPLE
help_function -Name sys_uptime
#>
function help_function {
    param(
        [Parameter(Mandatory)]
        [string]$Name
    )

    if (-not (help_exists -Name $Name)) {
        throw "Function '$Name' not found in toolkit."
    }

    Get-Help $Name -Full
}

<#
.SYNOPSIS
Show parameters of a toolkit function.
.DESCRIPTION
Returns parameter metadata for specified function.
.PARAMETER Name
Target function name.
.EXAMPLE
help_parameters -Name sys_ping
#>
function help_parameters {
    param(
        [Parameter(Mandatory)]
        [string]$Name
    )

    if (-not (help_exists -Name $Name)) {
        throw "Function '$Name' not found in toolkit."
    }

    (Get-Command $Name).Parameters.Values |
        Select-Object Name, ParameterType, IsMandatory, Position
}

<#
.SYNOPSIS
Show examples of a toolkit function.
.DESCRIPTION
Displays example section from comment-based help.
.PARAMETER Name
Target function name.
.EXAMPLE
help_examples -Name sys_ping
#>
function help_examples {
    param(
        [Parameter(Mandatory)]
        [string]$Name
    )

    if (-not (help_exists -Name $Name)) {
        throw "Function '$Name' not found in toolkit."
    }

    Get-Help $Name -Examples
}

# =========================
# LOCATION & SOURCE
# =========================

<#
.SYNOPSIS
Show where a toolkit function is defined.
.DESCRIPTION
Returns source file or scriptblock information.
.PARAMETER Name
Target function name.
.EXAMPLE
help_where -Name sys_uptime
#>
function help_where {
    param(
        [Parameter(Mandatory)]
        [string]$Name
    )

    if (-not (help_exists -Name $Name)) {
        throw "Function '$Name' not found in toolkit."
    }

    $cmd = Get-Command $Name

    [pscustomobject]@{
        Name       = $cmd.Name
        CommandType= $cmd.CommandType
        Module     = $cmd.ModuleName
        Source     = $cmd.Source
        ScriptPath = $cmd.ScriptBlock.File
    }
}

<#
.SYNOPSIS
Return source code of a toolkit function.
.DESCRIPTION
Outputs the scriptblock definition text.
.PARAMETER Name
Target function name.
.EXAMPLE
help_source -Name sys_uptime
#>
function help_source {
    param(
        [Parameter(Mandatory)]
        [string]$Name
    )

    if (-not (help_exists -Name $Name)) {
        throw "Function '$Name' not found in toolkit."
    }

    (Get-Command $Name).ScriptBlock.ToString()
}

# =========================
# AI SUPPORT
# =========================

<#
.SYNOPSIS
Export structured toolkit index.
.DESCRIPTION
Returns structured metadata for all toolkit functions.
Useful for AI reasoning and dynamic discovery.
.EXAMPLE
help_export_index
#>
function help_export_index {

    _help_get_toolkit_functions |
        Sort-Object Name |
        ForEach-Object {
            $help = Get-Help $_.Name -ErrorAction SilentlyContinue

            [pscustomobject]@{
                Name        = $_.Name
                Prefix      = _help_get_prefix $_.Name
                Module      = $_.ModuleName
                Parameters  = ($_.Parameters.Keys -join ", ")
                Synopsis    = $help.Synopsis
                ScriptPath  = $_.ScriptBlock.File
            }
        }
}

# =========================
# BUILT-IN COMMANDS
# =========================

<#
.SYNOPSIS
List built-in PowerShell commands.
.DESCRIPTION
Returns built-in cmdlets, aliases and module functions.
.PARAMETER Name
Optional name filter.
.EXAMPLE
help_builtin_list
#>
function help_builtin_list {
    param(
        [string]$Name = "*"
    )

    Get-Command -Name $Name |
        Where-Object { $_.ModuleName -ne $null } |
        Sort-Object Name |
        Select-Object Name, CommandType, ModuleName
}

<#
.SYNOPSIS
Show detailed information about a built-in command.
.DESCRIPTION
Returns metadata and help content for specified built-in command.
.PARAMETER Name
Command name.
.EXAMPLE
help_builtin_info -Name Get-Process
#>
function help_builtin_info {
    param(
        [Parameter(Mandatory)]
        [string]$Name
    )

    $cmd = Get-Command -Name $Name -ErrorAction Stop

    if ($cmd.ModuleName -eq $null) {
        throw "Command '$Name' is not a built-in module command."
    }

    $help = Get-Help $Name -Full -ErrorAction SilentlyContinue

    [pscustomobject]@{
        Name        = $cmd.Name
        CommandType = $cmd.CommandType
        Module      = $cmd.ModuleName
        Version     = $cmd.Version
        Source      = $cmd.Source
        Parameters  = ($cmd.Parameters.Keys -join ", ")
        Synopsis    = $help.Synopsis
        Syntax      = ($help.Syntax | Out-String).Trim()
    }
}

