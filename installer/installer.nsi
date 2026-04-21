!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"

!define APPNAME "Leasing App"
!define COMPANYNAME "Leasing Company"
!define DESCRIPTION "Leasing Application with PostgreSQL, Backend and Frontend"
!define VERSIONMAJOR 1
!define VERSIONMINOR 0
!define VERSIONBUILD 0
!define INSTALLSIZE 500000

Name "${APPNAME}"
OutFile "LeasingApp-Setup.exe"
InstallDir "$PROGRAMFILES\${APPNAME}"
InstallDirRegKey HKLM "Software\${COMPANYNAME}\${APPNAME}" ""
RequestExecutionLevel admin

Var PostgreSQLInstalled

!macro customInstall
  Call CheckPostgreSQL
  ${If} $PostgreSQLInstalled == "false"
    Call InstallPostgreSQL
  ${EndIf}
  Call InitPostgreSQL
  Call BuildBackend
  Call BuildFrontend
  Call CreateShortcuts
!macroend

Section "Install"
  SetOutPath $INSTDIR
  
  File /r "scripts\*.*"
  File /r "resources\*.*"
  
  WriteRegStr HKLM "Software\${COMPANYNAME}\${APPNAME}" "" $INSTDIR
  WriteRegStr HKLM "Software\${COMPANYNAME}\${APPNAME}" "Version" "${VERSIONMAJOR}.${VERSIONMINOR}.${VERSIONBUILD}"
  
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  
  Section "Start Menu Shortcuts"
    CreateDirectory "$SMPROGRAMS\${APPNAME}"
    CreateShortcut "$SMPROGRAMS\${APPNAME}\${APPNAME}.lnk" "$INSTDIR\start.bat"
    CreateShortcut "$SMPROGRAMS\${APPNAME}\Uninstall.lnk" "$INSTDIR\Uninstall.exe"
  SectionEnd
  
  Section "Desktop Shortcut"
    CreateShortcut "$DESKTOP\${APPNAME}.lnk" "$INSTDIR\start.bat"
  SectionEnd
SectionEnd

Section "Uninstall"
  MessageBox MB_YESNO "Remove PostgreSQL database?" IDNO skip_pg
    nsExec::Exec 'net stop postgresql-x64-14'
    RMDir /r "C:\PostgreSQL"
  skip_pg:
  
  RMDir /r "$INSTDIR"
  RMDir /r "$SMPROGRAMS\${APPNAME}"
  Delete "$DESKTOP\${APPNAME}.lnk"
  DeleteRegKey HKLM "Software\${COMPANYNAME}\${APPNAME}"
SectionEnd

Function CheckPostgreSQL
  StrCpy $PostgreSQLInstalled "true"
  IfFileExists "C:\PostgreSQL\14\bin\pg_ctl.exe" check_done
  IfFileExists "C:\Program Files\PostgreSQL\14\bin\pg_ctl.exe" check_done
  StrCpy $PostgreSQLInstalled "false"
check_done:
FunctionEnd

Function InstallPostgreSQL
  DetailPrint "Downloading PostgreSQL..."
  NSISdl::download "https://get.enterprisedb.com/postgresql/postgresql-14.12-1-windows-x64.exe" "$TEMP\postgresql-installer.exe"
  
  DetailPrint "Installing PostgreSQL..."
  ExecWait '"$TEMP\postgresql-installer.exe" --unattendedmodeui none --mode silent --superpassword "postgres" --serviceaccount "NT AUTHORITY\NetworkService"'
  
  DetailPrint "Waiting for PostgreSQL to start..."
  Sleep 5000
FunctionEnd

Function InitPostgreSQL
  DetailPrint "Initializing database..."
  
  SetOutPath "$INSTDIR"
  
  DetailPrint "Creating database 'leasing'..."
  nsExec::ExecToLog '"C:\PostgreSQL\14\bin\psql.exe" -U postgres -c "CREATE DATABASE leasing;"'
FunctionEnd

Function BuildBackend
  DetailPrint "Building backend..."
  
  SetOutPath "$INSTDIR\backend"
  
  ExecWait '"$INSTDIR\scripts\build-go.bat"'
FunctionEnd

Function BuildFrontend
  DetailPrint "Building frontend..."
  
  SetOutPath "$INSTDIR\frontend"
  
  ExecWait '"$INSTDIR\scripts\build-react.bat"'
FunctionEnd

Function CreateShortcuts
  WriteIniStr "$INSTDIR\config.ini" "Settings" "DataDir" "$INSTDIR\data"
FunctionEnd
