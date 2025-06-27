package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"news-aggregator/pkg/config"
	"news-aggregator/pkg/middleware"
)

type APIGateway struct {
	config *config.Config
	redis  *redis.Client
	logger *zap.Logger
}

func main() {
	cfg := config.Load()

	// Logger setup
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Redis connection
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})

	gateway := &APIGateway{
		config: cfg,
		redis:  rdb,
		logger: logger,
	}

	router := gin.Default()

	// Middleware
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	auth := middleware.NewAuthMiddleware(cfg.JWTSecret, rdb)
	router.Use(auth.RateLimit(cfg.RateLimitReqs, time.Duration(cfg.RateLimitWindow)*time.Second))

	gateway.setupRoutes(router, auth)

	logger.Info("API Gateway starting", zap.String("port", cfg.APIGatewayPort))
	log.Fatal(http.ListenAndServe(":"+cfg.APIGatewayPort, router))
}

func (g *APIGateway) setupRoutes(router *gin.Engine, auth *middleware.AuthMiddleware) {
	api := router.Group("/api/v1")
	{
		// Auth routes
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", g.proxyToAuth)
			authGroup.POST("/login", g.proxyToAuth)
			authGroup.POST("/verify", g.proxyToAuth)
		}

		// News routes
		newsGroup := api.Group("/news")
		{
			newsGroup.GET("", g.proxyToNewsAPI)
			newsGroup.GET("/:id", g.proxyToNewsAPI)
			newsGroup.GET("/source/:source", g.proxyToNewsAPI)

			// Protected routes
			protected := newsGroup.Group("")
			protected.Use(auth.JWTAuth())
			{
				protected.POST("/favorite/:id", g.proxyToNewsAPI)
			}
		}
	}

	// Health check
	router.GET("/health", g.healthCheck)
	router.GET("/", g.welcome)
}

func (g *APIGateway) proxyToAuth(c *gin.Context) {
	targetURL := fmt.Sprintf("http://localhost:%s%s", g.config.AuthServicePort, c.Request.RequestURI)
	g.proxyRequest(c, targetURL)
}

func (g *APIGateway) proxyToNewsAPI(c *gin.Context) {
	targetURL := fmt.Sprintf("http://localhost:%s%s", g.config.NewsAPIPort, c.Request.RequestURI)
	g.proxyRequest(c, targetURL)
}

func (g *APIGateway) proxyRequest(c *gin.Context, targetURL string) {
	// Read request body
	var body []byte
	if c.Request.Body != nil {
		body, _ = io.ReadAll(c.Request.Body)
		c.Request.Body.Close()
	}

	// Create new request
	req, err := http.NewRequest(c.Request.Method, targetURL, bytes.NewBuffer(body))
	if err != nil {
		g.logger.Error("Failed to create proxy request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Proxy request failed"})
		return
	}

	// Copy headers
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Copy query parameters
	req.URL.RawQuery = c.Request.URL.RawQuery

	// Make request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		g.logger.Error("Proxy request failed", zap.String("url", targetURL), zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		g.logger.Error("Failed to read proxy response", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// Return response
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), responseBody)
}

func (g *APIGateway) healthCheck(c *gin.Context) {
	status := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"services":  gin.H{},
	}

	// Check auth service
	authHealth := g.checkServiceHealth(fmt.Sprintf("http://localhost:%s/health", g.config.AuthServicePort))
	status["services"].(gin.H)["auth"] = authHealth

	// Check news API service
	newsAPIHealth := g.checkServiceHealth(fmt.Sprintf("http://localhost:%s/health", g.config.NewsAPIPort))
	status["services"].(gin.H)["news-api"] = newsAPIHealth

	overallHealthy := authHealth["status"] == "healthy" && newsAPIHealth["status"] == "healthy"

	statusCode := http.StatusOK
	if !overallHealthy {
		status["status"] = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, status)
}

func (g *APIGateway) checkServiceHealth(url string) gin.H {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return gin.H{"status": "unhealthy", "error": err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return gin.H{"status": "healthy"}
	}

	return gin.H{"status": "unhealthy", "http_status": resp.StatusCode}
}

func (g *APIGateway) welcome(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "News Aggregator API Gateway",
		"version": "1.0.0",
		"endpoints": gin.H{
			"auth": gin.H{
				"register": "POST /api/v1/auth/register",
				"login":    "POST /api/v1/auth/login",
				"verify":   "POST /api/v1/auth/verify",
			},
			"news": gin.H{
				"list":      "GET /api/v1/news",
				"get":       "GET /api/v1/news/:id",
				"by_source": "GET /api/v1/news/source/:source",
				"favorite":  "POST /api/v1/news/favorite/:id (auth required)",
			},
			"health": "GET /health",
		},
	})
}
