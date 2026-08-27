@echo off
REM 同时热重载两个服务（Air）
REM 用法：在仓库根目录执行 scripts\watch.bat
setlocal
PATH %PATH%;%USERPROFILE%\go\bin
set SCRIPT_DIR=%~dp0

start "user-service" cmd /k "cd /d %SCRIPT_DIR%..\services\user-service && air"
start "api-gateway" cmd /k "cd /d %SCRIPT_DIR%..\services\api-gateway && air"

echo Started two watch windows: user-service(5002) and api-gateway(5001)
echo Close the windows to stop the services.
endlocal
