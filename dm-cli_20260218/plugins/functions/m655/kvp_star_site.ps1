# =============================================================================
# KVP STAR – Domain toolkit
# Site-specific wrapper built on top of m365_core + m365_context
# Target: https://stadlerrailag.sharepoint.com/teams/kvpstar
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

<#
.SYNOPSIS
Ensures KVP STAR context is set.
.DESCRIPTION
If no SiteUrl is present in context, sets it automatically.
.EXAMPLE
kvpstar_context
#>
function kvpstar_context {

    $ctx = m365_context_get

    if (-not $ctx.SiteUrl) {
        m365_context_set -SiteUrl $script:KVPSTAR_URL
    }
}

<#
.SYNOPSIS
Lists all SharePoint lists of KVP STAR site.
.EXAMPLE
kvpstar_lists
#>
function kvpstar_lists {

    kvpstar_context

    $lists = m365_spo_lists_list

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

    kvpstar_context

    $items = m365_spo_items_list -ListTitle $ListTitle

    return $items | Select-Object Id, Title
}

<#
.SYNOPSIS
Returns detailed information about a single GenericList.
.PARAMETER ListTitle
Title of the list.
.EXAMPLE
kvpstar_list_details -ListTitle "Tasks"
#>
function kvpstar_list_details {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ListTitle
    )

    kvpstar_context

    $lists = m365_spo_lists_list

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

    $list = $matches[0]

    $fields = m365_core_invoke -Command (
        "spo field list --webUrl """ + (m365_context_get).SiteUrl + """ --listTitle """ + $list.Title + """"
    )

    $userFields = $fields | Where-Object {
        $_.Hidden -eq $false -and
        $_.ReadOnlyField -eq $false -and
        $_.FromBaseType -eq $false
    } | Select-Object Title, InternalName, TypeAsString, Required

    Write-Host ""
    Write-Host "List Details" -ForegroundColor Cyan
    Write-Host "------------"
    Write-Host "Id: $($list.Id)"
    Write-Host "Title: $($list.Title)"
    Write-Host "Description: $($list.Description)"
    Write-Host "ItemCount: $($list.ItemCount)"
    Write-Host "Created: $($list.Created)"
    Write-Host "Last Modified: $($list.LastItemModifiedDate)"
    Write-Host "Versioning: $($list.EnableVersioning)"
    Write-Host "Minor Versions: $($list.EnableMinorVersions)"
    Write-Host ""

    Write-Host "Columns" -ForegroundColor Cyan
    Write-Host "-------"

    $userFields | Format-Table `
        Title,
        InternalName,
        TypeAsString,
        Required -AutoSize
}

<#
.SYNOPSIS
Returns all user-defined columns of a GenericList.
.PARAMETER ListTitle
Partial or full list title (Smart Mode).
.EXAMPLE
kvpstar_list_columns -ListTitle "Tasks"
#>
function kvpstar_list_columns {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ListTitle
    )

    kvpstar_context

    $lists = m365_spo_lists_list

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

    $list = $matches[0]

    $fields = m365_core_invoke -Command (
        "spo field list --webUrl """ + (m365_context_get).SiteUrl + """ --listTitle """ + $list.Title + """"
    )

    $userFields = $fields | Where-Object {
        $_.Hidden -eq $false -and
        $_.ReadOnlyField -eq $false -and
        $_.FromBaseType -eq $false
    }

    return $userFields | Select-Object `
        Title,
        InternalName,
        TypeAsString,
        Required
}
