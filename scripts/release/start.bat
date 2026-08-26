@echo off
rem =====================================================
rem  FIRST Inventory - one-click launcher
rem  Starts the backend server then opens the browser.
rem  Close this window to stop the server.
rem =====================================================
cd /d "%~dp0"

if not exist inventory-server.exe (
    echo [ERROR] inventory-server.exe not found. Re-extract the package.
    pause
    exit /b 1
)

echo Starting FIRST Inventory server...
start "" "http://127.0.0.1:8000/#/checkout"
inventory-server.exe
pause
