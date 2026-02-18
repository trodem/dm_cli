# =============================================================================
# STIBS DB TOOLKIT – Analytical & Intelligence queries
# Functions to inspect MariaDB STIBS database and answer structural questions
# Requires docker-compose mariadb service already configured
# Entry point: dm stibs_db_*
#
# FUNCTIONS
#   stibs_db_status
#   stibs_db_query
#   stibs_db_databases
#   stibs_db_schema
#   stibs_db_count
#   stibs_db_shell
#   stibs_db_export_dump
#   stibs_db_import_dump
#   stibs_db_tables
#   stibs_db_mysql_query
#   stibs_db_mysql_tables
#   stibs_db_mysql_dump
#   stibs_db_container
#   stibs_db_env
#   stibs_db_biggest_table
#   stibs_db_top_tables
#   stibs_db_total_records
#   stibs_db_empty_tables
#   stibs_db_biggest_size
#   stibs_db_find_column
#
# INTELLIGENCE
#   stibs_db_fk
#   stibs_db_indexes
#   stibs_db_search
#   stibs_db_explain
#   stibs_db_sample
#   stibs_db_cardinality
#   stibs_db_orphans
#   stibs_db_doctor
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

<#
.SYNOPSIS
Invoke _stibs_db_get_config.
.DESCRIPTION
Helper/command function for _stibs_db_get_config.
.EXAMPLE
dm _stibs_db_get_config
#>
function _stibs_db_get_config {
    $cfg = _stibs_db_config
    if ($null -eq $cfg) {
        throw "STIBS DB config is not available."
    }
    return $cfg
}

<#
.SYNOPSIS
Invoke _stibs_db_assert_identifier.
.DESCRIPTION
Helper/command function for _stibs_db_assert_identifier.
.EXAMPLE
dm _stibs_db_assert_identifier
#>
function _stibs_db_assert_identifier {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Value,
        [string]$Name = "identifier"
    )

    if ($Value -notmatch '^[A-Za-z_][A-Za-z0-9_]*$') {
        throw "Invalid $Name '$Value'. Allowed pattern: [A-Za-z_][A-Za-z0-9_]*"
    }

    return $Value
}

<#
.SYNOPSIS
Invoke _stibs_db_escape_sql_literal.
.DESCRIPTION
Helper/command function for _stibs_db_escape_sql_literal.
.EXAMPLE
dm _stibs_db_escape_sql_literal
#>
function _stibs_db_escape_sql_literal {
    param([Parameter(Mandatory = $true)][string]$Value)
    return $Value.Replace("'", "''")
}

