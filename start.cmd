@echo off
chcp 65001 >nul 2>&1
rem =====================================================
rem  Dev launcher: Go backend (8000) + Vite dev server (5173)
rem
rem  Why a .cmd wrapper: Windows may refuse to run an unsigned
rem  .ps1 (execution policy), and some folders additionally trip
rem  PowerShell's "AuthorizationManager check failed" even when
rem  the policy would allow the file. Calling PowerShell with
rem  -ExecutionPolicy Bypass works in both cases.
rem
rem  Usage: double-click, or run  .\start.cmd
rem =====================================================
cd /d "%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0start.ps1" %*
if errorlevel 1 pause
