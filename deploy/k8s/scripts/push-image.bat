@echo off
setlocal EnableExtensions
rem ===================================================================
rem Offline image import: docker save locally -> upload to server -> ctr import
rem
rem Usage:
rem   push-image.bat ^<server-ip^> ^<account^> ^<image^> [^<image^>...] [--port=22] [--dir=/docker/images] [--sudo^|--no-sudo] [--skip-key-setup]
rem
rem Examples:
rem   push-image.bat 192.168.142.121 root museflow/web:latest
rem   push-image.bat 192.168.142.121 root museflow/web:latest museflow/api-gateway:latest
rem
rem Notes:
rem   - Keep this file ASCII only. cmd.exe reads batch files with the local code
rem     page, so UTF-8 Chinese comments break line parsing (not just display).
rem   - Arguments must not contain spaces; use backslashes for paths in CMD.
rem   - The password is prompted by ssh/scp: not echoed, never stored.
rem   - The real logic lives in push-image.ps1 next to this file.
rem ===================================================================

set "SCRIPT_DIR=%~dp0"
set "SERVER="
set "ACCOUNT="
set "POS_ARGS="
set "PORT="
set "DIR="
set "SUDO="
set "SKIPKEY="
set /a COUNT=0

:parse
if "%~1"=="" goto run
set "ARG=%~1"
if /I "%ARG%"=="--help" goto usage
if /I "%ARG%"=="-h" goto usage
rem cmd.exe treats = as an argument separator, so an unquoted --dir=images arrives
rem as two arguments (--dir and images); both spellings must be supported.
if /I "%ARG%"=="--port" (
    if "%~2"=="" goto usage
    set "PORT=%~2"
    shift
    shift
    goto parse
)
if /I "%ARG%"=="--dir" (
    if "%~2"=="" goto usage
    set "DIR=%~2"
    shift
    shift
    goto parse
)
set "PREFIX=%ARG:~0,7%"
if /I "%PREFIX%"=="--port=" set "PORT=%ARG:~7%" & shift & goto parse
set "PREFIX=%ARG:~0,6%"
if /I "%PREFIX%"=="--dir=" set "DIR=%ARG:~6%" & shift & goto parse
if /I "%ARG%"=="--sudo" set "SUDO=force" & shift & goto parse
if /I "%ARG%"=="--no-sudo" set "SUDO=never" & shift & goto parse
if /I "%ARG%"=="--skip-key-setup" set "SKIPKEY=1" & shift & goto parse
if "%ARG:~0,1%"=="-" goto unknown
if not defined SERVER set "SERVER=%ARG%" & set "POS_ARGS=%POS_ARGS% %ARG%" & set /a COUNT+=1 & shift & goto parse
if not defined ACCOUNT set "ACCOUNT=%ARG%" & set "POS_ARGS=%POS_ARGS% %ARG%" & set /a COUNT+=1 & shift & goto parse
set "POS_ARGS=%POS_ARGS% %ARG%" & set /a COUNT+=1
shift
goto parse

:run
if %COUNT% LSS 3 goto usage
set "PS_ARGS=%POS_ARGS%"
if defined PORT set "PS_ARGS=%PS_ARGS% -Port %PORT%"
if defined DIR set "PS_ARGS=%PS_ARGS% -RemoteDir %DIR%"
if defined SUDO set "PS_ARGS=%PS_ARGS% -Sudo %SUDO%"
if defined SKIPKEY set "PS_ARGS=%PS_ARGS% -SkipKeySetup"
powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%push-image.ps1" %PS_ARGS%
exit /b %ERRORLEVEL%

:usage
echo Not enough arguments, or you used --help. Need at least ^<server-ip^> ^<account^> ^<image^>.
echo.
echo Usage: push-image.bat ^<server-ip^> ^<account^> ^<image^> [^<image^>...] [options]
echo Options:
echo   --port=22                 SSH port, default 22
echo   --dir=/docker/images      directory on the server, default /docker/images
echo   --sudo / --no-sudo        force / disable remote sudo (default auto detect)
echo   --skip-key-setup          do not offer to install your public key
echo Example: push-image.bat 192.168.142.121 root museflow/web:latest
echo Notes:
echo   - arguments must not contain spaces; use backslashes for paths in CMD
echo   - first run offers to install your SSH public key (one password, then passwordless)
echo   - real logic is in push-image.ps1 next to this file
exit /b 1

:unknown
echo Unknown option: %ARG%
goto usage
