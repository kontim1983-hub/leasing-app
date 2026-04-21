@echo off
setlocal EnableDelayedExpansion

set "INSTALL_DIR=%~dp0"
set "DATA_DIR=%INSTALL_DIR%\data"
set "LOGS_DIR=%INSTALL_DIR%\logs"

if not exist "%DATA_DIR%" mkdir "%DATA_DIR%"
if not exist "%LOGS_DIR%" mkdir "%LOGS_DIR%"

echo ================================================
echo   Starting %APP_NAME%
echo ================================================
echo.

if exist "C:\PostgreSQL\14\bin\pg_ctl.exe" (
    echo [1/3] Starting PostgreSQL...
    net start postgresql-x64-14 >nul 2>&1
    echo    PostgreSQL started.
) else (
    echo [WARNING] PostgreSQL not found. Please install PostgreSQL 14.
)

echo [2/3] Starting Backend...
if exist "%INSTALL_DIR%\backend\leasing-app.exe" (
    start "Leasing Backend" cmd /k "cd /d "%INSTALL_DIR%\backend" && leasing-app.exe"
    echo    Backend started on http://localhost:8080
) else (
    echo    [ERROR] Backend executable not found.
    echo    Please rebuild: Run Install.bat again
)

echo [3/3] Starting Frontend...
if exist "%INSTALL_DIR%\frontend\static" (
    start "Leasing Frontend" cmd /k "cd /d "%INSTALL_DIR%\frontend" && npx serve -s . -l 3000"
    echo    Frontend started on http://localhost:3000
) else (
    if exist "%INSTALL_DIR%\frontend\index.html" (
        start "Leasing Frontend" cmd /k "cd /d "%INSTALL_DIR%\frontend" && npx serve -s . -l 3000"
        echo    Frontend started on http://localhost:3000
    ) else (
        echo    [ERROR] Frontend files not found.
    )
)

echo.
echo ================================================
echo   Leasing App is running!
echo.
echo   Frontend: http://localhost:3000
echo   Backend:  http://localhost:8080
echo ================================================
echo.
echo   Press any key to close this window...
timeout /t 2 /nobreak >nul
