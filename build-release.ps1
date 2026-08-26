# =========================================================
#  build-release.ps1
#  One-click publish: build frontend + backend, assemble a
#  self-contained package (no Go/Node needed on the target)
#  and produce FIRST-Inventory.zip in scripts/release/
# =========================================================
$ErrorActionPreference = "Stop"
$Root    = $PSScriptRoot
$GoDir   = Join-Path $Root "backend-go"
$Front   = Join-Path $Root "frontend"
$RelDir  = Join-Path $Root "scripts\release"

Write-Host "== Building frontend ==" -ForegroundColor Cyan
Set-Location -LiteralPath $Front
if (-not (Test-Path node_modules)) { npm install }
npm run build
if ($LASTEXITCODE -ne 0) { Write-Host "Frontend build failed" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "== Building backend ==" -ForegroundColor Cyan
Set-Location -LiteralPath $GoDir
go build -ldflags "-s -w" -o inventory-server.exe .
if ($LASTEXITCODE -ne 0) { Write-Host "Backend build failed" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "== Assembling package ==" -ForegroundColor Cyan
$Stage = Join-Path $RelDir "stage"
if (Test-Path $Stage) { Remove-Item -Recurse -Force $Stage }
New-Item -ItemType Directory -Path $Stage | Out-Null

# app files
Copy-Item (Join-Path $GoDir "inventory-server.exe") $Stage
Copy-Item -Recurse (Join-Path $Front "dist") (Join-Path $Stage "dist")
Copy-Item (Join-Path $RelDir "start.bat") $Stage
Copy-Item (Join-Path $RelDir "install-desktop.bat") $Stage
Copy-Item (Join-Path $RelDir "README.txt") $Stage

# data folder (so a fresh install is ready; keep existing data if any copied in later)
New-Item -ItemType Directory -Path (Join-Path $Stage "data") | Out-Null

Write-Host "== Compressing ==" -ForegroundColor Cyan
$Zip = Join-Path $RelDir "FIRST-Inventory.zip"
if (Test-Path $Zip) { Remove-Item -Force $Zip }
Compress-Archive -Path (Join-Path $Stage "*") -DestinationPath $Zip
Remove-Item -Recurse -Force $Stage

Write-Host ""
Write-Host "DONE: $Zip" -ForegroundColor Green
Write-Host "Send this zip to the target computer, extract and run start.bat. No Go/Node needed." -ForegroundColor Green
