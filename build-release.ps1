# =========================================================
#  build-release.ps1
#  One-click publish. Builds the frontend once, then cross
#  compiles a self-contained package per target platform.
#  The result needs neither Go nor Node.js on the target PC:
#  the Go binary is static (modernc.org/sqlite is pure Go, so
#  CGO stays off) and the UI is served from dist/.
#
#  Usage:
#    .\build-release.ps1
#    .\build-release.ps1 -Targets windows/amd64
#    .\build-release.ps1 -Targets windows/amd64,linux/amd64,darwin/arm64
#
#  Note: this machine's default GOARCH is 386, so the target
#  architecture is always set explicitly. Otherwise a 32-bit
#  build would be shipped by accident.
# =========================================================
[CmdletBinding()]
param(
    # os/arch pairs; windows/* get the .bat launchers, others get the raw binary.
    [string[]]$Targets = @("windows/amd64", "windows/arm64"),
    # Reuse an existing frontend/dist instead of rebuilding it.
    [switch]$SkipFrontend
)

$ErrorActionPreference = "Stop"
$Root   = $PSScriptRoot
$GoDir  = Join-Path $Root "backend-go"
$Front  = Join-Path $Root "frontend"
$RelSrc = Join-Path $Root "scripts\release"
$OutDir = Join-Path $RelSrc "out"

# npm prints user-config warnings on stderr. With $ErrorActionPreference set to
# Stop, PowerShell turns those into terminating errors, so a redirected or piped
# run aborted before the build even started. Only the exit code matters here.
function Invoke-Native {
    param([scriptblock]$Command, [string]$What)
    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        & $Command
        $code = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previous
    }
    if ($code -ne 0) {
        Write-Host "$What failed (exit code $code)" -ForegroundColor Red
        exit 1
    }
}

if (-not $SkipFrontend) {
    Write-Host "== Building frontend ==" -ForegroundColor Cyan
    if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
        Write-Host "npm not found. Install Node.js 18+ or pass -SkipFrontend." -ForegroundColor Red
        exit 1
    }
    Push-Location -LiteralPath $Front
    try {
        if (-not (Test-Path node_modules)) { Invoke-Native { npm install } "npm install" }
        Invoke-Native { npm run build } "Frontend build"
    } finally { Pop-Location }
}

$DistDir = Join-Path $Front "dist"
if (-not (Test-Path (Join-Path $DistDir "index.html"))) {
    Write-Host "frontend/dist/index.html is missing - build the frontend first." -ForegroundColor Red
    exit 1
}

if (Test-Path $OutDir) { Remove-Item -Recurse -Force $OutDir }
New-Item -ItemType Directory -Path $OutDir | Out-Null

# Static binaries only: no cgo means cross compilation just works and the
# target machine needs no runtime installed.
$env:CGO_ENABLED = "0"
$env:GOFLAGS = "-mod=mod"

foreach ($target in $Targets) {
    $parts = $target.Split("/")
    if ($parts.Count -ne 2) { Write-Host "Bad target '$target' (expected os/arch)" -ForegroundColor Red; exit 1 }
    $goos = $parts[0].Trim().ToLower()
    $goarch = $parts[1].Trim().ToLower()

    Write-Host ""
    Write-Host "== Building backend for $goos/$goarch ==" -ForegroundColor Cyan
    $env:GOOS = $goos
    $env:GOARCH = $goarch
    $exeName = if ($goos -eq "windows") { "inventory-server.exe" } else { "first-inventory" }
    $binPath = Join-Path $OutDir $exeName

    Push-Location -LiteralPath $GoDir
    try {
        Invoke-Native { go build -trimpath -ldflags "-s -w" -o $binPath . } "Backend build for $target"
    } finally { Pop-Location }

    $stage = Join-Path $OutDir "stage-$goos-$goarch"
    if (Test-Path $stage) { Remove-Item -Recurse -Force $stage }
    New-Item -ItemType Directory -Path $stage | Out-Null

    Move-Item $binPath (Join-Path $stage $exeName)
    Copy-Item -Recurse $DistDir (Join-Path $stage "dist")
    # MIT requires the licence text to travel with every copy.
    Copy-Item (Join-Path $Root "LICENSE") $stage

    if ($goos -eq "windows") {
        # Windows launchers: double-click start.bat and nothing else is needed.
        Copy-Item (Join-Path $RelSrc "start.bat") $stage
        Copy-Item (Join-Path $RelSrc "open-browser.ps1") $stage
        Copy-Item (Join-Path $RelSrc "install-desktop.bat") $stage
        Copy-Item (Join-Path $RelSrc "README.txt") $stage
    } else {
        # Shared with the GitHub Actions workflow so both stay in sync.
        $portable = Get-Content -Raw (Join-Path $RelSrc "README-portable.txt")
        $portable = $portable.Replace("{{OS}}", $goos).Replace("{{ARCH}}", $goarch)
        Set-Content -LiteralPath (Join-Path $stage "README.txt") -Value $portable -Encoding UTF8
    }

    $zipName = "FIRST-Inventory-$goos-$goarch.zip"
    $zipPath = Join-Path $OutDir $zipName
    if (Test-Path $zipPath) { Remove-Item -Force $zipPath }
    Compress-Archive -Path (Join-Path $stage "*") -DestinationPath $zipPath
    Remove-Item -Recurse -Force $stage

    $sizeMB = [math]::Round((Get-Item $zipPath).Length / 1MB, 1)
    Write-Host "   -> $zipName ($sizeMB MB)" -ForegroundColor Green
}

Remove-Item Env:\GOOS, Env:\GOARCH, Env:\CGO_ENABLED -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "DONE. Packages are in $OutDir" -ForegroundColor Green
Write-Host "Send a zip to the target computer, extract it and run start.bat." -ForegroundColor Green
Write-Host "No Go / Node.js is required there." -ForegroundColor Green
