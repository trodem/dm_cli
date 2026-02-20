# =============================================================================
# KVP STAR – SharePoint site toolkit (standalone)
# Read-only operations for the KVP STAR SharePoint site.
# Safety: Read-only — no write or delete operations.
# Entry point: kvpstar_*
#
# FUNCTIONS
#   kvpstar_context
#   kvpstar_lists
#   kvpstar_items
#   kvpstar_list_details
#   kvpstar_list_columns
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$script:KVPSTAR_URL = "https://stadlerrailag.sharepoint.com/teams/kvpstar"

# -----------------------------------------------------------------------------
# Internal helpers — m365 CLI plumbing
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Ensure a command is available in PATH.
.PARAMETER Name
Command name to validate.
.EXAMPLE
_assert_command_available -Name m365
#>
function _assert_command_available {
    param([Parameter(Mandatory = $true)][string]$Name)
    if (-not (Get-Command -Name $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' was not found in PATH."
    }
}

<#
.SYNOPSIS
Ensures the m365 CLI is available in PATH.
.EXAMPLE
_kvpstar_assert_cli
#>
function _kvpstar_assert_cli {
    _assert_command_available -Name m365
}

<#
.SYNOPSIS
Ensures the user is authenticated in m365 CLI.
.DESCRIPTION
Throws if m365 status returns a non-zero exit code.
.EXAMPLE
_kvpstar_assert_login
#>
function _kvpstar_assert_login {
    _kvpstar_assert_cli
    m365 status 1>$null 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "Not authenticated in m365 CLI. Run 'm365 login' first."
    }
}

<#
.SYNOPSIS
Executes an m365 CLI command and returns parsed JSON.
.PARAMETER Command
Command string without the leading 'm365' (e.g. "spo site list").
.EXAMPLE
_kvpstar_invoke -Command "spo list list --webUrl ""https://..."" "
#>
function _kvpstar_invoke {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Command
    )

    _kvpstar_assert_login

    $raw = Invoke-Expression "m365 $Command --output json"

    if (-not $raw) { return $null }

    try   { return ($raw | ConvertFrom-Json) }
    catch { throw "Failed to parse m365 CLI output as JSON." }
}

# -----------------------------------------------------------------------------
# Internal helpers — SharePoint operations
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Lists all SharePoint lists for the KVP STAR site.
.EXAMPLE
_kvpstar_spo_lists
#>
function _kvpstar_spo_lists {
    _kvpstar_invoke -Command "spo list list --webUrl ""$script:KVPSTAR_URL"""
}

<#
.SYNOPSIS
Lists items of a SharePoint list by title.
.PARAMETER ListTitle
Title of the list.
.EXAMPLE
_kvpstar_spo_items -ListTitle "Tasks"
#>
function _kvpstar_spo_items {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ListTitle
    )

    _kvpstar_invoke -Command "spo listitem list --webUrl ""$script:KVPSTAR_URL"" --title ""$ListTitle"""
}

<#
.SYNOPSIS
Resolves a GenericList by partial or full title.
.DESCRIPTION
Searches non-hidden GenericLists (BaseTemplate 100) matching the given title.
Throws if zero or multiple matches are found.
.PARAMETER ListTitle
Partial or full list title.
.EXAMPLE
_kvpstar_resolve_list -ListTitle "Tasks"
#>
function _kvpstar_resolve_list {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ListTitle
    )

    $lists = _kvpstar_spo_lists

    $matches = $lists | Where-Object {
        $_.Hidden -eq $false -and
        $_.BaseTemplate -eq 100 -and
        $_.Title -like "*$ListTitle*"
    }

    if (-not $matches) {
        throw "No GenericList matching '$ListTitle' found."
    }

    if ($matches.Count -gt 1) {
        $names = $matches | Select-Object -ExpandProperty Title
        throw "Multiple lists match '$ListTitle': $($names -join ', ')"
    }

    return $matches[0]
}

<#
.SYNOPSIS
Returns user-defined fields of a resolved list.
.DESCRIPTION
Fetches all fields via m365 CLI and filters out hidden, read-only, and base-type fields.
.PARAMETER List
A resolved list object (from _kvpstar_resolve_list).
.EXAMPLE
_kvpstar_get_user_fields -List $list
#>
function _kvpstar_get_user_fields {
    param(
        [Parameter(Mandatory = $true)]
        $List
    )

    $cmd = "spo field list --webUrl ""$script:KVPSTAR_URL"" --listTitle ""$($List.Title)"""
    $fields = _kvpstar_invoke -Command $cmd

    return $fields | Where-Object {
        $_.Hidden -eq $false -and
        $_.ReadOnlyField -eq $false -and
        $_.FromBaseType -eq $false
    }
}

# -----------------------------------------------------------------------------
# Public functions
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Returns the target site context for this toolkit.
.EXAMPLE
kvpstar_context
#>
function kvpstar_context {
    return [pscustomobject]@{
        Toolkit = "KVP STAR"
        SiteUrl = $script:KVPSTAR_URL
    }
}

<#
.SYNOPSIS
Lists all visible GenericLists of KVP STAR site.
.EXAMPLE
kvpstar_lists
#>
function kvpstar_lists {

    $lists = _kvpstar_spo_lists

    $genericLists = $lists | Where-Object {
        $_.Hidden -eq $false -and
        $_.BaseTemplate -eq 100
    }

    return $genericLists | Select-Object Id, Title
}

<#
.SYNOPSIS
Lists items of a specific list in KVP STAR site.
.PARAMETER ListTitle
Title of the list.
.EXAMPLE
kvpstar_items -ListTitle "Tasks"
#>
function kvpstar_items {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ListTitle
    )

    $items = _kvpstar_spo_items -ListTitle $ListTitle

    return $items | Select-Object Id, Title
}

<#
.SYNOPSIS
Returns detailed information about a single GenericList.
.PARAMETER ListTitle
Partial or full list title.
.EXAMPLE
kvpstar_list_details -ListTitle "Tasks"
#>
function kvpstar_list_details {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ListTitle
    )

    $list = _kvpstar_resolve_list -ListTitle $ListTitle
    $userFields = _kvpstar_get_user_fields -List $list

    return [pscustomobject]@{
        Id            = $list.Id
        Title         = $list.Title
        Description   = $list.Description
        ItemCount     = $list.ItemCount
        Created       = $list.Created
        LastModified  = $list.LastItemModifiedDate
        Versioning    = $list.EnableVersioning
        MinorVersions = $list.EnableMinorVersions
        Columns       = ($userFields | Select-Object Title, InternalName, TypeAsString, Required)
    }
}

<#
.SYNOPSIS
Returns all user-defined columns of a GenericList.
.PARAMETER ListTitle
Partial or full list title.
.EXAMPLE
kvpstar_list_columns -ListTitle "Tasks"
#>
function kvpstar_list_columns {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ListTitle
    )

    $list = _kvpstar_resolve_list -ListTitle $ListTitle
    $userFields = _kvpstar_get_user_fields -List $list

    return $userFields | Select-Object Title, InternalName, TypeAsString, Required
}
