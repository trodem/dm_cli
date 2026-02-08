param(
  [string]$InstallDir = "$env:USERPROFILE\Tools\dm",
  [switch]$NoPathUpdate,
  [switch]$NoCompletion
)

$ErrorActionPreference = "Stop"

function Add-PathIfMissing {
  param([string]$Dir)
  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if ([string]::IsNullOrWhiteSpace($userPath)) {
    [Environment]::SetEnvironmentVariable("Path", $Dir, "User")
    return $true
  }
  $parts = $userPath -split ";"
  foreach ($p in $parts) {
    if ($p.Trim().ToLowerInvariant() -eq $Dir.Trim().ToLowerInvariant()) {
      return $false
    }
  }
  [Environment]::SetEnvironmentVariable("Path", ($userPath.TrimEnd(";") + ";" + $Dir), "User")
  return $true
}

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$exeSource = Join-Path $scriptDir "dm.exe"
$binSource = Join-Path $scriptDir "dm"

if (Test-Path $exeSource) {
  $src = $exeSource
  $dstName = "dm.exe"
} elseif (Test-Path $binSource) {
  $src = $binSource
  $dstName = "dm"
} else {
  throw "Cannot find dm executable next to install.ps1."
}

New-Item -ItemType Directory -Force $InstallDir | Out-Null
$dst = Join-Path $InstallDir $dstName
Copy-Item -Force $src $dst

Write-Host "Installed: $dst"

if (-not $NoPathUpdate) {
  $changed = Add-PathIfMissing -Dir $InstallDir
  if ($changed) {
    Write-Host "PATH updated (User): $InstallDir"
    Write-Host "Open a new terminal to use 'dm' directly."
  } else {
    Write-Host "PATH already contains: $InstallDir"
  }
}

if (-not $NoCompletion -and $dstName -eq "dm.exe") {
  try {
    & $dst completion install | Out-Null
    Write-Host "Completion install attempted (PowerShell)."
  } catch {
    Write-Host "Completion install skipped: $($_.Exception.Message)"
  }
}

Write-Host "Done."
