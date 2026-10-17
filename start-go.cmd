@echo off
chcp 65001 >nul 2>&1
rem =====================================================
rem  Start only the Go backend on http://127.0.0.1:8000
rem  (it also serves frontend\dist when that folder exists).
rem
rem  Uses -ExecutionPolicy Bypass so an unsigned .ps1 still
rem  runs (see start.cmd for the full explanation).
rem =====================================================
cd /d "%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0start-go.ps1" %*
if errorlevel 1 pause
