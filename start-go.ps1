# Go backend startup script
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location "$scriptDir\backend-go"

# Use Chinese Go module proxy
$env:GOPROXY = "https://goproxy.cn,direct"
$env:GOSUMDB = "sum.golang.google.cn"

# No `go mod tidy` here: it rewrites go.mod/go.sum on every start (dirtying the
# working tree) and needs the network. `go build` downloads what is missing and
# leaves the manifests alone.
Write-Host "Building..." -ForegroundColor Cyan
go build -o inventory-server.exe .
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed! Check your Go installation and network." -ForegroundColor Red
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Host "Starting server on http://127.0.0.1:8000 (LAN: see the log below)" -ForegroundColor Green
Write-Host "Dev UI: run start-frontend.ps1 -> http://127.0.0.1:5173/#/checkout" -ForegroundColor Gray
Write-Host "Built UI: run build-frontend.ps1 -> http://127.0.0.1:8000/#/checkout" -ForegroundColor Gray
.\inventory-server.exe
