# FIRST Inventory - Go backend + Vue frontend (dev)
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

if (-not (Test-PortListen 8000)) {
    Write-Host "Starting Go backend on http://127.0.0.1:8000 ..." -ForegroundColor Cyan
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location -LiteralPath '$GoDir'; & '$Root\start-go.ps1'"
    Start-Sleep -Seconds 4
} else {
    Write-Host "Port 8000 in use (backend may already run)." -ForegroundColor Yellow
}

if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    Write-Host "npm not found. Install Node.js: https://nodejs.org/" -ForegroundColor Yellow
    Write-Host "API only: http://127.0.0.1:8000/api/health" -ForegroundColor Yellow
    pause
    exit 0
}

$frontendCmd = @"
Set-Location -LiteralPath '$Frontend'
if (-not (Test-Path node_modules)) { npm install }
npm run dev
"@

Write-Host "Starting frontend http://127.0.0.1:5173 ..." -ForegroundColor Cyan
Start-Process powershell -ArgumentList "-NoExit", "-Command", $frontendCmd

Start-Sleep -Seconds 4
Start-Process "http://127.0.0.1:5173/#/checkout"
Write-Host "Open: http://127.0.0.1:5173/#/checkout"
