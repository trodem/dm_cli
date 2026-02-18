# =============================================================================
# BROWSER TOOLKIT – Local browser operational layer
# Production-safe local browser helpers
# Non-destructive defaults, deterministic behavior, no admin requirements
# Entry point: browser_*
#
# FUNCTIONS
#   browser_close_all
#	browser_standard 
#   browser_open
#   browser_localhost
#   browser_wait_and_open
#   browser_open_many
#   browser_test_port
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

<#
.SYNOPSIS
Invoke browser_close_all.
.DESCRIPTION
Helper/command function for browser_close_all.
.EXAMPLE
dm browser_close_all
#>
function browser_close_all {
    param()

    $regPath = "HKCU:\Software\Microsoft\Windows\Shell\Associations\UrlAssociations\http\UserChoice"

    if (-not (Test-Path $regPath)) {
        throw "Cannot determine default browser."
    }

    $progId = (Get-ItemProperty -Path $regPath).ProgId

    $processMap = @{
        "ChromeHTML"  = "chrome"
        "MSEdgeHTM"   = "msedge"
        "FirefoxURL"  = "firefox"
        "OperaStable" = "opera"
        "BraveHTML"   = "brave"
        "SafariURL"   = "safari"
    }

    $processName = $processMap[$progId]

    if (-not $processName) {
        throw "Unsupported or unknown default browser ($progId)."
    }

    $procs = Get-Process -Name $processName -ErrorAction SilentlyContinue

    if (-not $procs) {
        return [pscustomobject]@{
            Browser = $processName
            Closed  = 0
            Status  = "No running instances"
        }
    }

    $count = $procs.Count
    $procs | Stop-Process -Force

    [pscustomobject]@{
        Browser = $processName
        Closed  = $count
        Status  = "Terminated"
    }
}


<#
.SYNOPSIS
Invoke browser_standard.
.DESCRIPTION
Helper/command function for browser_standard.
.EXAMPLE
dm browser_standard
#>
function browser_standard {
    param()

    $regPath = "HKCU:\Software\Microsoft\Windows\Shell\Associations\UrlAssociations\http\UserChoice"

    if (-not (Test-Path $regPath)) {
        throw "Cannot determine default browser (registry key not found)."
    }

    $progId = (Get-ItemProperty -Path $regPath).ProgId

    $browserMap = @{
        "ChromeHTML"      = "Google Chrome"
        "MSEdgeHTM"       = "Microsoft Edge"
        "FirefoxURL"      = "Mozilla Firefox"
        "OperaStable"     = "Opera"
        "SafariURL"       = "Safari"
        "BraveHTML"       = "Brave"
    }

    $browserName = $browserMap[$progId]

    if (-not $browserName) {
        $browserName = "Unknown ($progId)"
    }

    [pscustomobject]@{
        BrowserName = $browserName
        ProgId      = $progId
    }
}


# -----------------------------------------------------------------------------
# Open any URL in default browser
# -----------------------------------------------------------------------------
<#
.SYNOPSIS
Invoke browser_open.
.DESCRIPTION
Helper/command function for browser_open.
.EXAMPLE
dm browser_open
#>
function browser_open {
    param(
        [Parameter(Mandatory = $true, Position = 0)]
        [string]$Url
    )

    Start-Process $Url

    [pscustomobject]@{
        Action = "Open"
        Url    = $Url
        Status = "Launched"
    }
}

# -----------------------------------------------------------------------------
# Open localhost with port
# -----------------------------------------------------------------------------
<#
.SYNOPSIS
Invoke browser_localhost.
.DESCRIPTION
Helper/command function for browser_localhost.
.EXAMPLE
dm browser_localhost
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

    [pscustomobject]@{
        Action = "OpenLocalhost"
        Url    = $url
        Status = "Launched"
    }
}

# -----------------------------------------------------------------------------
# Test if localhost port is open
# -----------------------------------------------------------------------------
<#
.SYNOPSIS
Invoke browser_test_port.
.DESCRIPTION
Helper/command function for browser_test_port.
.EXAMPLE
dm browser_test_port
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

    [pscustomobject]@{
        Port   = $Port
        IsOpen = $result.TcpTestSucceeded
    }
}

# -----------------------------------------------------------------------------
# Wait for localhost port and open browser
# -----------------------------------------------------------------------------
<#
.SYNOPSIS
Invoke browser_wait_and_open.
.DESCRIPTION
Helper/command function for browser_wait_and_open.
.EXAMPLE
dm browser_wait_and_open
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

# -----------------------------------------------------------------------------
# Open multiple URLs
# -----------------------------------------------------------------------------
<#
.SYNOPSIS
Invoke browser_open_many.
.DESCRIPTION
Helper/command function for browser_open_many.
.EXAMPLE
dm browser_open_many
#>
function browser_open_many {
    param(
        [Parameter(Mandatory = $true, Position = 0)]
        [string[]]$Urls
    )

    foreach ($url in $Urls) {
        Start-Process $url
    }

    [pscustomobject]@{
        Count  = $Urls.Count
        Status = "Launched"
    }
}
