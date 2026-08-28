@echo off
echo ===================================================
echo   Building EKA ID Windows Installer (.exe)
echo ===================================================

echo [1/3] Compiling Go backend server.exe...
docker run --rm -v "%~dp0services\api:/app" -w /app golang:1.23-alpine sh -c "GOOS=windows GOARCH=amd64 go build -o server.exe ./cmd/server"
copy /Y "%~dp0services\api\server.exe" "%~dp0apps\desktop\resources\bin\server.exe"

echo [2/3] Building Next.js static export...
cd "%~dp0apps\web"
call npm.cmd run build
xcopy /E /I /Y "%~dp0apps\web\out" "%~dp0apps\desktop\web\out"

echo [3/3] Packaging Windows NSIS Installer...
cd "%~dp0apps\desktop"
call npx.cmd electron-builder --win nsis --x64

echo ===================================================
echo   Installer Ready!
echo   Location: %~dp0apps\desktop\dist\EKA-ID-Setup-1.0.0.exe
echo ===================================================
pause
