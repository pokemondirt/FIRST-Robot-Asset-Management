@echo off
chcp 65001 >nul 2>&1
rem =====================================================
rem  FIRST Inventory - one-click launcher
rem  Starts the backend server, opens the browser once the
rem  server is up, and stays in the foreground so closing
rem  this window stops the server.
rem
rem  The server listens on every network interface, so other
rem  computers on the same LAN can open the LAN address that
rem  the server prints below.
rem =====================================================
cd /d "%~dp0"
title FIRST Inventory Server

if not exist inventory-server.exe (
    echo [错误] 找不到 inventory-server.exe，请重新解压压缩包。
    echo [ERROR] inventory-server.exe not found. Re-extract the package.
    pause
    exit /b 1
)

rem Open the browser in the background; it waits for port 8000 itself.
start "" /b powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0open-browser.ps1"

echo.
echo   FIRST 机器人出入库系统 已启动 / FIRST Inventory is starting...
echo.
echo   本机 / this computer : http://127.0.0.1:8000/#/checkout
echo   局域网 / LAN         : 见下面日志中的 "On this network" 一行
echo                          see the "On this network" line below
echo.
echo   关闭本窗口即停止服务 / Close this window to stop the server.
echo.

inventory-server.exe
pause
