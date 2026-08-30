@echo off
REM ============================================================
REM  MuseFlow 开发环境一键热重载（仓库根目录入口）
REM
REM  用法（在仓库根目录双击或执行）：
REM    dev.bat            启动全部后端服务（默认，3 个窗口）
REM    dev.bat gateway    仅 api-gateway（HTTP :5001）
REM    dev.bat user       仅 user-service（gRPC :5002）
REM    dev.bat worker     仅 user-service worker（asynq 消费端，无端口）
REM    dev.bat web        仅前端（web\pnpm dev）
REM    dev.bat full       后端全部 + 前端
REM    dev.bat help       显示帮助
REM
REM  每个服务在独立窗口中运行各自的 air，互不影响；
REM  关闭对应窗口即可停止该服务（或窗口内 Ctrl+C）。
REM ============================================================
setlocal EnableDelayedExpansion

set "ROOT=%~dp0"
if "%ROOT:~-1%"=="\" set "ROOT=%ROOT:~0,-1%"

REM 把 Go 工具目录加入 PATH，保证能找到 air
set "PATH=%PATH%;%USERPROFILE%\go\bin"

set "TARGET=%~1"
if "%TARGET%"=="" set "TARGET=all"

if /i "%TARGET%"=="help" goto :help
if /i "%TARGET%"=="-h" goto :help
if /i "%TARGET%"=="--help" goto :help
if /i "%TARGET%"=="/?" goto :help

echo.
echo   MuseFlow dev launcher
echo   root : %ROOT%
echo.

REM air 只用于 Go 服务；纯前端模式（web）不需要
if /i not "%TARGET%"=="web" (
  where air >nul 2>&1
  if errorlevel 1 (
    echo   [ERROR] air not found. Install it first:
    echo           go install github.com/air-verse/air@latest
    echo.
    exit /b 1
  )
)

if not exist "%ROOT%\.env" (
  echo   [WARN] %ROOT%\.env not found; services fall back to defaults / system env.
  echo         Copy .env.example to .env and fill in the values.
  echo.
)

set "STARTED=0"

if /i "%TARGET%"=="all"      call :start_gateway & call :start_user & call :start_worker & goto :done
if /i "%TARGET%"=="full"     call :start_gateway & call :start_user & call :start_worker & call :start_web & goto :done
if /i "%TARGET%"=="gateway"  call :start_gateway & goto :done
if /i "%TARGET%"=="user"     call :start_user    & goto :done
if /i "%TARGET%"=="worker"   call :start_worker  & goto :done
if /i "%TARGET%"=="web"      call :start_web     & goto :done

echo   [ERROR] unknown target: %TARGET%
echo.
goto :help

REM ---------------- 各服务启动例程 ----------------

:start_gateway
call :launch "api-gateway" "%ROOT%\services\api-gateway" "" "HTTP :5001"
exit /b 0

:start_user
call :launch "user-service" "%ROOT%\services\user-service" "" "gRPC :5002"
exit /b 0

:start_worker
REM Worker 用独立配置（.air.worker.toml），入口为 cmd/worker
call :launch "user-service-worker" "%ROOT%\services\user-service" ".air.worker.toml" "asynq consumer, no port"
exit /b 0

:start_web
if not exist "%ROOT%\web\package.json" (
  echo   [WARN] web\package.json not found, skip frontend.
  exit /b 0
)
echo   -> web (Vite dev server)
start "web" /D "%ROOT%\web" cmd /k pnpm dev
set /a STARTED+=1
exit /b 0

REM launch <窗口标题> <服务目录> <air 配置（可空）> <说明>
REM 用 start /D 指定工作目录，避免在命令行里拼接 cd 与 &&（引号嵌套易出错）
:launch
set "TITLE=%~1"
set "DIR=%~2"
set "CFG=%~3"
set "DESC=%~4"
set "AIRCMD=air"
if not "%CFG%"=="" set "AIRCMD=air -c %CFG%"

echo   -> %TITLE% (%DESC%)
start "%TITLE%" /D "%DIR%" cmd /k %AIRCMD%
set /a STARTED+=1
exit /b 0

REM ---------------- 收尾与帮助 ----------------

:done
echo.
echo   Started %STARTED% process(es) in separate windows.
echo   Close each window (or Ctrl+C inside it) to stop that service.
echo.
endlocal
exit /b 0

:help
echo   Usage: dev.bat [all/gateway/user/worker/web/full/help]
echo.
echo     all       api-gateway + user-service + user-service worker (default)
echo     gateway   api-gateway only          (HTTP  :5001)
echo     user      user-service only         (gRPC  :5002)
echo     worker    user-service worker only  (asynq consumer, no port)
echo     web       frontend only             (pnpm dev)
echo     full      backend all + frontend
echo     help      show this message
echo.
echo   Requires air: go install github.com/air-verse/air@latest
echo.
endlocal
exit /b 0
