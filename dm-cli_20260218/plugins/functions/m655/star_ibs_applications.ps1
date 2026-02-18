# =============================================================================
# STAR IBS APPLICATIONS – Domain toolkit
# Site-specific wrapper built on top of m365_core + m365_context
# Target: https://stadlerrailag.sharepoint.com/teams/staribsapplications
# Entry point: star_ibs_applications_*
#
# FUNCTIONS
#   star_ibs_applications_context
#   star_ibs_applications_lists
#   star_ibs_applications_items
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$script:STAR_IBS_APPLICATIONS_URL = "https://stadlerrailag.sharepoint.com/teams/staribsapplications"

<#
.SYNOPSIS
Ensures STAR IBS Applications context is set.
.DESCRIPTION
If no SiteUrl is present in context, sets it automatically.
.EXAMPLE
star_ibs_applications_context
#>
function star_ibs_applications_context {

    $ctx = m365_context_get

    if (-not $ctx.SiteUrl) {
        m365_context_set -SiteUrl $script:STAR_IBS_APPLICATIONS_URL
    }
}

<#
.SYNOPSIS
Lists all SharePoint lists of STAR IBS Applications site.
.EXAMPLE
star_ibs_applications_lists
#>
function star_ibs_applications_lists {

    star_ibs_applications_context

    $lists = m365_spo_lists_list

    $genericLists = $lists | Where-Object {
        $_.Hidden -eq $false -and
        $_.BaseTemplate -eq 100
    }

    return $genericLists | Select-Object Id, Title
}




<#
.SYNOPSIS
Lists items of a specific list in STAR IBS Applications site.
.PARAMETER ListTitle
Title of the list.
.EXAMPLE
star_ibs_applications_items -ListTitle "Tasks"
#>
function star_ibs_applications_items {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ListTitle
    )

    star_ibs_applications_context

    $items = m365_spo_items_list -ListTitle $ListTitle

    return $items | Select-Object Id, Title
}

<#
.SYNOPSIS
Returns detailed information about a single GenericList.
.PARAMETER ListTitle
Title of the list.
.EXAMPLE
star_ibs_applications_list_details -ListTitle "Tasks"
#>
function star_ibs_applications_list_details {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ListTitle
    )

    star_ibs_applications_context

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

    # Retrieve user columns
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
star_ibs_applications_list_columns -ListTitle "Tasks"
#>
function star_ibs_applications_list_columns {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ListTitle
    )

    star_ibs_applications_context

    # Smart match for list
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

    # Retrieve fields
    $fields = m365_core_invoke -Command (
        "spo field list --webUrl """ + (m365_context_get).SiteUrl + """ --listTitle """ + $list.Title + """"
    )

    # Filter only user columns
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
