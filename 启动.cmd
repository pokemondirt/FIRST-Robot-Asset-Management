@echo off
chcp 65001 >nul 2>&1
rem =====================================================
rem  FIRST 机器人物资出入库系统 - 一键启动
rem
rem  双击本文件即可：自动构建前端（如未构建）、
rem  编译服务端、启动并打开浏览器。
rem
rem  关闭弹出的窗口 = 停止服务。
rem =====================================================
cd /d "%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\start-server.ps1"
if errorlevel 1 pause
