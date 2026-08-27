@echo off
REM 启动 api-gateway 热重载（Air）
REM 用法：在仓库根目录执行 scripts\watch-gateway.bat
setlocal
PATH %PATH%;%USERPROFILE%\go\bin
set SCRIPT_DIR=%~dp0
pushd "%SCRIPT_DIR%..\services\api-gateway"
if not exist "..\..\.env" (
  echo [WARN] .env not found, config falls back to defaults / system env
)
air
popd
endlocal
