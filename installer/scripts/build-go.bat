@echo off
setlocal

set INSTALL_DIR=%~dp0..
set BACKEND_DIR=%INSTALL_DIR%\backend
set POSTGRESQL_BIN=C:\PostgreSQL\14\bin

cd /d "%BACKEND_DIR%"

where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo Go is not installed. Please install Go.
    exit /b 1
)

echo Building Go backend...
go build -o "%INSTALL_DIR%\backend\leasing-app.exe" .

if %ERRORLEVEL% EQU 0 (
    echo Backend built successfully.
) else (
    echo Failed to build backend.
    exit /b 1
)

exit /b 0
