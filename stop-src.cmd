@echo off
setlocal

REM Stops backend started from src tree
taskkill /F /IM bole-drama.exe >nul 2>nul
if %ERRORLEVEL% EQU 0 (
  echo stopped.
) else (
  echo not running.
)
