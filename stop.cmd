@echo off
setlocal

taskkill /F /IM bole-drama.exe >nul 2>nul
if %ERRORLEVEL% EQU 0 (
  echo stopped.
) else (
  echo not running.
)
