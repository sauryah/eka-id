Write-Host "=== Starting EKA ID Local Development Stack ===" -ForegroundColor Cyan
docker compose up -d
Write-Host "PostgreSQL running on localhost:5433" -ForegroundColor Green
Write-Host "Redis running on localhost:6380" -ForegroundColor Green
Write-Host "API Gateway running on http://localhost:8080" -ForegroundColor Green
Write-Host "Next.js Web Portal: cd apps/web && npm.cmd run dev" -ForegroundColor Yellow