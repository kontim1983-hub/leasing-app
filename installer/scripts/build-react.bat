@echo off
setlocal

set INSTALL_DIR=%~dp0..
set FRONTEND_DIR=%INSTALL_DIR%\frontend

cd /d "%FRONTEND_DIR%"

where npm >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo Node.js/npm is not installed. Please install Node.js.
    exit /b 1
)

echo Installing frontend dependencies...
call npm install --legacy-peer-deps

if %ERRORLEVEL% NEQ 0 (
    echo Failed to install dependencies.
    exit /b 1
)

echo Building frontend...
call npm run build

if %ERRORLEVEL% EQU 0 (
    echo Frontend built successfully.
    if not exist "%INSTALL_DIR%\frontend\build" (
        echo Build directory not found.
        exit /b 1
    )
) else (
    echo Failed to build frontend.
    exit /b 1
)

exit /b 0
