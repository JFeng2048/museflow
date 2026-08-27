@echo off
REM 启动 api-gateway（go run，无热重载）
REM 用法：在仓库根目录执行 scripts\run-gateway.bat
setlocal
set SCRIPT_DIR=%~dp0
pushd "%SCRIPT_DIR%..\services\api-gateway"
if not exist "..\..\.env" (
  echo [WARN] .env not found, config falls back to defaults / system env
)
go run ./cmd/server
popd
endlocal
