@echo off
setlocal EnableExtensions
rem bhuppi.cmd — cmd.exe shim so `folders go` can cd this window.
rem The real binary lives beside this script or under %LOCALAPPDATA%\bhuppi\lib.
set "BIN=%~dp0bhuppi.exe"
if not exist "%BIN%" set "BIN=%~dp0..\lib\bhuppi.exe"
if not exist "%BIN%" set "BIN=%LOCALAPPDATA%\bhuppi\lib\bhuppi.exe"

if /I "%~1"=="--dry-run" goto run
if /I "%~1"=="--help" goto run
if /I "%~1"=="-h" goto run
if /I "%~1"=="--plain" goto run

if /I "%~1"=="folders" if /I "%~2"=="go" if not "%~3"=="" (
  cd /d "%~3"
  exit /b %errorlevel%
)
if /I "%~1"=="files" if /I "%~2"=="go" if not "%~3"=="" (
  cd /d "%~3"
  exit /b %errorlevel%
)

:run
if not exist "%BIN%" (
  echo bhuppi.exe is not installed. Run install.cmd from the bhuppi folder. 1>&2
  exit /b 2
)
"%BIN%" %*
exit /b %errorlevel%
