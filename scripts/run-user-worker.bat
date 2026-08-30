@echo off
REM 启动 user-service 的异步任务 Worker（asynq 消费端）
REM 用法：在仓库根目录执行 scripts\run-user-worker.bat
REM 注意：Worker 与 gRPC 服务是两个独立进程，需分别启动
setlocal
set SCRIPT_DIR=%~dp0
pushd "%SCRIPT_DIR%..\services\user-service"
if not exist "..\..\.env" (
  echo [WARN] .env not found, config falls back to defaults / system env
)
go run ./cmd/worker
popd
endlocal
