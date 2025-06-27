@echo off
echo ========================================
echo   News Aggregator Services Launcher
echo ========================================
echo.

echo Checking Docker containers...
docker ps

echo.
echo Starting PostgreSQL if not running...
docker ps | findstr postgres >nul
if errorlevel 1 (
    echo PostgreSQL not found, starting...
    docker run -d --name postgres -e POSTGRES_DB=newsdb -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=123456 -p 5432:5432 postgres:15
    timeout /t 10 /nobreak >nul
) else (
    echo PostgreSQL already running
)

echo.
echo Starting Redis if not running...
docker ps | findstr redis >nul
if errorlevel 1 (
    echo Redis not found, starting...
    docker run -d --name redis -p 6379:6379 redis:7-alpine
    timeout /t 3 /nobreak >nul
) else (
    echo Redis already running
)

echo.
echo ========================================
echo   Starting Go Services
echo ========================================
echo.

echo Starting Auth Service (Port 8083)...
start "Auth Service" cmd /k "echo Auth Service Starting... && go run ./cmd/auth-service/main.go"
timeout /t 2 /nobreak >nul

echo Starting News API (Port 8081)...
start "News API" cmd /k "echo News API Starting... && go run ./cmd/news-api/main.go"
timeout /t 2 /nobreak >nul

echo Starting News Scraper (Port 8082)...
start "News Scraper" cmd /k "echo News Scraper Starting... && go run ./cmd/news-scraper/main.go"
timeout /t 2 /nobreak >nul

echo Starting API Gateway (Port 8080)...
start "API Gateway" cmd /k "echo API Gateway Starting... && go run ./cmd/api-gateway/main.go"
timeout /t 2 /nobreak >nul

echo Starting Web Server (Port 3000)...
start "Web Server" cmd /k "echo Web Server Starting... && go run ./cmd/web-server/main.go"

echo.
echo ========================================
echo   Service URLs
echo ========================================
echo   Auth Service:  http://localhost:8083
echo   News API:      http://localhost:8081
echo   News Scraper:  http://localhost:8082
echo   API Gateway:   http://localhost:8080
echo   Web Interface: http://localhost:3000
echo.
echo ========================================
echo   Postman Collection
echo ========================================
echo   Import file: News-Aggregator-API.postman_collection.json
echo   Read guide:  POSTMAN_GUIDE.md
echo.
echo Waiting 15 seconds for services to start...
timeout /t 15 /nobreak >nul

echo.
echo Testing services...
curl -s http://localhost:8081/health >nul 2>&1
if errorlevel 1 (
    echo [X] News API: Not ready
) else (
    echo [✓] News API: Healthy
)

curl -s http://localhost:8083/health >nul 2>&1
if errorlevel 1 (
    echo [X] Auth Service: Not ready  
) else (
    echo [✓] Auth Service: Healthy
)

echo.
echo ========================================
echo   Ready to test with Postman!
echo ========================================
pause 