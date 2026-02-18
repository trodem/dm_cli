# =============================================================================
# DM TRANSLATE TOOLKIT – Systran translation API operational layer
# Production-safe translation helpers
# Non-destructive defaults, explicit -Force for overwrite
# No external dependencies
# Entry point: dm_translate_*
#
# FUNCTIONS
#   dm_translate_text
#   dm_translate_file
#   dm_translate_folder
#   dm_translate_detect_language
#   dm_translate_supported_languages
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal Helpers
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Invoke _assert_path_exists.
.DESCRIPTION
Helper/command function for _assert_path_exists.
.EXAMPLE
dm _assert_path_exists
#>
function _assert_path_exists {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )
    if (-not (Test-Path -Path $Path)) {
        throw "Path '$Path' does not exist."
    }
}

<#
.SYNOPSIS
Invoke _get_systran_api_key.
.DESCRIPTION
Helper/command function for _get_systran_api_key.
.EXAMPLE
dm _get_systran_api_key
#>
function _get_systran_api_key {
    param(
        [string]$ApiKey
    )

    if ($ApiKey) {
        return $ApiKey
    }

    if ($env:SYSTRAN_API_KEY) {
        return $env:SYSTRAN_API_KEY
    }

    throw "Systran API key not provided. Use -ApiKey or set SYSTRAN_API_KEY environment variable."
}

<#
.SYNOPSIS
Invoke _invoke_systran_request.
.DESCRIPTION
Helper/command function for _invoke_systran_request.
.EXAMPLE
dm _invoke_systran_request
#>
function _invoke_systran_request {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Endpoint,

        [Parameter(Mandatory = $true)]
        [hashtable]$Body,

        [Parameter(Mandatory = $true)]
        [string]$ApiKey,

        [int]$TimeoutSeconds = 60
    )

    $headers = @{
        "Authorization" = "Bearer $ApiKey"
        "Content-Type"  = "application/json"
    }

    $json = $Body | ConvertTo-Json -Depth 10

    Invoke-RestMethod `
        -Uri "https://stadler.mysystran.com:8904" `
        -Method Post `
        -Headers $headers `
        -Body $json `
        -TimeoutSec $TimeoutSeconds
}

# -----------------------------------------------------------------------------
# Public Functions
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Translate raw text using Systran API.
#>
function translate_text {
    param(
        [Parameter(Mandatory = $true, Position = 0)]
        [string]$Text,

        [Parameter(Mandatory = $true, Position = 1)]
        [string]$SourceLang,

        [Parameter(Mandatory = $true, Position = 2)]
        [string]$TargetLang,

        [Parameter(Position = 3)]
        [string]$ApiKey,

        [Parameter(Position = 4)]
        [int]$TimeoutSeconds = 60
    )

    $key = _get_systran_api_key -ApiKey $ApiKey

    $body = @{
        input  = $Text
        source = $SourceLang
        target = $TargetLang
    }

    $response = _invoke_systran_request `
        -Endpoint "translation/text/translate" `
        -Body $body `
        -ApiKey $key `
        -TimeoutSeconds $TimeoutSeconds

    [pscustomobject]@{
        Original     = $Text
        Translated   = $response.outputs[0].output
        SourceLang   = $SourceLang
        TargetLang   = $TargetLang
        Characters   = $Text.Length
    }
}

<#
.SYNOPSIS
Translate a single file.
#>
function translate_file {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path,

        [Parameter(Mandatory = $true)]
        [string]$SourceLang,

        [Parameter(Mandatory = $true)]
        [string]$TargetLang,

        [Parameter(Mandatory = $true)]
        [string]$OutputPath,

        [string]$Encoding = "UTF8",

        [switch]$Force,
        [switch]$DryRun,

        [string]$ApiKey
    )

    _assert_path_exists -Path $Path

    if ((Test-Path $OutputPath) -and (-not $Force)) {
        throw "Output file already exists. Use -Force to overwrite."
    }

    $content = Get-Content -Path $Path -Raw -Encoding $Encoding

    if ($DryRun) {
        return [pscustomobject]@{
            File        = $Path
            OutputPath  = $OutputPath
            Characters  = $content.Length
            DryRun      = $true
        }
    }

    $result = translate_text `
        -Text $content `
        -SourceLang $SourceLang `
        -TargetLang $TargetLang `
        -ApiKey $ApiKey

    Set-Content `
        -Path $OutputPath `
        -Value $result.Translated `
        -Encoding $Encoding `
        -Force:$Force

    [pscustomobject]@{
        File        = $Path
        OutputPath  = $OutputPath
        Characters  = $content.Length
        Translated  = $true
    }
}

<#
.SYNOPSIS
Translate all files in a folder.
#>
function translate_folder {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path,

        [Parameter(Mandatory = $true)]
        [string]$SourceLang,

        [Parameter(Mandatory = $true)]
        [string]$TargetLang,

        [Parameter(Mandatory = $true)]
        [string]$OutputPath,

        [string]$Filter = "*.txt",

        [string]$Encoding = "UTF8",

        [switch]$Recurse,
        [switch]$Force,
        [switch]$DryRun,

        [string]$ApiKey
    )

    _assert_path_exists -Path $Path

    if (-not (Test-Path $OutputPath)) {
        New-Item -ItemType Directory -Path $OutputPath | Out-Null
    }

    $files = Get-ChildItem `
        -Path $Path `
        -Filter $Filter `
        -File `
        -Recurse:$Recurse

    $results = @()
    $totalChars = 0

    foreach ($file in $files) {

        $relative = $file.FullName.Substring($Path.Length).TrimStart("\")
        $dest = Join-Path $OutputPath $relative
        $destDir = Split-Path $dest -Parent

        if (-not (Test-Path $destDir)) {
            New-Item -ItemType Directory -Path $destDir -Force | Out-Null
        }

        $r = translate_file `
            -Path $file.FullName `
            -SourceLang $SourceLang `
            -TargetLang $TargetLang `
            -OutputPath $dest `
            -Encoding $Encoding `
            -Force:$Force `
            -DryRun:$DryRun `
            -ApiKey $ApiKey

        $results += $r
        $totalChars += $r.Characters
    }

    [pscustomobject]@{
        SourcePath     = $Path
        OutputPath     = $OutputPath
        FileCount      = $results.Count
        TotalCharacters= $totalChars
        DryRun         = $DryRun.IsPresent
    }
}

<#
.SYNOPSIS
Detect language of a text.
#>
function translate_detect_language {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Text,

        [string]$ApiKey
    )

    $key = _get_systran_api_key -ApiKey $ApiKey

    $body = @{
        input = $Text
    }

    _invoke_systran_request `
        -Endpoint "translation/text/detect" `
        -Body $body `
        -ApiKey $key
}

<#
.SYNOPSIS
List supported translation languages.
#>
function translate_supported_languages {
    param(
        [string]$ApiKey
    )

    $key = _get_systran_api_key -ApiKey $ApiKey

    _invoke_systran_request `
        -Endpoint "translation/supportedLanguages" `
        -Body @{} `
        -ApiKey $key
}
