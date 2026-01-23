# Bole Drama - Windows Package

## Run

1) Double-click `start.cmd`
2) Open `http://localhost:5678/`

## Notes

- FFmpeg is required (ffmpeg + ffprobe in PATH).
- Config file is created on first run at `configs\config.yaml`.
- Data is stored in `data\` (SQLite + uploads).

## Source Code Included

This package includes the buildable source code under `src\` (excluding `web\node_modules` and `web\dist`).
You can zip and move this folder to another machine; it remains runnable via `start.cmd`.

## Build (from repo)

Run in PowerShell:

```powershell
./build/windows/build.ps1
```
