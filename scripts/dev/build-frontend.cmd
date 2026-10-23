@echo off
chcp 65001 >nul 2>&1
rem 重新构建前端并写入 backend-go/webembed/dist（会被内嵌进 exe）
cd /d "%~dp0..\.."
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0build-frontend.ps1" %*
if errorlevel 1 pause
