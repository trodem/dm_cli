# =============================================================================
# M365 CORE – Microsoft 365 CLI operational core
# Minimal, production-safe base wrappers for dm-cli
# Entry point: m365_core_*
#
# FUNCTIONS
#   m365_core_assert_cli
#   m365_core_assert_login
#   m365_core_status_json
#   m365_core_invoke
#	m365_spo_lists_list
#	m365_spo_items_list 
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

<#
.SYNOPSIS
Ensures Microsoft 365 CLI (m365) exists in PATH.
.DESCRIPTION
Throws a terminating error if 'm365' is not available.
.EXAMPLE
m365_core_assert_cli
#>
function m365_core_assert_cli {
    if (-not (Get-Command m365 -ErrorAction SilentlyContinue)) {
        throw "Required command 'm365' is not available in PATH."
    }
}

<#
.SYNOPSIS
Returns m365 status as JSON object.
.DESCRIPTION
Parses 'm365 status --output json' into a PowerShell object.
.EXAMPLE
m365_core_status_json
#>
function m365_core_status_json {
    m365_core_assert_cli
    $raw = m365 status --output json
    return ($raw | ConvertFrom-Json)
}

<#
.SYNOPSIS
Ensures user is authenticated in m365 CLI.
.DESCRIPTION
Throws if not logged in.
.EXAMPLE
m365_core_assert_login
#>
function m365_core_assert_login {

    m365_core_assert_cli

    m365 status 1>$null 2>$null

    if ($LASTEXITCODE -ne 0) {
        throw "Not authenticated in m365 CLI. Run 'm365 login' first."
    }
}



<#
.SYNOPSIS
Executes an m365 CLI command string.
.DESCRIPTION
Central execution wrapper. By default requires login; can be bypassed.
.PARAMETER Command
Command without leading 'm365'.
.PARAMETER NoLoginRequired
If set, does not enforce login (useful for 'm365 login' and 'm365 status').
.EXAMPLE
m365_core_invoke -Command "spo site list"
.EXAMPLE
m365_core_invoke -Command "status" -NoLoginRequired
#>
function m365_core_invoke {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Command,
        [switch]$NoLoginRequired
    )

    m365_core_assert_cli

    if (-not $NoLoginRequired) {
        m365_core_assert_login
    }

    # Force JSON output for deterministic parsing
    $fullCommand = "m365 $Command --output json"

    $raw = Invoke-Expression $fullCommand

    if (-not $raw) {
        return $null
    }

    try {
        return ($raw | ConvertFrom-Json)
    }
    catch {
        throw "Failed to parse m365 CLI output as JSON."
    }
}



<#
.SYNOPSIS
Lists SharePoint lists using current context.
.DESCRIPTION
Uses SiteUrl from m365_context. Requires context to be set.
.EXAMPLE
m365_spo_lists_list
#>
function m365_spo_lists_list {

    $ctx = m365_context_get
    if (-not $ctx.SiteUrl) {
        throw "No SiteUrl set in context. Use m365_context_set first."
    }

    m365_core_invoke -Command ("spo list list --webUrl """ + $ctx.SiteUrl + """")
}


<#
.SYNOPSIS
Lists items from a SharePoint list using current context.
.PARAMETER ListTitle
Title of the list.
.EXAMPLE
m365_spo_items_list -ListTitle "Tasks"
#>
function m365_spo_items_list {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ListTitle
    )

    $ctx = m365_context_get
    if (-not $ctx.SiteUrl) {
        throw "No SiteUrl set in context. Use m365_context_set first."
    }

    m365_core_invoke -Command (
        "spo listitem list --webUrl """ + $ctx.SiteUrl + """ --title """ + $ListTitle + """"
    )
}
