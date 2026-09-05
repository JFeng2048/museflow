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
set "PREFIX=%ARG:~0,13%"
if /I "%PREFIX%"=="--kubeconfig=" set "KUBECONFIG_PATH=%ARG:~13%" & shift & goto parse
echo 用法: deploy.bat [base^|app^|all] [--kubeconfig=C:\path\to\kubeconfig] 1>&2
exit /b 1

:run
set "SCRIPT_DIR=%~dp0"
if defined KUBECONFIG_PATH (
	powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%deploy.ps1" -Scope %SCOPE% -Kubeconfig "%KUBECONFIG_PATH%"
) else (
	powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%deploy.ps1" -Scope %SCOPE%
)
exit /b %ERRORLEVEL%
