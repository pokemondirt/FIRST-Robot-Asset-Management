# 构建前端，并把它拷进 Go 模块以便内嵌进 exe。
#
# 一般不需要手动运行；根目录的「启动.cmd」在第一次启动时会自动调用它
# （带 -NoPause）。改了前端想立刻看到效果时，开发模式用 scripts\dev\dev.cmd。
#
# $ErrorActionPreference 用 "Continue"：npm 会往 stderr 打用户配置警告，
# 用 "Stop" 会把这些警告当成错误中断脚本。失败通过 $LASTEXITCODE 判断。
param(
    [switch]$NoPause
)

$ErrorActionPreference = "Continue"

$Here     = $PSScriptRoot                          # scripts\dev
$Root     = Split-Path -Parent (Split-Path -Parent $Here)
$Frontend = Join-Path $Root "frontend"

function Stop-WithPause([string]$Message) {
    Write-Host "[错误] $Message" -ForegroundColor Red
    if (-not $NoPause) { pause }
    exit 1
}

Set-Location -LiteralPath $Frontend

if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    Stop-WithPause "找不到 npm，请安装 Node.js 18+：https://nodejs.org/"
}

if (-not (Test-Path node_modules)) {
    Write-Host "安装前端依赖（第一次需要几分钟）..." -ForegroundColor Cyan
    npm install
    if ($LASTEXITCODE -ne 0) { Stop-WithPause "npm install 失败" }
}

Write-Host "构建前端 ..." -ForegroundColor Cyan
npm run build
if ($LASTEXITCODE -ne 0) { Stop-WithPause "前端构建失败" }

# 拷进 Go 模块，让 go build 内嵌进去。保留 .gitkeep，
# 这样还没构建过前端时 go build 也能通过。
Write-Host ""
Write-Host "写入 backend-go/webembed/dist ..." -ForegroundColor Cyan
$EmbedDir = Join-Path $Root "backend-go\webembed\dist"
New-Item -ItemType Directory -Force -Path $EmbedDir | Out-Null
Get-ChildItem -LiteralPath $EmbedDir -Force |
    Where-Object { $_.Name -ne '.gitkeep' } |
    Remove-Item -Recurse -Force
Copy-Item (Join-Path $Frontend "dist\*") $EmbedDir -Recurse -Force

Write-Host ""
Write-Host "完成。前端已内嵌，编译并启动：" -ForegroundColor Green
Write-Host "  ..\..\启动.cmd" -ForegroundColor Green
if (-not $NoPause) { pause }
