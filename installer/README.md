# Leasing App Installer

## Overview

This folder contains the installer files for the Leasing App application.

## Files

- `Install.bat` - Main installer script (runs with admin privileges)
- `installer.nsi` - NSIS installer script (requires NSIS to compile)
- `scripts/` - Build and startup scripts
- `resources/` - Configuration files

## Requirements

- Windows 10/11 (64-bit)
- PostgreSQL 14 (automatically detected, can be installed separately)
- Go 1.20+ (for building backend)
- Node.js 18+ (for building frontend)

## Quick Start

### Option 1: Batch Installer (Recommended)

1. Copy the entire `installer` folder to target Windows machine
2. Also copy `backend` and `frontend` folders
3. Run `Install.bat` as Administrator
4. Launch from Desktop shortcut

### Option 2: NSIS Installer

1. Install NSIS from https://nsis.sourceforge.io/
2. Compile `installer.nsi`
3. Run the generated `LeasingApp-Setup.exe`

## Post-Installation

After installation, you can start the application using:
- Desktop shortcut
- Start Menu shortcut
- Manual: Run `%INSTALL_DIR%\scripts\start.bat`

## Uninstall

To uninstall:
1. Stop all running services (Backend, Frontend)
2. Stop PostgreSQL: `net stop postgresql-x64-14`
3. Delete installation directory: `%PROGRAMFILES%\LeasingApp`
4. Remove shortcuts from Desktop and Start Menu

## Troubleshooting

### PostgreSQL Connection Issues
```
1. Ensure PostgreSQL service is running: net start postgresql-x64-14
2. Check credentials in backend environment or config
3. Default: Host=localhost, Port=5432, User=postgres, Password=postgres
```

### Frontend Not Loading
```
1. Ensure Node.js is installed
2. Rebuild: Run Install.bat again
3. Check port 3000 is available
```

### Backend Errors
```
1. Ensure Go is installed
2. Rebuild: Run Install.bat again
3. Check logs in %INSTALL_DIR%\logs
```