<#
.SYNOPSIS
Invoke _stibs_db_escape_like.
.DESCRIPTION
Helper/command function for _stibs_db_escape_like.
.EXAMPLE
dm _stibs_db_escape_like
#>
function _stibs_db_escape_like {
    param([Parameter(Mandatory = $true)][string]$Value)
    $escaped = $Value.Replace("\", "\\").Replace("%", "\%").Replace("_", "\_")
    return $escaped.Replace("'", "''")
}

<#
.SYNOPSIS
Invoke _stibs_db_assert_limit.
.DESCRIPTION
Helper/command function for _stibs_db_assert_limit.
.EXAMPLE
dm _stibs_db_assert_limit
#>
function _stibs_db_assert_limit {
    param(
        [int]$Value,
        [int]$Default = 5
    )

    if ($Value -le 0) {
        return $Default
    }

    if ($Value -gt 1000) {
        throw "Limit cannot be greater than 1000."
    }

    return $Value
}

# ------------------------------------------------------------
# CORE
# ------------------------------------------------------------

<#
.SYNOPSIS
Verifica se il container MariaDB è attivo.
#>
function stibs_db_status {
    _assert_command_available -Name docker
    $cfg = _stibs_db_get_config
    docker ps --filter "name=$($cfg.Container)"
}

<#
.SYNOPSIS
Esegue una query MariaDB nel container Docker.
.PARAMETER Sql
Query SQL da eseguire.
#>
function stibs_db_query {
    param(
        [Parameter(Mandatory)]
        [string]$Sql
    )

    _assert_command_available -Name docker
    $cfg = _stibs_db_get_config

    docker exec -i $($cfg.Container) `
        mysql -u$($cfg.User) -p$($cfg.Password) $($cfg.Database) `
        --batch --skip-column-names `
        -e "$Sql"
}

<#
.SYNOPSIS
Mostra database MariaDB.
#>
function stibs_db_databases {
    stibs_db_query "SHOW DATABASES;"
}

<#
.SYNOPSIS
Mostra schema tabella.
#>
function stibs_db_schema {
    param(
        [Parameter(Mandatory)]
        [string]$Table
    )

    $safeTable = _stibs_db_assert_identifier -Value $Table -Name "table"
    stibs_db_query "DESCRIBE $safeTable;"
}

<#
.SYNOPSIS
Conta righe di una tabella.
#>
function stibs_db_count {
    param(
        [Parameter(Mandatory)]
        [string]$Table
    )

    $safeTable = _stibs_db_assert_identifier -Value $Table -Name "table"
    stibs_db_query "SELECT COUNT(*) AS total FROM $safeTable;"
}

<#
.SYNOPSIS
Apre shell MariaDB nel container.
#>
function stibs_db_shell {
    _assert_command_available -Name docker
    $cfg = _stibs_db_get_config
    docker exec -it $($cfg.Container) `
        mysql -u$($cfg.User) -p$($cfg.Password) $($cfg.Database)
}

# ------------------------------------------------------------
# EXPORT / IMPORT
# ------------------------------------------------------------

<#
.SYNOPSIS
Esporta il database MariaDB in un file .zip.
#>
function stibs_db_export_dump {
    param(
        [string]$Output = "$env:USERPROFILE\Downloads"
    )

    _assert_command_available -Name docker
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"

    if ($Output -match '\.zip$') {
        $parent = Split-Path $Output -Parent
        if ([string]::IsNullOrWhiteSpace($parent)) {
            $parent = "$env:USERPROFILE\Downloads"
        }

        if (-not (Test-Path $parent)) {
            New-Item -ItemType Directory -Path $parent -Force | Out-Null
        }

        $zipFile = Join-Path (Resolve-Path $parent).Path (Split-Path $Output -Leaf)
    }
    else {
        if (-not (Test-Path $Output)) {
            New-Item -ItemType Directory -Path $Output -Force | Out-Null
        }

        $dir = (Resolve-Path $Output).Path
        $cfg = _stibs_db_get_config
        $zipFile = Join-Path $dir "$($cfg.Database)-$timestamp.zip"
    }

    $cfg = _stibs_db_get_config
    $sqlFile = Join-Path $env:TEMP "$($cfg.Database)-$timestamp.sql"

    docker exec $($cfg.Container) `
        mysqldump --single-transaction --quick --lock-tables=false `
        -u$($cfg.User) -p$($cfg.Password) $($cfg.Database) `
        | Out-File -FilePath $sqlFile -Encoding utf8

    Compress-Archive -Path $sqlFile -DestinationPath $zipFile -Force
    Remove-Item $sqlFile -Force

    Write-Output "Dump creato in: $zipFile"
}

<#
.SYNOPSIS
Ripristina il database MariaDB da un dump .zip.
.PARAMETER ZipFile
Percorso del file zip contenente il dump SQL.
.PARAMETER Force
Salta la conferma interattiva prima del drop/recreate database.
.EXAMPLE
dm stibs_db_import_dump C:\Users\me\Downloads\stibs-20260218-101010.zip
.EXAMPLE
dm stibs_db_import_dump C:\Users\me\Downloads\stibs-20260218-101010.zip -Force
#>
function stibs_db_import_dump {
    param(
        [Parameter(Mandatory)]
        [string]$ZipFile,
        [switch]$Force
    )

    _assert_command_available -Name docker
    if (-not (Test-Path -LiteralPath $ZipFile)) {
        throw "File non trovato: $ZipFile"
    }

    if (-not $Force) {
        if (-not (_confirm_action -Prompt "This will drop and recreate database. Continue")) {
            Write-Output "Operazione annullata."
            return
        }
    }

    $tempDir = Join-Path $env:TEMP ("dbrestore_" + (Get-Date -Format "yyyyMMddHHmmss"))
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

    Expand-Archive -Path $ZipFile -DestinationPath $tempDir -Force

    $sqlFile = Get-ChildItem $tempDir -Filter *.sql | Select-Object -First 1

    $cfg = _stibs_db_get_config
    $dbNameLiteral = _stibs_db_escape_sql_literal -Value $cfg.Database

    docker exec $($cfg.Container) `
        mysql -u$($cfg.User) -p$($cfg.Password) `
        -e "DROP DATABASE IF EXISTS $dbNameLiteral; CREATE DATABASE $dbNameLiteral;"

    Get-Content $sqlFile.FullName -Raw | docker exec -i $($cfg.Container) `
        mysql -u$($cfg.User) -p$($cfg.Password) $($cfg.Database)

    Remove-Item $tempDir -Recurse -Force

    Write-Output "Restore completato con successo"
}

# ------------------------------------------------------------
# GENERIC MYSQL WRAPPERS
# ------------------------------------------------------------

<#
.SYNOPSIS
Esegue query MySQL generica.
#>
function stibs_db_mysql_query {
    param(
        [Parameter(Mandatory)][string]$Container,
        [Parameter(Mandatory)][string]$User,
        [Parameter(Mandatory)][string]$Password,
        [Parameter(Mandatory)][string]$Database,
        [Parameter(Mandatory)][string]$Query
    )

    _assert_command_available -Name docker
    docker exec -i $Container mysql -u$User -p$Password $Database -e $Query
}

<#
.SYNOPSIS
Elenca tabelle MySQL generiche.
#>
function stibs_db_mysql_tables {
    param(
        [Parameter(Mandatory)][string]$Container,
        [Parameter(Mandatory)][string]$User,
        [Parameter(Mandatory)][string]$Password,
        [Parameter(Mandatory)][string]$Database
    )

    stibs_db_mysql_query $Container $User $Password $Database "SHOW TABLES;"
}

<#
.SYNOPSIS
Dump MySQL generico.
#>
function stibs_db_mysql_dump {
    param(
        [Parameter(Mandatory)][string]$Container,
        [Parameter(Mandatory)][string]$User,
        [Parameter(Mandatory)][string]$Password,
        [Parameter(Mandatory)][string]$Database,
        [Parameter(Mandatory)][string]$Output
    )

    _assert_command_available -Name docker
    docker exec $Container mysqldump -u$User -p$Password $Database > $Output
}

# ------------------------------------------------------------
# DISCOVERY
# ------------------------------------------------------------

<#
.SYNOPSIS
Restituisce container MariaDB.
#>
function stibs_db_container {
    _assert_command_available -Name docker
    docker compose ps -q mariadb
}

<#
.SYNOPSIS
Recupera variabili ambiente MariaDB.
#>
function stibs_db_env {
    _assert_command_available -Name docker
    $cfg = _stibs_db_get_config
    docker inspect $($cfg.Container) |
        ConvertFrom-Json |
        Select-Object -ExpandProperty Config |
        Select-Object -ExpandProperty Env
}

# ------------------------------------------------------------
# ANALYTICS
# ------------------------------------------------------------

<#
.SYNOPSIS
Elenca tabelle database.
#>
function stibs_db_tables {
    stibs_db_query "SHOW TABLES;"
}

<#
.SYNOPSIS
Tabella con più record.
#>
function stibs_db_biggest_table {
    $cfg = _stibs_db_get_config
    $dbNameLiteral = _stibs_db_escape_sql_literal -Value $cfg.Database

    $sql = @"
SELECT table_name, table_rows
FROM information_schema.tables
WHERE table_schema = '$dbNameLiteral'
ORDER BY table_rows DESC
LIMIT 1;
"@

    stibs_db_query $sql
}

<#
.SYNOPSIS
Top tabelle per numero record.
#>
function stibs_db_top_tables {
    param([int]$Limit = 5)
    $safeLimit = _stibs_db_assert_limit -Value $Limit -Default 5
    $cfg = _stibs_db_get_config
    $dbNameLiteral = _stibs_db_escape_sql_literal -Value $cfg.Database

    $sql = @"
SELECT table_name, table_rows
FROM information_schema.tables
WHERE table_schema = '$dbNameLiteral'
ORDER BY table_rows DESC
LIMIT $safeLimit;
"@

    stibs_db_query $sql
}

<#
.SYNOPSIS
Numero totale record database.
#>
function stibs_db_total_records {
    $cfg = _stibs_db_get_config
    $dbNameLiteral = _stibs_db_escape_sql_literal -Value $cfg.Database

    $sql = @"
SELECT SUM(table_rows) AS total_records
FROM information_schema.tables
WHERE table_schema = '$dbNameLiteral';
"@

    stibs_db_query $sql
}

<#
.SYNOPSIS
Tabelle vuote.
#>
function stibs_db_empty_tables {
    $cfg = _stibs_db_get_config
    $dbNameLiteral = _stibs_db_escape_sql_literal -Value $cfg.Database

    $sql = @"
SELECT table_name
FROM information_schema.tables
WHERE table_schema = '$dbNameLiteral'
AND table_rows = 0;
"@

    stibs_db_query $sql
}

<#
.SYNOPSIS
Tabelle più grandi per dimensione.
#>
function stibs_db_biggest_size {
    param([int]$Limit = 5)
    $safeLimit = _stibs_db_assert_limit -Value $Limit -Default 5
    $cfg = _stibs_db_get_config
    $dbNameLiteral = _stibs_db_escape_sql_literal -Value $cfg.Database

    $sql = @"
SELECT 
    table_name,
    ROUND((data_length + index_length) / 1024 / 1024, 2) AS size_mb
FROM information_schema.tables
WHERE table_schema = '$dbNameLiteral'
ORDER BY size_mb DESC
LIMIT $safeLimit;
"@

    stibs_db_query $sql
}

<#
.SYNOPSIS
Trova colonne nel database.
#>
function stibs_db_find_column {
    param([Parameter(Mandatory)][string]$Column)
    $cfg = _stibs_db_get_config
    $dbNameLiteral = _stibs_db_escape_sql_literal -Value $cfg.Database
    $columnLike = _stibs_db_escape_like -Value $Column

    $sql = @"
SELECT table_name, column_name
FROM information_schema.columns
WHERE table_schema = '$dbNameLiteral'
AND column_name LIKE '%$columnLike%' ESCAPE '\\';
"@

    stibs_db_query $sql
}

# ------------------------------------------------------------
# INTELLIGENCE
# ------------------------------------------------------------

<#
.SYNOPSIS
Foreign key relations per tabella.
#>
function stibs_db_fk {
    param([Parameter(Mandatory)][string]$Table)
    $cfg = _stibs_db_get_config
    $dbNameLiteral = _stibs_db_escape_sql_literal -Value $cfg.Database
    $tableLiteral = _stibs_db_escape_sql_literal -Value $Table

    stibs_db_query @"
SELECT table_name, column_name, referenced_table_name, referenced_column_name
FROM information_schema.key_column_usage
WHERE table_schema = '$dbNameLiteral'
AND (table_name = '$tableLiteral' OR referenced_table_name = '$tableLiteral');
"@
}

<#
.SYNOPSIS
Mostra indici tabella.
#>
function stibs_db_indexes {
    param([Parameter(Mandatory)][string]$Table)
    $safeTable = _stibs_db_assert_identifier -Value $Table -Name "table"
    stibs_db_query "SHOW INDEX FROM $safeTable;"
}

<#
.SYNOPSIS
Explain query.
#>
function stibs_db_explain {
    param([Parameter(Mandatory)][string]$Query)
    stibs_db_query "EXPLAIN $Query;"
}

<#
.SYNOPSIS
Sample dati tabella.
#>
function stibs_db_sample {
    param(
        [Parameter(Mandatory)][string]$Table,
        [int]$Limit = 10
    )
    $safeTable = _stibs_db_assert_identifier -Value $Table -Name "table"
    $safeLimit = _stibs_db_assert_limit -Value $Limit -Default 10
    stibs_db_query "SELECT * FROM $safeTable LIMIT $safeLimit;"
}

<#
.SYNOPSIS
Trova record orfani.
#>
function stibs_db_orphans {
    param(
        [Parameter(Mandatory)][string]$ChildTable,
        [Parameter(Mandatory)][string]$ChildColumn,
        [Parameter(Mandatory)][string]$ParentTable,
        [Parameter(Mandatory)][string]$ParentColumn
    )

    $safeChildTable = _stibs_db_assert_identifier -Value $ChildTable -Name "child table"
    $safeChildColumn = _stibs_db_assert_identifier -Value $ChildColumn -Name "child column"
    $safeParentTable = _stibs_db_assert_identifier -Value $ParentTable -Name "parent table"
    $safeParentColumn = _stibs_db_assert_identifier -Value $ParentColumn -Name "parent column"

    stibs_db_query @"
SELECT *
FROM $safeChildTable c
LEFT JOIN $safeParentTable p
ON c.$safeChildColumn = p.$safeParentColumn
WHERE p.$safeParentColumn IS NULL;
"@
}

<#
.SYNOPSIS
Diagnostica database.
#>
function stibs_db_doctor {

    Write-Host "Checking DB connection..."
    stibs_db_query "SELECT 1;"

    Write-Host "Tables:"
    stibs_db_tables

    Write-Host "Empty tables:"
    stibs_db_empty_tables

    Write-Host "Largest tables:"
    stibs_db_biggest_size
}
