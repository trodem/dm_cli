# Pre-push check: runs tests and linter.
# Usage: .\scripts\pre-push.ps1
# Or configure as git hook (see below).

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "=== pre-push: go test ===" -ForegroundColor Cyan
go test ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "Tests failed. Push aborted." -ForegroundColor Red
    exit 1
}

Write-Host "=== pre-push: golangci-lint ===" -ForegroundColor Cyan
if (Get-Command golangci-lint -ErrorAction SilentlyContinue) {
    golangci-lint run
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Lint failed. Push aborted." -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "golangci-lint not installed, skipping (CI will catch lint errors)." -ForegroundColor Yellow
}

Write-Host "=== pre-push: all checks passed ===" -ForegroundColor Green
