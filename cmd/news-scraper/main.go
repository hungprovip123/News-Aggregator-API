package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"news-aggregator/pkg/config"
	"news-aggregator/pkg/models"
)

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
}

type NewsScraperService struct {
	db       *gorm.DB
	kafka    *kafka.Writer
	config   *config.Config
	logger   *zap.Logger
	shutdown chan bool
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

	// Auto migrate
	db.AutoMigrate(&models.News{})

	// Kafka writer
	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.KafkaBrokers...),
		Topic:    "news_updates",
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkaWriter.Close()

	service := &NewsScraperService{
		db:       db,
		kafka:    kafkaWriter,
		config:   cfg,
		logger:   logger,
		shutdown: make(chan bool),
	}

	// Start scraping with goroutines
	go service.startScraping()

	logger.Info("News scraper service started")
	<-service.shutdown
}

func (s *NewsScraperService) startScraping() {
	ticker := time.NewTicker(5 * time.Minute) // Scrape every 5 minutes
	defer ticker.Stop()

	// Initial scrape
	s.scrapeAllSources()

	for {
		select {
		case <-ticker.C:
			s.scrapeAllSources()
		case <-s.shutdown:
			return
		}
	}
}

func (s *NewsScraperService) scrapeAllSources() {
	s.logger.Info("Starting news scraping cycle")

	var wg sync.WaitGroup
	for _, source := range s.config.NewsSources {
		if source == "" {
			continue
		}

		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			s.scrapeSource(url)
		}(source)
	}

	wg.Wait()
	s.logger.Info("News scraping cycle completed")
}

func (s *NewsScraperService) scrapeSource(url string) {
	s.logger.Info("Scraping source", zap.String("url", url))

	resp, err := http.Get(url)
	if err != nil {
		s.logger.Error("Failed to fetch RSS", zap.String("url", url), zap.Error(err))
		return
	}
	defer resp.Body.Close()

	var rss RSS
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		s.logger.Error("Failed to parse RSS", zap.String("url", url), zap.Error(err))
		return
	}

	// Process each item
	for _, item := range rss.Channel.Items {
		if item.Link == "" || item.Title == "" {
			continue
		}

		// Check if article already exists
		var existingNews models.News
		if err := s.db.Where("url = ?", item.Link).First(&existingNews).Error; err == nil {
			continue // Article already exists
		}

		// Parse published date
		pubTime := time.Now()
		if item.PubDate != "" {
			if parsed, err := time.Parse(time.RFC1123, item.PubDate); err == nil {
				pubTime = parsed
			}
		}

		// Create news entry
		news := models.News{
			Title:       item.Title,
			Description: item.Description,
			URL:         item.Link,
			Source:      rss.Channel.Title,
			PublishedAt: pubTime,
		}

		// Save to database
		if err := s.db.Create(&news).Error; err != nil {
			s.logger.Error("Failed to save news", zap.Error(err))
			continue
		}

		// Send to Kafka
		s.sendToKafka(news)

		s.logger.Info("Thu thập tin: " + news.Title)
	}
}

func (s *NewsScraperService) sendToKafka(news models.News) {
	message := kafka.Message{
		Key: []byte(fmt.Sprintf("news_%d", news.ID)),
		Value: []byte(fmt.Sprintf(`{"id":%d,"title":"%s","url":"%s","source":"%s"}`,
			news.ID, news.Title, news.URL, news.Source)),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.kafka.WriteMessages(ctx, message); err != nil {
		s.logger.Error("Failed to send message to Kafka", zap.Error(err))
	}
}
