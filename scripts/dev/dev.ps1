# 开发模式：Go 后端（8000）+ Vite 开发服（5173，带热更新）
#
# 日常使用请双击根目录的「启动.cmd」；这个脚本是给改代码时用的：
# 改前端会自动热更新，不用重新编译 exe。
$ErrorActionPreference = "Continue"

$Here     = $PSScriptRoot                              # scripts\dev
$Root     = Split-Path -Parent (Split-Path -Parent $Here)
$GoDir    = Join-Path $Root "backend-go"
$Frontend = Join-Path $Root "frontend"

function Test-PortListen([int]$Port) {
    try {
        return $null -ne (Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue)
    } catch {
        return $null -ne (netstat -ano | Select-String ":\s*$Port\s+.*LISTENING")
    }
}

Write-Host "==============================================" -ForegroundColor Cyan
Write-Host "   FIRST 物资出入库 - 开发模式" -ForegroundColor Cyan
Write-Host "==============================================" -ForegroundColor Cyan

# 后端：编译后在新窗口运行（保留日志窗口）
if (-not (Test-PortListen 8000)) {
    Write-Host "启动后端 http://127.0.0.1:8000 ..." -ForegroundColor Cyan
    $backendCmd = @"
Set-Location -LiteralPath '$GoDir'
go build -o inventory-server.exe .
if (`$LASTEXITCODE -ne 0) { Write-Host '编译失败' -ForegroundColor Red; Read-Host '按回车关闭'; exit 1 }
.\inventory-server.exe
"@
    # -ExecutionPolicy Bypass：子窗口否则可能拒绝运行未签名脚本
    Start-Process powershell -ArgumentList "-NoExit", "-ExecutionPolicy", "Bypass", "-Command", $backendCmd
    Start-Sleep -Seconds 4
} else {
    Write-Host "端口 8000 已被占用，后端可能已在运行。" -ForegroundColor Yellow
}

if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    Write-Host "找不到 npm，请安装 Node.js 18+：https://nodejs.org/" -ForegroundColor Red
    Write-Host "后端接口仍可用：http://127.0.0.1:8000/api/health" -ForegroundColor Yellow
    Read-Host "按回车关闭"
    exit 1
}

$frontendCmd = @"
Set-Location -LiteralPath '$Frontend'
if (-not (Test-Path node_modules)) { npm install }
npm run dev
"@

Write-Host "启动前端开发服 http://127.0.0.1:5173 ..." -ForegroundColor Cyan
Start-Process powershell -ArgumentList "-NoExit", "-ExecutionPolicy", "Bypass", "-Command", $frontendCmd

Start-Sleep -Seconds 4
Start-Process "http://127.0.0.1:5173/#/checkout"
Write-Host ""
Write-Host "浏览器：http://127.0.0.1:5173/#/checkout" -ForegroundColor Green
Write-Host "改前端会热更新；改后端需要重启那个后端窗口。" -ForegroundColor Gray
