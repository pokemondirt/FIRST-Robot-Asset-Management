# Start Vue dev server (port 5173) - keep backend window running on 8000
#
# $ErrorActionPreference is "Continue" so the vite dev server is not killed by
# an unrelated npm warning written to stderr.
$ErrorActionPreference = "Continue"
$Frontend = Join-Path $PSScriptRoot "frontend"
Set-Location -LiteralPath $Frontend

if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    Write-Host "[ERROR] npm not found. Install Node.js: https://nodejs.org/" -ForegroundColor Red
    pause
    exit 1
}

if (-not (Test-Path node_modules)) {
    Write-Host "npm install ..."
    npm install
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[ERROR] npm install failed" -ForegroundColor Red
        pause
        exit 1
    }
}

Write-Host "Frontend: http://127.0.0.1:5173/#/checkout" -ForegroundColor Green
Write-Host "Go backend required: .\start-go.ps1 (port 8000)" -ForegroundColor Yellow
npm run dev
