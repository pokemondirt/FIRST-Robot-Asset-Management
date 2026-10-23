@echo off
chcp 65001 >nul 2>&1
rem 打发布包（默认 windows/amd64 + windows/arm64），输出到 scripts\release\out
rem 可传参，例如：build-release.cmd -Targets windows/amd64
cd /d "%~dp0..\.."
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0build-release.ps1" %*
pause
