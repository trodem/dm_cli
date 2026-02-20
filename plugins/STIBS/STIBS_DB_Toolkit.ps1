# =============================================================================
# STIBS DB TOOLKIT – Analytical & intelligence queries (standalone)
# Inspect the MariaDB STIBS database: structure, data, relationships.
# Requires docker and a running MariaDB container.
# Safety: Read-only defaults. Import requires -Force or confirmation.
# Entry point: stibs_db_*
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
#   stibs_db_container
#   stibs_db_env
#   stibs_db_tables
#   stibs_db_biggest_table
#   stibs_db_top_tables
#   stibs_db_total_records
#   stibs_db_empty_tables
#   stibs_db_biggest_size
#   stibs_db_find_column
#   stibs_db_fk
#   stibs_db_indexes
#   stibs_db_explain
#   stibs_db_sample
#   stibs_db_orphans
#   stibs_db_doctor
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal helpers — guards and config
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Ensure a command is available in PATH.
.PARAMETER Name
Command name to validate.
.EXAMPLE
_assert_command_available -Name docker
#>
function _assert_command_available {
    param([Parameter(Mandatory = $true)][string]$Name)
    if (-not (Get-Command -Name $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' was not found in PATH."
    }
}

<#
.SYNOPSIS
Ensure a filesystem path exists.
.PARAMETER Path
Path to validate.
.EXAMPLE
_assert_path_exists -Path "C:\Data"
#>
function _assert_path_exists {
    param([Parameter(Mandatory = $true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Required path '$Path' does not exist."
    }
}

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
Read an environment variable with a fallback default.
.PARAMETER Name
Environment variable name.
.PARAMETER Default
Value to return if the variable is unset or empty.
.EXAMPLE
_env_or_default -Name "DM_STIBS_DB_USER" -Default "stibs"
#>
function _env_or_default {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Default
    )
    $value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($value)) { return $Default }
    return $value
}

<#
.SYNOPSIS
Load STIBS database connection config.
.DESCRIPTION
Builds the config from environment variables with sensible defaults.
.EXAMPLE
_stibs_db_get_config
#>
function _stibs_db_get_config {
    return [pscustomobject]@{
        Container = _env_or_default -Name "DM_STIBS_DB_CONTAINER" -Default "docker-mariadb-1"
        User      = _env_or_default -Name "DM_STIBS_DB_USER"      -Default "stibs"
        Password  = _env_or_default -Name "DM_STIBS_DB_PASSWORD"  -Default "stibs"
        Database  = _env_or_default -Name "DM_STIBS_DB_NAME"      -Default "stibs"
    }
}

# -----------------------------------------------------------------------------
# Internal helpers — SQL safety
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Validate a SQL identifier against injection.
.DESCRIPTION
Throws if the value does not match the safe identifier pattern [A-Za-z_][A-Za-z0-9_]*.
.PARAMETER Value
Identifier string to validate.
.PARAMETER Name
Label for the error message (default: "identifier").
.EXAMPLE
_stibs_db_assert_identifier -Value "users" -Name "table"
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
Escape a string for use inside SQL single quotes.
.DESCRIPTION
Doubles single-quote characters to prevent SQL injection.
.PARAMETER Value
String to escape.
.EXAMPLE
_stibs_db_escape_sql_literal -Value "O'Brien"
#>
function _stibs_db_escape_sql_literal {
    param([Parameter(Mandatory = $true)][string]$Value)
    return $Value.Replace("'", "''")
}

<#
.SYNOPSIS
Escape a string for use in a SQL LIKE clause.
.DESCRIPTION
Escapes backslash, percent and underscore characters, then doubles single quotes.
.PARAMETER Value
String to escape.
.EXAMPLE
_stibs_db_escape_like -Value "user_name"
#>
function _stibs_db_escape_like {
    param([Parameter(Mandatory = $true)][string]$Value)
    $escaped = $Value.Replace("\", "\\").Replace("%", "\%").Replace("_", "\_")
    return $escaped.Replace("'", "''")
}

<#
.SYNOPSIS
Validate and clamp a LIMIT value for SQL queries.
.DESCRIPTION
Returns the default if value is zero or negative; throws if above 1000.
.PARAMETER Value
Requested limit.
.PARAMETER Default
Fallback value when Value is non-positive (default: 5).
.EXAMPLE
_stibs_db_assert_limit -Value 20 -Default 5
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

# -----------------------------------------------------------------------------
# Core
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Check if the MariaDB container is running.
.DESCRIPTION
Lists Docker containers matching the configured STIBS DB container name.
.EXAMPLE
stibs_db_status
#>
function stibs_db_status {
    _assert_command_available -Name docker
    $cfg = _stibs_db_get_config
    docker ps --filter "name=$($cfg.Container)"
}

<#
.SYNOPSIS
Execute a SQL query in the STIBS MariaDB container.
.DESCRIPTION
Runs the given SQL statement via docker exec against the configured database.
.PARAMETER Sql
SQL statement to execute.
.EXAMPLE
stibs_db_query -Sql "SELECT COUNT(*) FROM users;"
#>
function stibs_db_query {
    param(
        [Parameter(Mandatory = $true)]
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
List all databases in the MariaDB instance.
.EXAMPLE
stibs_db_databases
#>
function stibs_db_databases {
    stibs_db_query -Sql "SHOW DATABASES;"
}

<#
.SYNOPSIS
Show column schema of a table.
.PARAMETER Table
Table name to describe.
.EXAMPLE
stibs_db_schema -Table users
#>
function stibs_db_schema {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Table
    )

    $safeTable = _stibs_db_assert_identifier -Value $Table -Name "table"
    stibs_db_query -Sql "DESCRIBE $safeTable;"
}

<#
.SYNOPSIS
Count rows in a table.
.PARAMETER Table
Table name.
.EXAMPLE
stibs_db_count -Table users
#>
function stibs_db_count {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Table
    )

    $safeTable = _stibs_db_assert_identifier -Value $Table -Name "table"
    stibs_db_query -Sql "SELECT COUNT(*) AS total FROM $safeTable;"
}

<#
.SYNOPSIS
Open interactive MariaDB shell in the container.
.EXAMPLE
stibs_db_shell
#>
function stibs_db_shell {
    _assert_command_available -Name docker
    $cfg = _stibs_db_get_config
    docker exec -it $($cfg.Container) `
        mysql -u$($cfg.User) -p$($cfg.Password) $($cfg.Database)
}

# -----------------------------------------------------------------------------
# Export / Import
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Export the STIBS database as a zipped SQL dump.
.PARAMETER Output
Output directory or .zip file path (default: ~/Downloads).
.EXAMPLE
stibs_db_export_dump
.EXAMPLE
stibs_db_export_dump -Output "C:\backups\stibs.zip"
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

    return [pscustomobject]@{
        Path      = $zipFile
        Timestamp = $timestamp
    }
}

<#
.SYNOPSIS
Restore the STIBS database from a zipped SQL dump.
.DESCRIPTION
Drops and recreates the database, then imports the SQL file from the zip.
Requires -Force or interactive confirmation.
.PARAMETER ZipFile
Path to the zip file containing the SQL dump.
.PARAMETER Force
Skip interactive confirmation before drop/recreate.
.EXAMPLE
stibs_db_import_dump -ZipFile "C:\backups\stibs-20260218-101010.zip"
.EXAMPLE
stibs_db_import_dump -ZipFile "C:\backups\stibs-20260218-101010.zip" -Force
#>
function stibs_db_import_dump {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ZipFile,
        [switch]$Force
    )

    _assert_command_available -Name docker
    _assert_path_exists -Path $ZipFile

    if (-not $Force) {
        if (-not (_confirm_action -Prompt "This will drop and recreate the database. Continue")) {
            return [pscustomobject]@{ Status = "Cancelled" }
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

    return [pscustomobject]@{ Status = "Restored"; Source = $ZipFile }
}

# -----------------------------------------------------------------------------
# Discovery
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Return the STIBS MariaDB container ID.
.EXAMPLE
stibs_db_container
#>
function stibs_db_container {
    _assert_command_available -Name docker
    docker compose ps -q mariadb
}

<#
.SYNOPSIS
Show environment variables of the MariaDB container.
.EXAMPLE
stibs_db_env
#>
function stibs_db_env {
    _assert_command_available -Name docker
    $cfg = _stibs_db_get_config
    docker inspect $($cfg.Container) |
        ConvertFrom-Json |
        Select-Object -ExpandProperty Config |
        Select-Object -ExpandProperty Env
}

<#
.SYNOPSIS
List all tables in the STIBS database.
.EXAMPLE
stibs_db_tables
#>
function stibs_db_tables {
    stibs_db_query -Sql "SHOW TABLES;"
}

# -----------------------------------------------------------------------------
# Analytics
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Show the table with the most rows.
.EXAMPLE
stibs_db_biggest_table
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

    stibs_db_query -Sql $sql
}

<#
.SYNOPSIS
Show top tables ranked by row count.
.PARAMETER Limit
Number of tables to show (default 5, max 1000).
.EXAMPLE
stibs_db_top_tables
.EXAMPLE
stibs_db_top_tables -Limit 10
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

    stibs_db_query -Sql $sql
}

<#
.SYNOPSIS
Show total record count across all tables.
.EXAMPLE
stibs_db_total_records
#>
function stibs_db_total_records {
    $cfg = _stibs_db_get_config
    $dbNameLiteral = _stibs_db_escape_sql_literal -Value $cfg.Database

    $sql = @"
SELECT SUM(table_rows) AS total_records
FROM information_schema.tables
WHERE table_schema = '$dbNameLiteral';
"@

    stibs_db_query -Sql $sql
}

<#
.SYNOPSIS
List tables with zero rows.
.EXAMPLE
stibs_db_empty_tables
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

    stibs_db_query -Sql $sql
}

<#
.SYNOPSIS
Show largest tables by disk size in MB.
.PARAMETER Limit
Number of tables to show (default 5, max 1000).
.EXAMPLE
stibs_db_biggest_size
.EXAMPLE
stibs_db_biggest_size -Limit 10
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

    stibs_db_query -Sql $sql
}

<#
.SYNOPSIS
Find columns by name pattern across all tables.
.PARAMETER Column
Column name substring to search for.
.EXAMPLE
stibs_db_find_column -Column "email"
#>
function stibs_db_find_column {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Column
    )

    $cfg = _stibs_db_get_config
    $dbNameLiteral = _stibs_db_escape_sql_literal -Value $cfg.Database
    $columnLike = _stibs_db_escape_like -Value $Column

    $sql = @"
SELECT table_name, column_name
FROM information_schema.columns
WHERE table_schema = '$dbNameLiteral'
AND column_name LIKE '%$columnLike%' ESCAPE '\\';
"@

    stibs_db_query -Sql $sql
}

# -----------------------------------------------------------------------------
# Intelligence
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Show foreign key relationships for a table.
.DESCRIPTION
Returns all foreign keys where the table is either the source or the referenced table.
.PARAMETER Table
Table name to inspect.
.EXAMPLE
stibs_db_fk -Table users
#>
function stibs_db_fk {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Table
    )

    $cfg = _stibs_db_get_config
    $dbNameLiteral = _stibs_db_escape_sql_literal -Value $cfg.Database
    $tableLiteral = _stibs_db_escape_sql_literal -Value $Table

    stibs_db_query -Sql @"
SELECT table_name, column_name, referenced_table_name, referenced_column_name
FROM information_schema.key_column_usage
WHERE table_schema = '$dbNameLiteral'
AND (table_name = '$tableLiteral' OR referenced_table_name = '$tableLiteral');
"@
}

<#
.SYNOPSIS
Show indexes defined on a table.
.PARAMETER Table
Table name to inspect.
.EXAMPLE
stibs_db_indexes -Table users
#>
function stibs_db_indexes {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Table
    )

    $safeTable = _stibs_db_assert_identifier -Value $Table -Name "table"
    stibs_db_query -Sql "SHOW INDEX FROM $safeTable;"
}

<#
.SYNOPSIS
Run EXPLAIN on a SQL query to show execution plan.
.PARAMETER Query
SQL query to explain (without the EXPLAIN keyword).
.EXAMPLE
stibs_db_explain -Query "SELECT * FROM users WHERE id = 1"
#>
function stibs_db_explain {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Query
    )

    stibs_db_query -Sql "EXPLAIN $Query;"
}

<#
.SYNOPSIS
Return a sample of rows from a table.
.PARAMETER Table
Table name to sample.
.PARAMETER Limit
Number of rows to return (default 10, max 1000).
.EXAMPLE
stibs_db_sample -Table users
.EXAMPLE
stibs_db_sample -Table users -Limit 3
#>
function stibs_db_sample {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Table,
        [int]$Limit = 10
    )

    $safeTable = _stibs_db_assert_identifier -Value $Table -Name "table"
    $safeLimit = _stibs_db_assert_limit -Value $Limit -Default 10
    stibs_db_query -Sql "SELECT * FROM $safeTable LIMIT $safeLimit;"
}

<#
.SYNOPSIS
Find orphan rows where the foreign key references a missing parent.
.PARAMETER ChildTable
Table containing the foreign key column.
.PARAMETER ChildColumn
Foreign key column in the child table.
.PARAMETER ParentTable
Referenced parent table.
.PARAMETER ParentColumn
Referenced column in the parent table.
.EXAMPLE
stibs_db_orphans -ChildTable orders -ChildColumn user_id -ParentTable users -ParentColumn id
#>
function stibs_db_orphans {
    param(
        [Parameter(Mandatory = $true)][string]$ChildTable,
        [Parameter(Mandatory = $true)][string]$ChildColumn,
        [Parameter(Mandatory = $true)][string]$ParentTable,
        [Parameter(Mandatory = $true)][string]$ParentColumn
    )

    $safeChildTable  = _stibs_db_assert_identifier -Value $ChildTable  -Name "child table"
    $safeChildColumn = _stibs_db_assert_identifier -Value $ChildColumn -Name "child column"
    $safeParentTable = _stibs_db_assert_identifier -Value $ParentTable -Name "parent table"
    $safeParentColumn = _stibs_db_assert_identifier -Value $ParentColumn -Name "parent column"

    stibs_db_query -Sql @"
SELECT c.*
FROM $safeChildTable c
LEFT JOIN $safeParentTable p
ON c.$safeChildColumn = p.$safeParentColumn
WHERE p.$safeParentColumn IS NULL;
"@
}

<#
.SYNOPSIS
Run a quick health check on the STIBS database.
.DESCRIPTION
Verifies connectivity, counts tables, reports empty tables and largest tables by size.
.EXAMPLE
stibs_db_doctor
#>
function stibs_db_doctor {

    $connection = stibs_db_query -Sql "SELECT 1;"
    $tables     = stibs_db_tables
    $empty      = stibs_db_empty_tables
    $largest    = stibs_db_biggest_size

    return [pscustomobject]@{
        ConnectionOk  = ($null -ne $connection)
        Tables        = $tables
        EmptyTables   = $empty
        LargestBySize = $largest
    }
}
