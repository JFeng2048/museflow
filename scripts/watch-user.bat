@echo off
REM 启动 user-service 热重载（Air）
REM 用法：在仓库根目录执行 scripts\watch-user.bat
setlocal
PATH %PATH%;%USERPROFILE%\go\bin
set SCRIPT_DIR=%~dp0
pushd "%SCRIPT_DIR%..\services\user-service"
if not exist "..\..\.env" (
  echo [WARN] .env not found, config falls back to defaults / system env
)
air
popd
endlocal
