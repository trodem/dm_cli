<#
.SYNOPSIS
Conta i file in una cartella.
.DESCRIPTION
Restituisce il numero totale di file (non directory).
.EXAMPLE
dm count_files C:\Temp
#>
function count_files {
    param(
        [Parameter(Mandatory)]
        [string]$Path
    )
    _assert_path_exists -Path $Path
    (Get-ChildItem -Path $Path -File -Recurse).Count
}

<#
.SYNOPSIS
Conta le directory in una cartella.
.EXAMPLE
dm count_dirs C:\Projects
#>
function count_dirs {
    param(
        [Parameter(Mandatory)]
        [string]$Path
    )
    _assert_path_exists -Path $Path
    (Get-ChildItem -Path $Path -Directory -Recurse).Count
}

<#
.SYNOPSIS
Mostra i file più grandi in una cartella.
.EXAMPLE
dm biggest_files C:\Data 10
#>
function biggest_files {
    param(
        [Parameter(Mandatory)]
        [string]$Path,
        [int]$Top = 10
    )
    _assert_path_exists -Path $Path
    Get-ChildItem -Path $Path -File -Recurse |
        Sort-Object Length -Descending |
        Select-Object -First $Top Name, Length, FullName
}

<#
.SYNOPSIS
Dimensione totale di una cartella.
.EXAMPLE
dm folder_size C:\Media
#>
function folder_size {
    param(
        [Parameter(Mandatory)]
        [string]$Path
    )
    _assert_path_exists -Path $Path
    $bytes = (Get-ChildItem -Path $Path -File -Recurse | Measure-Object Length -Sum).Sum
    "{0:N2} GB" -f ($bytes / 1GB)
}

<#
.SYNOPSIS
Apri cartella corrente in Explorer.
.EXAMPLE
dm open_here
#>
function open_here {
    Start-Process explorer.exe .
}

<#
.SYNOPSIS
Trova processi per nome.
.EXAMPLE
dm pgrep chrome
#>
function pgrep {
    param(
        [Parameter(Mandatory)]
        [string]$Name
    )
    Get-Process *$Name* | Select-Object Name, Id, CPU, WorkingSet
}

<#
.SYNOPSIS
Termina processo per nome.
.EXAMPLE
dm pkill chrome
#>
function pkill {
    param(
        [Parameter(Mandatory)]
        [string]$Name
    )
    Get-Process *$Name* -ErrorAction SilentlyContinue | Stop-Process -Force
}

<#
.SYNOPSIS
Mostra porte TCP in ascolto.
.EXAMPLE
dm ports
#>
function ports {
    Get-NetTCPConnection -State Listen |
        Select-Object LocalAddress, LocalPort, OwningProcess
}

<#
.SYNOPSIS
Ping rapido.
.EXAMPLE
dm pingg google.com
#>
function pingg {
    param(
        [Parameter(Mandatory)]
        [string]$Target
    )
    Test-Connection -ComputerName $Target -Count 4
}

<#
.SYNOPSIS
IP locale.
.EXAMPLE
dm myip
#>
function myip {
    Get-NetIPAddress -AddressFamily IPv4 |
        Where-Object {$_.InterfaceAlias -notlike "*Loopback*"} |
        Select-Object IPAddress, InterfaceAlias
}

<#
.SYNOPSIS
Docker: container in esecuzione + uso risorse.
.EXAMPLE
dm dstat
#>
function dstat {
    _assert_command_available -Name docker
    docker stats --no-stream
}

<#
.SYNOPSIS
Docker: immagini locali.
.EXAMPLE
dm dimages
#>
function dimages {
    _assert_command_available -Name docker
    docker images
}

<#
.SYNOPSIS
Docker: pulizia veloce.
.EXAMPLE
dm dclean
#>
function dclean {
    _assert_command_available -Name docker
    docker system prune -f
}

<#
.SYNOPSIS
Copia testo negli appunti.
.EXAMPLE
dm clip "hello"
#>
function clip {
    param(
        [Parameter(Mandatory)]
        [string]$Text
    )
    $Text | Set-Clipboard
}

<#
.SYNOPSIS
Leggi appunti.
.EXAMPLE
dm paste
#>
function paste {
    Get-Clipboard
}

<#
.SYNOPSIS
Log veloce con timestamp.
.EXAMPLE
dm log "deploy partito"
#>
function log {
    param(
        [Parameter(Mandatory)]
        [string]$Message
    )
    Write-Host "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') | $Message"
}

<#
.SYNOPSIS
Apri profilo PowerShell.
.EXAMPLE
dm profile
#>
function profile {
    code $PROFILE
}

<#
.SYNOPSIS
Reload profilo PowerShell.
.EXAMPLE
dm reload_profile
#>
function reload_profile {
    . $PROFILE
    Write-Host ">> Profilo ricaricato"
}
