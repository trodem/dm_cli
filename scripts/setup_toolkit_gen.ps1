param(
  [switch]$RunTests,
  [switch]$SkipInstall,
  [string]$OutputPath = "dist/dm-toolkit-gen.exe",
  [string]$InstallPath = "plugins/dm-toolkit-gen.exe",
  [string]$InitName = "",
  [string]$InitPrefix = "",
  [string]$InitCategory = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

$cacheDir = Join-Path $repoRoot ".dm-gocache"
if (-not (Test-Path -LiteralPath $cacheDir)) {
  New-Item -ItemType Directory -Path $cacheDir | Out-Null
}
$env:GOCACHE = $cacheDir

if ($RunTests) {
  Write-Host "==> Running tests"
  & go test ./...
  if ($LASTEXITCODE -ne 0) {
    throw "go test failed with exit code $LASTEXITCODE"
  }
}

$outputAbs = if ([System.IO.Path]::IsPathRooted($OutputPath)) {
  $OutputPath
} else {
  Join-Path $repoRoot $OutputPath
}

$outputDir = Split-Path -Parent $outputAbs
if (-not (Test-Path -LiteralPath $outputDir)) {
  New-Item -ItemType Directory -Path $outputDir | Out-Null
}

Write-Host "==> Building dm-toolkit-gen"
& go build -o $outputAbs ./cmd/dm-toolkit-gen
if ($LASTEXITCODE -ne 0) {
  throw "go build failed with exit code $LASTEXITCODE"
}
Write-Host "Built: $outputAbs"

if (-not $SkipInstall) {
  $installAbs = if ([System.IO.Path]::IsPathRooted($InstallPath)) {
    $InstallPath
  } else {
    Join-Path $repoRoot $InstallPath
  }
  $installDir = Split-Path -Parent $installAbs
  if (-not (Test-Path -LiteralPath $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
  }

  Write-Host "==> Installing in plugins folder"
  Copy-Item -Path $outputAbs -Destination $installAbs -Force
  Write-Host "Installed: $installAbs"
}

$hasInit = (-not [string]::IsNullOrWhiteSpace($InitName)) -or
  (-not [string]::IsNullOrWhiteSpace($InitPrefix)) -or
  (-not [string]::IsNullOrWhiteSpace($InitCategory))

if ($hasInit) {
  if ([string]::IsNullOrWhiteSpace($InitName) -or [string]::IsNullOrWhiteSpace($InitPrefix)) {
    throw "To auto-create a toolkit pass both -InitName and -InitPrefix."
  }

  Write-Host "==> Creating initial toolkit scaffold"
  $initArgs = @("init", "--repo", $repoRoot, "--name", $InitName, "--prefix", $InitPrefix)
  if (-not [string]::IsNullOrWhiteSpace($InitCategory)) {
    $initArgs += @("--category", $InitCategory)
  }

  & $outputAbs @initArgs
  if ($LASTEXITCODE -ne 0) {
    throw "dm-toolkit-gen init failed with exit code $LASTEXITCODE"
  }
}

Write-Host "==> Done"
Write-Host "Usage:"
Write-Host "  $outputAbs validate --repo $repoRoot"
Write-Host "  $outputAbs init --repo $repoRoot --name MSWord --prefix word --category office"
Write-Host "  $outputAbs add --repo $repoRoot --file plugins/functions/office/MSWord_Toolkit.ps1 --prefix word --func export_pdf --param InputPath --param OutputPath --confirm"

