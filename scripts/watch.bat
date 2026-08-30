@echo off
REM 同时热重载全部后端服务（Air）
REM 用法：在仓库根目录执行 scripts\watch.bat [all|gateway|user|worker|web|full]
REM
REM 本脚本只是根目录 dev.bat 的兼容入口，实现已统一收口到 dev.bat，
REM 避免两处维护同一份启动逻辑。
setlocal
set "SCRIPT_DIR=%~dp0"
call "%SCRIPT_DIR%..\dev.bat" %*
endlocal
