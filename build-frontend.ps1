# Build Vue frontend for Go server (static files on port 8000)
#
# $ErrorActionPreference is "Continue" on purpose: npm prints user-config
# warnings on stderr, and with "Stop" PowerShell turns those into terminating
# errors. Failures are detected through $LASTEXITCODE instead.
$ErrorActionPreference = "Continue"
$Frontend = Join-Path $PSScriptRoot "frontend"
Set-Location -LiteralPath $Frontend

if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    Write-Host "[ERROR] Install Node.js: https://nodejs.org/" -ForegroundColor Red
    pause
    exit 1
}

if (-not (Test-Path node_modules)) {
    npm install
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[ERROR] npm install failed" -ForegroundColor Red
        pause
        exit 1
    }
}

npm run build
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] Frontend build failed" -ForegroundColor Red
    pause
    exit 1
}

Write-Host ""
Write-Host "Done. Start Go server: .\start-go.ps1" -ForegroundColor Green
Write-Host "Open: http://127.0.0.1:8000/#/checkout" -ForegroundColor Green
pause
