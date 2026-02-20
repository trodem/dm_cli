# =============================================================================
# BROWSER TOOLKIT – Local browser operational layer (standalone)
# Manage and launch the default browser.
# Safety: Non-destructive defaults. Close-all requires -Force or confirmation.
# Entry point: browser_*
#
# FUNCTIONS
#   browser_standard
#   browser_close_all
#   browser_open
#   browser_localhost
#   browser_wait_and_open
#   browser_open_many
#   browser_test_port
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal helpers
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Ask for yes/no confirmation before a risky action.
.PARAMETER Prompt
Message shown to the user.
.EXAMPLE
if (-not (_confirm_action -Prompt "Continue?")) { return }
#>
function _confirm_action {
    param([Parameter(Mandatory = $true)][string]$Prompt)
    $answer = Read-Host "$Prompt [y/N]"
    if ([string]::IsNullOrWhiteSpace($answer)) { return $false }
    return $answer.Trim().ToLowerInvariant() -in @("y", "yes")
}

<#
.SYNOPSIS
Resolve the default browser from the Windows registry.
.DESCRIPTION
Returns the process name and display name of the default browser.
Throws if the registry key is missing or the browser is unknown.
.EXAMPLE
_resolve_default_browser
#>
function _resolve_default_browser {
    $regPath = "HKCU:\Software\Microsoft\Windows\Shell\Associations\UrlAssociations\http\UserChoice"

    if (-not (Test-Path $regPath)) {
        throw "Cannot determine default browser (registry key not found)."
    }

    $progId = (Get-ItemProperty -Path $regPath).ProgId

    $browserMap = @{
        "ChromeHTML"  = @{ Process = "chrome";   Name = "Google Chrome" }
        "MSEdgeHTM"   = @{ Process = "msedge";   Name = "Microsoft Edge" }
        "FirefoxURL"  = @{ Process = "firefox";  Name = "Mozilla Firefox" }
        "OperaStable" = @{ Process = "opera";    Name = "Opera" }
        "BraveHTML"   = @{ Process = "brave";    Name = "Brave" }
        "SafariURL"   = @{ Process = "safari";   Name = "Safari" }
    }

    $entry = $browserMap[$progId]

    if (-not $entry) {
        throw "Unsupported or unknown default browser ($progId)."
    }

    return [pscustomobject]@{
        ProgId      = $progId
        ProcessName = $entry.Process
        BrowserName = $entry.Name
    }
}

# -----------------------------------------------------------------------------
# Info
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Show the default browser name and ProgId.
.DESCRIPTION
Reads the Windows registry to identify the current default browser.
.EXAMPLE
browser_standard
#>
function browser_standard {
    _resolve_default_browser
}

# -----------------------------------------------------------------------------
# Close
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Close all instances of the default browser.
.DESCRIPTION
Detects the default browser via registry and terminates all its processes.
Requires -Force or interactive confirmation.
.PARAMETER Force
Skip interactive confirmation.
.EXAMPLE
browser_close_all -Force
#>
function browser_close_all {
    param([switch]$Force)

    $browser = _resolve_default_browser
    $procs = Get-Process -Name $browser.ProcessName -ErrorAction SilentlyContinue

    if (-not $procs) {
        return [pscustomobject]@{
            Browser = $browser.BrowserName
            Closed  = 0
            Status  = "No running instances"
        }
    }

    if (-not $Force) {
        $count = $procs.Count
        if (-not (_confirm_action -Prompt "Close $count $($browser.BrowserName) process(es)?")) {
            return [pscustomobject]@{
                Browser = $browser.BrowserName
                Closed  = 0
                Status  = "Cancelled"
            }
        }
    }

    $count = $procs.Count
    $procs | Stop-Process -Force

    return [pscustomobject]@{
        Browser = $browser.BrowserName
        Closed  = $count
        Status  = "Terminated"
    }
}

# -----------------------------------------------------------------------------
# Open
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Open a URL in the default browser.
.DESCRIPTION
Launches the specified URL using the system default browser.
.PARAMETER Url
URL to open.
.EXAMPLE
browser_open -Url "https://github.com"
#>
function browser_open {
    param(
        [Parameter(Mandatory = $true, Position = 0)]
        [string]$Url
    )

    Start-Process $Url

    return [pscustomobject]@{
        Action = "Open"
        Url    = $Url
        Status = "Launched"
    }
}

