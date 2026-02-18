# =============================================================================
# M365 CONTEXT – Environment and Site context manager
# Lightweight context storage for dm-cli
# Entry point: m365_context_*
#
# FUNCTIONS
#   m365_context_set
#   m365_context_get
#   m365_context_clear
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$script:M365_CONTEXT = @{
    SiteUrl = $null
	ReadOnly = $true
}

<#
.SYNOPSIS
Sets the active SharePoint SiteUrl context.
.DESCRIPTION
Stores the SiteUrl in memory for the current session.
.PARAMETER SiteUrl
Full SharePoint site URL.
.EXAMPLE
m365_context_set -SiteUrl "https://contoso.sharepoint.com/sites/dev"
#>
function m365_context_set {
    param(
        [Parameter(Mandatory = $true)]
        [string]$SiteUrl
    )

    if ($SiteUrl -notmatch "^https://") {
        throw "SiteUrl must be a valid HTTPS URL."
    }

    $script:M365_CONTEXT.SiteUrl = $SiteUrl.TrimEnd("/")
}

<#
.SYNOPSIS
Returns current context.
.DESCRIPTION
Returns stored SiteUrl.
.EXAMPLE
m365_context_get
#>
function m365_context_get {
    return [PSCustomObject]@{
        SiteUrl = $script:M365_CONTEXT.SiteUrl
    }
}

<#
.SYNOPSIS
Clears current context.
.DESCRIPTION
Resets stored SiteUrl.
.EXAMPLE
m365_context_clear
#>
function m365_context_clear {
    $script:M365_CONTEXT.SiteUrl = $null
}

<#
.SYNOPSIS
Enables read-only mode.
#>
function m365_context_readonly_enable {
    $script:M365_CONTEXT.ReadOnly = $true
}

<#
.SYNOPSIS
Disables read-only mode.
#>
function m365_context_readonly_disable {
    $script:M365_CONTEXT.ReadOnly = $false
}

<#
.SYNOPSIS
Ensures write operations are allowed.
#>
function m365_context_assert_writable {

    if ($script:M365_CONTEXT.ReadOnly -eq $true) {
        throw "Operation blocked. Toolkit is in read-only mode."
    }
}


<#
.SYNOPSIS
Requires explicit confirmation for destructive operations.
.PARAMETER Message
Confirmation message.
.EXAMPLE
m365_context_confirm_destructive -Message "Delete list X?"
#>
function m365_context_confirm_destructive {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Message
    )

    Write-Host ""
    Write-Host "DESTRUCTIVE OPERATION" -ForegroundColor Red
    Write-Host $Message -ForegroundColor Yellow
    Write-Host ""

    $confirmation = Read-Host "Type YES to continue"

    if ($confirmation -ne "YES") {
        throw "Operation cancelled by user."
    }
}
