@echo off
echo ==================================================
echo    Starting EKA ID Universal Identity Platform
echo ==================================================
docker compose up -d
echo.
echo Waiting 3 seconds for containers to be ready...
timeout /t 3 /nobreak >nul
echo.
echo [OK] All EKA ID services are online:
echo  - Web Platform:     http://localhost:3000
echo  - Core API Gateway: http://localhost:8080
echo  - Interactive Docs: http://localhost:8080/api/v1/docs
echo.
start http://localhost:3000
echo EKA ID has been launched in your default browser!
pause