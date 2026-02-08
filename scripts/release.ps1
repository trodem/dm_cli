param(
  [Parameter(Mandatory = $true)]
  [string]$Version,
  [string]$TargetOS = "windows",
  [string]$TargetArch = "amd64",
  [switch]$AllTargets,
  [switch]$SkipTests
)

$ErrorActionPreference = "Stop"

if (-not $Version.StartsWith("v")) {
  $Version = "v$Version"
}

$repoRoot = Split-Path -Parent $PSScriptRoot
if ([string]::IsNullOrWhiteSpace($repoRoot)) {
  $repoRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
}
Set-Location $repoRoot

$distRoot = Join-Path $repoRoot "dist"
if (-not (Test-Path $distRoot)) {
  New-Item -ItemType Directory -Path $distRoot | Out-Null
}

if (-not $SkipTests) {
  Write-Host "==> Running tests"
  & go test ./...
  if ($LASTEXITCODE -ne 0) {
    throw "go test failed with exit code $LASTEXITCODE"
  }
}

if ($AllTargets) {
  $targets = @(
    @{ os = "windows"; arch = "amd64" },
    @{ os = "linux"; arch = "amd64" },
    @{ os = "darwin"; arch = "amd64" },
    @{ os = "darwin"; arch = "arm64" }
  )
} else {
  $targets = @(@{ os = $TargetOS; arch = $TargetArch })
}

$prevGOOS = $env:GOOS
$prevGOARCH = $env:GOARCH
try {
  foreach ($target in $targets) {
    $os = $target.os
    $arch = $target.arch
    $artifactName = "dm-$Version-$os-$arch"
    $stageDir = Join-Path $distRoot $artifactName
    $binName = if ($os -eq "windows") { "dm.exe" } else { "dm" }
    $binPath = Join-Path $stageDir $binName
    $zipPath = Join-Path $distRoot "$artifactName.zip"

    if (Test-Path $stageDir) {
      Remove-Item -Recurse -Force $stageDir
    }
    New-Item -ItemType Directory -Path $stageDir | Out-Null

    Write-Host "==> Building $artifactName"
    $env:GOOS = $os
    $env:GOARCH = $arch

    & go build `
      -trimpath `
      -ldflags "-s -w -X cli/internal/app.Version=$Version" `
      -o $binPath `
      .

    if ($LASTEXITCODE -ne 0) {
      throw "go build failed for $artifactName with exit code $LASTEXITCODE"
    }

    if (Test-Path "README.md") {
      Copy-Item "README.md" (Join-Path $stageDir "README.md")
    }
    if (Test-Path "README.txt") {
      Copy-Item "README.txt" (Join-Path $stageDir "README.txt")
    }
    if (Test-Path "LICENSE") {
      Copy-Item "LICENSE" (Join-Path $stageDir "LICENSE")
    }
    if (Test-Path "scripts/install.ps1") {
      Copy-Item "scripts/install.ps1" (Join-Path $stageDir "install.ps1")
    }

    if (Test-Path $zipPath) {
      Remove-Item -Force $zipPath
    }

    Write-Host "==> Packaging $zipPath"
    Compress-Archive -Path (Join-Path $stageDir "*") -DestinationPath $zipPath

    $hash = (Get-FileHash -Algorithm SHA256 $zipPath).Hash.ToLower()
    $hashPath = "$zipPath.sha256"
    Set-Content -Path $hashPath -Value "$hash  $(Split-Path -Leaf $zipPath)"

    Write-Host ""
    Write-Host "Release ready:"
    Write-Host "  Folder : $stageDir"
    Write-Host "  Zip    : $zipPath"
    Write-Host "  SHA256 : $hashPath"
    Write-Host ""
  }
}
finally {
  $env:GOOS = $prevGOOS
  $env:GOARCH = $prevGOARCH
}
