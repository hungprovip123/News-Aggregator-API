package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"news-aggregator/pkg/config"
	"news-aggregator/pkg/middleware"
	"news-aggregator/pkg/models"
)

type NewsAPIService struct {
	db     *gorm.DB
	redis  *redis.Client
	config *config.Config
	logger *zap.Logger
}

func main() {
	cfg := config.Load()

	// Logger setup
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Database connection
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}

	// Redis connection
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})

	service := &NewsAPIService{
		db:     db,
		redis:  rdb,
		config: cfg,
		logger: logger,
	}

	router := gin.Default()

	// Setup middleware
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	auth := middleware.NewAuthMiddleware(cfg.JWTSecret, rdb)
	router.Use(auth.RateLimit(cfg.RateLimitReqs, time.Duration(cfg.RateLimitWindow)*time.Second))

	service.setupRoutes(router, auth)

	logger.Info("News API service starting", zap.String("port", cfg.NewsAPIPort))
	log.Fatal(http.ListenAndServe(":"+cfg.NewsAPIPort, router))
}

func (s *NewsAPIService) setupRoutes(router *gin.Engine, auth *middleware.AuthMiddleware) {
	api := router.Group("/api/v1")
	{
		// Public endpoints
		api.GET("/news", s.getNews)
		api.GET("/news/:id", s.getNewsById)
		api.GET("/news/source/:source", s.getNewsBySource)

		// Protected endpoints
		protected := api.Group("")
		protected.Use(auth.JWTAuth())
		{
			protected.POST("/news/favorite/:id", s.favoriteNews)
		}
	}

	// Health check
	router.GET("/health", s.healthCheck)
}

func (s *NewsAPIService) getNews(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	source := c.Query("source")
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Check cache first
	cacheKey := fmt.Sprintf("news:page_%d:limit_%d:source_%s:search_%s", page, limit, source, search)
	cached, err := s.redis.Get(context.Background(), cacheKey).Result()
	if err == nil && cached != "" {
		c.Header("X-Cache", "HIT")
		c.Data(http.StatusOK, "application/json", []byte(cached))
		return
	}

	// Build query
	query := s.db.Model(&models.News{})

	if source != "" {
		query = query.Where("source ILIKE ?", "%"+source+"%")
	}

	if search != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get news with pagination
	var news []models.News
	if err := query.Order("published_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&news).Error; err != nil {
		s.logger.Error("Failed to fetch news", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch news"})
		return
	}

	response := models.NewsResponse{
		Data:  news,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	// Cache response for 5 minutes
	responseJSON, _ := json.Marshal(response)
	s.redis.Set(context.Background(), cacheKey, responseJSON, 5*time.Minute)

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, response)
}

func (s *NewsAPIService) getNewsById(c *gin.Context) {
	id := c.Param("id")

	// Check cache
	cacheKey := fmt.Sprintf("news:id_%s", id)
	cached, err := s.redis.Get(context.Background(), cacheKey).Result()
	if err == nil && cached != "" {
		c.Header("X-Cache", "HIT")
		c.Data(http.StatusOK, "application/json", []byte(cached))
		return
	}

	var news models.News
	if err := s.db.Where("id = ?", id).First(&news).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "News not found"})
			return
		}
		s.logger.Error("Failed to fetch news", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch news"})
		return
	}

	// Cache for 10 minutes
	newsJSON, _ := json.Marshal(news)
	s.redis.Set(context.Background(), cacheKey, newsJSON, 10*time.Minute)

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, news)
}

func (s *NewsAPIService) getNewsBySource(c *gin.Context) {
	source := c.Param("source")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	var news []models.News
	var total int64

	// Get total count
	s.db.Model(&models.News{}).Where("source ILIKE ?", "%"+source+"%").Count(&total)

	// Get news
	if err := s.db.Where("source ILIKE ?", "%"+source+"%").
		Order("published_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&news).Error; err != nil {
		s.logger.Error("Failed to fetch news by source", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch news"})
		return
	}

	response := models.NewsResponse{
		Data:  news,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	c.JSON(http.StatusOK, response)
}

func (s *NewsAPIService) favoriteNews(c *gin.Context) {
	newsID := c.Param("id")
	userID := c.GetString("userID")

	// Implementation for favorite functionality
	// This would typically involve a user_favorites table
	s.logger.Info("User favorited news",
		zap.String("userID", userID),
		zap.String("newsID", newsID))

	c.JSON(http.StatusOK, gin.H{"message": "News favorited successfully"})
}

func (s *NewsAPIService) healthCheck(c *gin.Context) {
	// Check database connection
	sqlDB, err := s.db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "database connection failed",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "database ping failed",
		})
		return
	}

	// Check Redis connection
	if err := s.redis.Ping(context.Background()).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "redis connection failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}
