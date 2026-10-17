@echo off
chcp 65001 >nul 2>&1
rem =====================================================
rem  可选：扫码时自动把输入法切到英文
rem
rem  只在「出入库」页面的扫码框获得焦点时生效；
rem  光标离开扫码框后不再干预。Ctrl+C 停止。
rem =====================================================
cd /d "%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0ime-english-on-scan.ps1" %*
pause
