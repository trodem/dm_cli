Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$varTestPath="C:\Users\Demtro\Downloads"


<#
.SYNOPSIS
Invoke dev_path.
.DESCRIPTION
Helper/command function for dev_path.
.EXAMPLE
dm dev_path
#>
function dev_path {
    Set-Location $varDevPath
}


<#
.SYNOPSIS
Invoke dm_cli_path.
.DESCRIPTION
Helper/command function for dm_cli_path.
.EXAMPLE
dm dm_cli_path
#>
function dm_cli_path {
    Set-Location $varTestPath
}

<#
.SYNOPSIS
Invoke c.
.DESCRIPTION
Helper/command function for c.
.EXAMPLE
dm c
#>
function c { Clear-Host }

<#
.SYNOPSIS
Invoke grep.
.DESCRIPTION
Helper/command function for grep.
.EXAMPLE
dm grep
#>
function grep {
    Select-String $args
}

<#
.SYNOPSIS
Invoke mkcd.
.DESCRIPTION
Helper/command function for mkcd.
.EXAMPLE
dm mkcd
#>
function mkcd {
    param($name)
    mkdir $name
    cd $name
}
