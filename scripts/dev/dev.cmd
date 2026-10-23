@echo off
chcp 65001 >nul 2>&1
rem 开发模式：Go 后端(8000) + Vite 开发服(5173，热更新)
cd /d "%~dp0..\.."
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0dev.ps1"
if errorlevel 1 pause
