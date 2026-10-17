@echo off
chcp 65001 >nul 2>&1
rem =====================================================
rem  Build the release packages into scripts\release\out
rem  (windows/amd64 + windows/arm64 by default).
rem
rem  Extra arguments are forwarded, e.g.:
rem    build-release.cmd -Targets windows/amd64
rem    build-release.cmd -SkipFrontend
rem =====================================================
cd /d "%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0build-release.ps1" %*
pause
