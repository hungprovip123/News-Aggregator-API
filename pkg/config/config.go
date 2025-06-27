package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL     string
	RedisURL        string
	KafkaBrokers    []string
	JWTSecret       string
	JWTExpireHours  int
	APIGatewayPort  string
	NewsAPIPort     string
	ScraperPort     string
	AuthServicePort string
	RateLimitReqs   int
	RateLimitWindow int
	NewsSources     []string
}

func Load() *Config {
	if err := godotenv.Load("config.env"); err != nil {
		log.Println("No config.env file found, using environment variables")
	}

	cfg := &Config{
		DatabaseURL:     buildDatabaseURL(),
		RedisURL:        buildRedisURL(),
		KafkaBrokers:    strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		JWTSecret:       getEnv("JWT_SECRET", "default-secret"),
		JWTExpireHours:  getEnvInt("JWT_EXPIRE_HOURS", 24),
		APIGatewayPort:  getEnv("API_GATEWAY_PORT", "8080"),
		NewsAPIPort:     getEnv("NEWS_API_PORT", "8081"),
		ScraperPort:     getEnv("NEWS_SCRAPER_PORT", "8082"),
		AuthServicePort: getEnv("AUTH_SERVICE_PORT", "8083"),
		RateLimitReqs:   getEnvInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow: getEnvInt("RATE_LIMIT_WINDOW", 60),
		NewsSources:     strings.Split(getEnv("NEWS_SOURCES", ""), ","),
	}

	return cfg
}

func buildDatabaseURL() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "newsuser")
	password := getEnv("DB_PASSWORD", "newspass")
	dbname := getEnv("DB_NAME", "newsdb")

	return "host=" + host + " port=" + port + " user=" + user + " password=" + password + " dbname=" + dbname + " sslmode=disable"
}

func buildRedisURL() string {
	host := getEnv("REDIS_HOST", "localhost")
	port := getEnv("REDIS_PORT", "6379")
	password := getEnv("REDIS_PASSWORD", "")

	url := host + ":" + port
	if password != "" {
		url = password + "@" + url
	}
	return url
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
