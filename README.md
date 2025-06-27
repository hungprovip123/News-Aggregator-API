# 📰 News Aggregator API

Hệ thống tập hợp tin tức với microservices architecture, JWT authentication và web interface.

## 🚀 Chạy ứng dụng

### **Cách 1: Sử dụng script (Khuyến nghị)**
```bash
start-services.bat
```

### **Cách 2: Thủ công**
```bash
# Terminal 1 - Auth Service
go run .\cmd\auth-service\main.go

# Terminal 2 - News API  
go run .\cmd\news-api\main.go

# Terminal 3 - News Scraper
go run .\cmd\news-scraper\main.go

# Terminal 4 - Web Server
go run .\cmd\web-server\main.go
```

## 🌐 Truy cập ứng dụng

- **Web Interface**: http://localhost:3000
- **Auth Service**: http://localhost:8083  
- **News API**: http://localhost:8081
- **API Gateway**: http://localhost:8080

## 📋 Chức năng chính

### **Authentication**
- Đăng ký user mới
- Đăng nhập và lấy JWT token
- Xác thực token

### **News API**
- Xem danh sách tin tức
- Tìm kiếm tin tức
- Phân trang kết quả
- Lọc theo nguồn tin

### **Web Interface**
- Test API trực tiếp trên browser
- JWT token tự động lưu
- Real-time testing không cần Postman

## ⚙️ Cấu hình

File `config.env` chứa cấu hình chính:

```env
# Database
DB_USER=postgres
DB_PASSWORD=123456
DB_NAME=newsdb

# JWT
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# Ports
AUTH_SERVICE_PORT=8083
NEWS_API_PORT=8081
```

## 🔧 Cài đặt database (tuỳ chọn)

```bash
# PostgreSQL
docker run -d --name postgres -e POSTGRES_DB=newsdb -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=123456 -p 5432:5432 postgres:15

# Redis
docker run -d --name redis -p 6379:6379 redis:7-alpine
```

## 📊 API Endpoints

### **Auth Service**
```
POST /api/v1/register    # Đăng ký
POST /api/v1/login       # Đăng nhập  
POST /api/v1/verify      # Xác thực token
GET  /health             # Health check
```

### **News API**
```
GET /api/v1/news         # Lấy danh sách tin tức
GET /api/v1/news/:id     # Lấy tin tức theo ID
GET /api/v1/news/source/:source  # Lọc theo nguồn
GET /health              # Health check
```

## 🛠️ Cấu trúc project

```
week4/
├── cmd/                 # Các services
│   ├── auth-service/
│   ├── news-api/
│   ├── news-scraper/
│   └── web-server/
├── pkg/                 # Shared code
├── web/                 # Web interface
├── config.env           # Cấu hình
└── start-services.bat   # Script khởi động
```

## 🔥 Quick Start

1. **Clone project**
2. **Chạy**: `start-services.bat`
3. **Mở**: http://localhost:3000
4. **Test**: Register → Login → Get News

## 🐛 Troubleshooting

**Services không khởi động:**
- Kiểm tra port có bị sử dụng: `netstat -ano | findstr :8083`
- Restart script: Dừng services và chạy lại

**Web interface lỗi:**
- Đảm bảo tất cả services đang chạy
- Hard refresh browser (Ctrl+F5)

**Database lỗi:**
- Services vẫn hoạt động mà không cần database
- Có thể chạy PostgreSQL container nếu cần

## ✨ Tính năng nổi bật

- 🔐 JWT Authentication với bcrypt
- 🌐 Web interface test API real-time
- 🚀 Microservices architecture  
- 📱 CORS enabled cho browser calls
- ⚡ Redis caching và rate limiting
- 🔍 Health monitoring cho tất cả services 