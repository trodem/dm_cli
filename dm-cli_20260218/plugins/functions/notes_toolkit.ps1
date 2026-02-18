# =============================================================================
# NOTES TOOLKIT – Single-file, non-interactive (dm compatible)
# All notes saved in one file. Input via parameters only.
# =============================================================================

<#
.SYNOPSIS
Invoke notes_file.
.DESCRIPTION
Helper/command function for notes_file.
.EXAMPLE
dm notes_file
#>
function notes_file {
    $base = Join-Path $env:USERPROFILE "notes"
    if (-not (Test-Path $base)) { New-Item -ItemType Directory -Path $base | Out-Null }

    $file = Join-Path $base "notes.txt"
    if (-not (Test-Path $file)) { New-Item -ItemType File -Path $file | Out-Null }

    return $file
}


<#
.SYNOPSIS
Add a new note.
.PARAMETER Text
Text of the note.
.EXAMPLE
dm notes_new "Call customer tomorrow"
#>
function notes_new {
    param(
        [Parameter(Mandatory)]
        [string]$Text
    )

    $file = notes_file
    $entry = "[{0}] {1}" -f (Get-Date -Format "yyyy-MM-dd HH:mm"), $Text

    Add-Content -Path $file -Value $entry
}


<#
.SYNOPSIS
Append multiline note.
.PARAMETER Text
Multiline text allowed.
.EXAMPLE
dm notes_write "Line1`nLine2"
#>
function notes_write {
    param(
        [Parameter(Mandatory)]
        [string]$Text
    )

    $file = notes_file
    $header = "[{0}]" -f (Get-Date -Format "yyyy-MM-dd HH:mm")

    Add-Content -Path $file -Value $header
    Add-Content -Path $file -Value $Text
}


<#
.SYNOPSIS
Read all notes.
.EXAMPLE
dm notes_read
#>
function notes_read {
    Get-Content (notes_file)
}


<#
.SYNOPSIS
Show last lines of notes.
.PARAMETER Lines
Number of lines.
.EXAMPLE
dm notes_last 20
#>
function notes_last {
    param([int]$Lines = 20)
    Get-Content (notes_file) -Tail $Lines
}


<#
.SYNOPSIS
Search notes.
.PARAMETER Text
Text to search.
.EXAMPLE
dm notes_search "meeting"
#>
function notes_search {
    param(
        [Parameter(Mandatory)]
        [string]$Text
    )

    Select-String -Path (notes_file) -Pattern $Text
}


<#
.SYNOPSIS
Clear notes file.
.EXAMPLE
dm notes_clear
#>
function notes_clear {
    Clear-Content (notes_file)
}


<#
.SYNOPSIS
Open notes folder.
.EXAMPLE
dm notes_folder
#>
function notes_folder {
    Start-Process (Split-Path (notes_file))
}
