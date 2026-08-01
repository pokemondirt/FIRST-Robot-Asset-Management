# FIRST Inventory - backend setup (run once)
$ErrorActionPreference = "Stop"
$Root = $PSScriptRoot
$Backend = Join-Path $Root "backend"
. (Join-Path $Root "scripts\Find-Python.ps1")

Write-Host "=== FIRST Inventory - Backend Setup ===" -ForegroundColor Cyan

$python = Find-PythonExe
if (-not $python) {
    Write-Host ""
    Write-Host "[ERROR] Python 3.11+ not found." -ForegroundColor Red
    Write-Host "Install from: https://www.python.org/downloads/" -ForegroundColor Yellow
    Write-Host "Check: Add python.exe to PATH during install." -ForegroundColor Yellow
    Write-Host "Then close PowerShell, open a new window, run this script again." -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Or try: py -3 -m venv backend\.venv" -ForegroundColor Gray
    pause
    exit 1
}

Write-Host "Using Python: $python" -ForegroundColor Green
Set-Location -LiteralPath $Backend

$venvPy = Join-Path $Backend ".venv\Scripts\python.exe"
if (-not (Test-Path $venvPy)) {
    Write-Host "Creating virtual environment..."
    & $python -m venv .venv
    if (-not (Test-Path $venvPy)) {
        Write-Host "[ERROR] Failed to create .venv" -ForegroundColor Red
        pause
        exit 1
    }
}

Write-Host "Installing packages (first time may take 3-10 min)..."
& $venvPy -m pip install --upgrade pip
& $venvPy -m pip cache purge 2>$null
& $venvPy -m pip install -r requirements.txt --prefer-binary
if ($LASTEXITCODE -ne 0) {
    Write-Host "[WARN] Install failed. Try Python 3.12 from python.org (3.14 is often slower)." -ForegroundColor Yellow
    pause
    exit 1
}

Write-Host ""
Write-Host "Backend setup OK." -ForegroundColor Green
Write-Host 'Next: run .\start.ps1' -ForegroundColor Green
pause
