@echo off
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0start-ai-server.ps1"
set "exit_code=%errorlevel%"
if not "%exit_code%"=="0" (
  echo.
  echo ai-server failed to start. Press any key to close.
  pause >nul
)
exit /b %exit_code%
