@echo off
rem =====================================================
rem  Create a desktop shortcut to launch the system.
rem =====================================================
cd /d "%~dp0"

powershell -NoProfile -Command "$ws = New-Object -ComObject WScript.Shell; $lnk = $ws.CreateShortcut([Environment]::GetFolderPath('Desktop') + '\FIRST-Inventory.lnk'); $lnk.TargetPath = '%~dp0start.bat'; $lnk.WorkingDirectory = '%~dp0'; $lnk.IconLocation = '%~dp0inventory-server.exe,0'; $lnk.Save()"

echo Desktop shortcut created: FIRST-Inventory
pause
