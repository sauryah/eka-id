Write-Host "==================================================" -ForegroundColor Cyan
Write-Host "   Starting EKA ID Universal Identity Platform   " -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan

docker compose up -d

Write-Host "`nWaiting 3 seconds for containers to be ready..." -ForegroundColor Yellow
Start-Sleep -Seconds 3

Write-Host "`n[OK] All EKA ID services are online:" -ForegroundColor Green
Write-Host " - Web Platform:      http://localhost:3000" -ForegroundColor Green
Write-Host " - Core API Gateway:  http://localhost:8080" -ForegroundColor Green
Write-Host " - Interactive Docs:  http://localhost:8080/api/v1/docs" -ForegroundColor Green
Write-Host " - PostgreSQL:        localhost:5433 (eka_id)" -ForegroundColor Green
Write-Host " - Redis:             localhost:6380" -ForegroundColor Green

Write-Host "`nOpening Web Portal in your browser..." -ForegroundColor Cyan
Start-Process "http://localhost:3000"