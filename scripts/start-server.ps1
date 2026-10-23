#Requires -Version 5.1
<#
  一键启动（由根目录的 启动.cmd 调用，一般不需要直接运行这个文件）

  做的事：
    1. 前端没构建过就先构建（需要 Node.js；已构建过则跳过）
    2. 编译 Go 服务端，前端会被内嵌进 exe
    3. 启动服务，端口就绪后自动打开浏览器

  关掉这个窗口 = 停止服务。
#>
$ErrorActionPreference = "Continue"

$Here    = $PSScriptRoot                        # scripts\
$Root    = Split-Path -Parent $Here             # 仓库根目录
$GoDir   = Join-Path $Root "backend-go"
$Front   = Join-Path $Root "frontend"
$Dist    = Join-Path $Front "dist"
$ExePath = Join-Path $GoDir "inventory-server.exe"
$Opener  = Join-Path $Here "release\open-browser.ps1"
$Port    = 8000

function Test-PortInUse([int]$Port) {
    try {
        return $null -ne (Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue)
    } catch {
        return $null -ne (netstat -ano | Select-String ":\s*$Port\s+.*LISTENING")
    }
}

Write-Host ""
Write-Host "==============================================" -ForegroundColor Cyan
Write-Host "   FIRST 机器人物资出入库系统" -ForegroundColor Cyan
Write-Host "==============================================" -ForegroundColor Cyan
Write-Host ""

# 已经有实例在跑：直接打开浏览器，不要重复启动（端口会冲突）。
if (Test-PortInUse $Port) {
    Write-Host "端口 $Port 已在监听，说明服务已经在运行。" -ForegroundColor Yellow
    Write-Host "正在打开浏览器 ..." -ForegroundColor Yellow
    Start-Process "http://127.0.0.1:$Port/#/checkout"
    Write-Host ""
    Write-Host "如需重启，请先关闭另一个正在运行的服务窗口。" -ForegroundColor Gray
    Read-Host "按回车关闭本窗口"
    exit 0
}

# 1) 前端
if (-not (Test-Path (Join-Path $Dist "index.html"))) {
    Write-Host "[1/3] 第一次运行，正在构建前端（需要 Node.js）..." -ForegroundColor Cyan
    $buildFront = Join-Path $Here "dev\build-frontend.ps1"
    & $buildFront -NoPause
    if (-not (Test-Path (Join-Path $Dist "index.html"))) {
        Write-Host ""
        Write-Host "[错误] 前端构建失败。请确认已安装 Node.js 18+：https://nodejs.org/" -ForegroundColor Red
        Read-Host "按回车退出"
        exit 1
    }
} else {
    Write-Host "[1/3] 前端已构建，跳过。" -ForegroundColor DarkGray
}

# 2) 编译服务端（前端会被内嵌进 exe）
Write-Host "[2/3] 正在编译服务端 ..." -ForegroundColor Cyan
Push-Location -LiteralPath $GoDir
go build -o inventory-server.exe .
$buildCode = $LASTEXITCODE
Pop-Location
if ($buildCode -ne 0) {
    Write-Host ""
    Write-Host "[错误] 编译失败。请确认已安装 Go 1.22+：https://go.dev/dl/" -ForegroundColor Red
    Read-Host "按回车退出"
    exit 1
}

# 3) 启动并打开浏览器
Write-Host "[3/3] 启动服务 ..." -ForegroundColor Cyan
Write-Host ""
if (Test-Path $Opener) {
    try {
        Start-Process powershell -WindowStyle Hidden -ArgumentList @(
            "-NoProfile", "-ExecutionPolicy", "Bypass",
            "-File", "`"$Opener`"", "-Port", "$Port"
        ) | Out-Null
    } catch {
        Write-Host "（自动打开浏览器失败，请手动访问 http://127.0.0.1:$Port/#/checkout）" -ForegroundColor Yellow
    }
}
Write-Host "浏览器地址: http://127.0.0.1:$Port/#/checkout" -ForegroundColor Green
Write-Host "局域网地址会打印在下面的日志里（On this network 那一行）。" -ForegroundColor Gray
Write-Host "关闭本窗口即可停止服务。" -ForegroundColor Gray
Write-Host ""

Push-Location -LiteralPath $GoDir
.\inventory-server.exe
Pop-Location
Read-Host "服务已停止，按回车关闭窗口"
