# FIRST Inventory - single port (8000): build frontend, serve via Go backend
$ErrorActionPreference = "Continue"
$Root = $PSScriptRoot
$GoDir = Join-Path $Root "backend-go"
$Frontend = Join-Path $Root "frontend"

function Test-PortListen([int]$Port) {
    try {
        return $null -ne (Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue)
    } catch {
        return (netstat -ano | Select-String ":\s*$Port\s+.*LISTENING")
    }
}

if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    Write-Host "[ERROR] npm not found. Install Node.js: https://nodejs.org/" -ForegroundColor Red
    pause
    exit 1
}

Set-Location -LiteralPath $Frontend
if (-not (Test-Path node_modules)) { npm install }
Write-Host "Building frontend ..." -ForegroundColor Cyan
npm run build
if ($LASTEXITCODE -ne 0) {
    Write-Host "Frontend build failed!" -ForegroundColor Red
    pause
    exit 1
}

if (-not (Test-PortListen 8000)) {
    Write-Host "Starting Go backend on http://127.0.0.1:8000 ..." -ForegroundColor Cyan
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location -LiteralPath '$GoDir'; & '$Root\start-go.ps1'"
    Start-Sleep -Seconds 5
} else {
    Write-Host "Port 8000 in use (backend may already run)." -ForegroundColor Yellow
}

Start-Process "http://127.0.0.1:8000/#/checkout"
Write-Host "Open: http://127.0.0.1:8000/#/checkout" -ForegroundColor Green