<#
.SYNOPSIS
Open localhost at a given port in the browser.
.DESCRIPTION
Builds an http://localhost:<Port>/<Path> URL and opens it.
.PARAMETER Port
TCP port number (1-65535).
.PARAMETER Path
Optional URL path appended after the port.
.EXAMPLE
browser_localhost -Port 3000
.EXAMPLE
browser_localhost -Port 4200 -Path "admin"
#>
function browser_localhost {
    param(
        [Parameter(Mandatory = $true, Position = 0)]
        [int]$Port,

        [Parameter(Position = 1)]
        [string]$Path = ""
    )

    if ($Port -lt 1 -or $Port -gt 65535) {
        throw "Invalid port number: $Port"
    }

    $url = "http://localhost:$Port"

    if ($Path -and $Path.Trim().Length -gt 0) {
        $url = "$url/$($Path.TrimStart('/'))"
    }

    Start-Process $url

    return [pscustomobject]@{
        Action = "OpenLocalhost"
        Url    = $url
        Status = "Launched"
    }
}

<#
.SYNOPSIS
Open multiple URLs in the default browser.
.DESCRIPTION
Launches each URL in sequence using the system default browser.
.PARAMETER Urls
Array of URLs to open.
.EXAMPLE
browser_open_many -Urls "https://github.com","https://google.com"
#>
function browser_open_many {
    param(
        [Parameter(Mandatory = $true, Position = 0)]
        [string[]]$Urls
    )

    foreach ($url in $Urls) {
        Start-Process $url
    }

    return [pscustomobject]@{
        Count  = $Urls.Count
        Status = "Launched"
    }
}

# -----------------------------------------------------------------------------
# Port testing
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Test if a localhost port is open.
.DESCRIPTION
Uses Test-NetConnection to check TCP connectivity on localhost.
.PARAMETER Port
TCP port number to test (1-65535).
.EXAMPLE
browser_test_port -Port 8080
#>
function browser_test_port {
    param(
        [Parameter(Mandatory = $true, Position = 0)]
        [int]$Port
    )

    if ($Port -lt 1 -or $Port -gt 65535) {
        throw "Invalid port number: $Port"
    }

    $result = Test-NetConnection `
        -ComputerName "localhost" `
        -Port $Port `
        -WarningAction SilentlyContinue

    return [pscustomobject]@{
        Port   = $Port
        IsOpen = $result.TcpTestSucceeded
    }
}

<#
.SYNOPSIS
Wait for a localhost port to become available then open browser.
.DESCRIPTION
Polls the port at intervals and opens the URL once it responds.
.PARAMETER Port
TCP port to wait for (1-65535).
.PARAMETER TimeoutSeconds
Maximum seconds to wait (default 30).
.PARAMETER PollIntervalSeconds
Seconds between checks (default 1).
.EXAMPLE
browser_wait_and_open -Port 4200
.EXAMPLE
browser_wait_and_open -Port 3000 -TimeoutSeconds 60
#>
function browser_wait_and_open {
    param(
        [Parameter(Mandatory = $true, Position = 0)]
        [int]$Port,

        [Parameter(Position = 1)]
        [int]$TimeoutSeconds = 30,

        [Parameter(Position = 2)]
        [int]$PollIntervalSeconds = 1
    )

    if ($Port -lt 1 -or $Port -gt 65535) {
        throw "Invalid port number: $Port"
    }

    $elapsed = 0

    while ($elapsed -lt $TimeoutSeconds) {

        $result = Test-NetConnection `
            -ComputerName "localhost" `
            -Port $Port `
            -WarningAction SilentlyContinue

        if ($result.TcpTestSucceeded) {

            $url = "http://localhost:$Port"
            Start-Process $url

            return [pscustomobject]@{
                Port   = $Port
                Status = "Opened"
                Url    = $url
            }
        }

        Start-Sleep -Seconds $PollIntervalSeconds
        $elapsed += $PollIntervalSeconds
    }

    throw "Port $Port did not become available within $TimeoutSeconds seconds."
}
