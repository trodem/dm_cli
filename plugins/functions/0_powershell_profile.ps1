Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$varTestPath="C:\Users\Demtro\Downloads"


function dev_path {
    Set-Location $varDevPath
}


function dm_cli_path {
    Set-Location $varTestPath
}

function c { Clear-Host }

function grep {
    Select-String $args
}

function mkcd {
    param($name)
    mkdir $name
    cd $name
}
