# Go backend startup script
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location "$scriptDir\backend-go"

# Use Chinese Go module proxy
$env:GOPROXY = "https://goproxy.cn,direct"
$env:GOSUMDB = "sum.golang.google.cn"

Write-Host "Downloading dependencies..." -ForegroundColor Cyan
go mod tidy
if ($LASTEXITCODE -ne 0) {
    Write-Host "go mod tidy failed!" -ForegroundColor Red
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Host "Building..." -ForegroundColor Cyan
go build -o inventory-server.exe .
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Host "Starting server on http://127.0.0.1:8000" -ForegroundColor Green
Write-Host "Dev UI: run start-frontend.ps1 -> http://127.0.0.1:5173/#/checkout" -ForegroundColor Gray
Write-Host "Built UI: run build-frontend.ps1 -> http://127.0.0.1:8000/#/checkout" -ForegroundColor Gray
.\inventory-server.exe
