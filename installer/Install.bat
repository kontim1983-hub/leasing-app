@echo off
setlocal EnableDelayedExpansion

set "INSTALL_DIR=%USERPROFILE%\LeasingApp"
set "APP_NAME=Leasing App"
set "VERSION=1.0.0"
set "PROJECT_ROOT=%~dp0.."

echo ================================================
echo   %APP_NAME% Portable Installer v%VERSION%
echo ================================================
echo.

echo [1/5] Creating directories...
if not exist "%INSTALL_DIR%\backend" mkdir "%INSTALL_DIR%\backend"
if not exist "%INSTALL_DIR%\frontend" mkdir "%INSTALL_DIR%\frontend"

echo [2/5] Copying backend...
if exist "%PROJECT_ROOT%\backend\leasing-app.exe" (
    copy /y "%PROJECT_ROOT%\backend\leasing-app.exe" "%INSTALL_DIR%\backend\" >nul
    echo    OK
) else (
    echo    [ERROR] Backend not found!
)

echo [3/5] Copying frontend...
if exist "%PROJECT_ROOT%\frontend\build" (
    xcopy /s /y "%PROJECT_ROOT%\frontend\build\*.*" "%INSTALL_DIR%\frontend\" >nul 2>&1
) else (
    if exist "%PROJECT_ROOT%\frontend\index.html" (
        xcopy /s /y "%PROJECT_ROOT%\frontend\*.*" "%INSTALL_DIR%\frontend\" >nul 2>&1
    ) else (
        echo    [ERROR] Frontend not found!
    )
)
echo    OK

echo [4/5] Copying start script...
copy /y "%PROJECT_ROOT%\installer\scripts\start.bat" "%INSTALL_DIR%\" >nul

echo [5/5] Creating shortcuts...
powershell -Command "$WshShell = New-Object -comObject WScript.Shell; $Shortcut = $WshShell.CreateShortcut('%USERPROFILE%\Desktop\Leasing App.lnk'); $Shortcut.TargetPath = '%INSTALL_DIR%\start.bat'; $Shortcut.WorkingDirectory = '%INSTALL_DIR%'; $Shortcut.Description = 'Start Leasing App'; $Shortcut.Save()"

if not exist "%APPDATA%\Microsoft\Windows\Start Menu\Programs" (
    mkdir "%APPDATA%\Microsoft\Windows\Start Menu\Programs"
)
powershell -Command "$WshShell = New-Object -comObject WScript.Shell; $Shortcut = $WshShell.CreateShortcut('%APPDATA%\Microsoft\Windows\Start Menu\Programs\Leasing App.lnk'); $Shortcut.TargetPath = '%INSTALL_DIR%\start.bat'; $Shortcut.WorkingDirectory = '%INSTALL_DIR%'; $Shortcut.Description = 'Start Leasing App'; $Shortcut.Save()"

echo.
echo ================================================
echo   Installation Complete!
echo ================================================
echo.
echo   Installation directory: %INSTALL_DIR%
echo.
echo   Run 'Leasing App' from Desktop to start.
echo.
echo   Frontend: http://localhost:3000
echo   Backend:  http://localhost:8080
echo ================================================

pause
