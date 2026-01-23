@echo off
setlocal enabledelayedexpansion

cd /d "%~dp0"

if not exist "configs\config.yaml" (
  copy "configs\config.example.yaml" "configs\config.yaml" >nul
)

if not exist "data\storage" (
  mkdir "data\storage" >nul 2>nul
)

where ffmpeg >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
  REM Try to locate ffmpeg installed via winget (Gyan.FFmpeg)
  for /f "delims=" %%F in ('dir /b /s "%LOCALAPPDATA%\Microsoft\WinGet\Packages\Gyan.FFmpeg_*\*\bin\ffmpeg.exe" 2^>nul') do (
    set "FFMPEG_BIN=%%~dpF"
    goto :ffmpeg_found
  )
)

:ffmpeg_found
if not "%FFMPEG_BIN%"=="" (
  set "PATH=%FFMPEG_BIN%;%PATH%"
)

where ffmpeg >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
  echo.
  echo ERROR: ffmpeg not found.
  echo - Quick install (winget): winget install Gyan.FFmpeg
  echo - Or download: https://ffmpeg.org/download.html
  echo.
  pause
  exit /b 1
)

where ffprobe >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
  echo.
  echo ERROR: ffprobe not found (usually comes with ffmpeg).
  echo - Quick install (winget): winget install Gyan.FFmpeg
  echo.
  pause
  exit /b 1
)

echo Starting bole-drama...
bole-drama.exe
