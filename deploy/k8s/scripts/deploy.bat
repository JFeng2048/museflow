@echo off
setlocal EnableExtensions

set "SCOPE=all"
set "KUBECONFIG_PATH="

:parse
if "%~1"=="" goto run
if /I "%~1"=="base" set "SCOPE=base" & shift & goto parse
if /I "%~1"=="app" set "SCOPE=app" & shift & goto parse
if /I "%~1"=="all" set "SCOPE=all" & shift & goto parse
set "ARG=%~1"
rem Keep this file ASCII only: cmd.exe reads batch files with the local code page,
rem so UTF-8 Chinese comments break line parsing (not just display).
rem cmd.exe treats = as an argument separator, so an unquoted --kubeconfig=C:\x.yaml
rem arrives as two arguments; both spellings must be supported.
if /I "%ARG%"=="--kubeconfig" (
    if "%~2"=="" goto usage
    set "KUBECONFIG_PATH=%~2"
    shift
    shift
    goto parse
)
set "PREFIX=%ARG:~0,13%"
if /I "%PREFIX%"=="--kubeconfig=" set "KUBECONFIG_PATH=%ARG:~13%" & shift & goto parse

:usage
echo Usage: deploy.bat [base^|app^|all] [--kubeconfig=C:\path\to\kubeconfig] 1>&2
exit /b 1

:run
set "SCRIPT_DIR=%~dp0"
if defined KUBECONFIG_PATH (
	powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%deploy.ps1" -Scope %SCOPE% -Kubeconfig "%KUBECONFIG_PATH%"
) else (
	powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%deploy.ps1" -Scope %SCOPE%
)
exit /b %ERRORLEVEL%
