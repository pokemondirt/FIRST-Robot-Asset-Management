@echo off
chcp 65001 >nul 2>&1
rem =====================================================
rem  Build the frontend into frontend\dist so the Go server
rem  can serve it on a single port.
rem =====================================================
cd /d "%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0build-frontend.ps1" %*
if errorlevel 1 pause
