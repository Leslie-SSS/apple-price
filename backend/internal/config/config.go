package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	Port        string
	Host        string

	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	ScraperInterval    time.Duration
	ScraperUserAgent   string
	DataDir            string
	CORSOrigins        string
}

func Load() (*Config, error) {
	// Load .env file if exists (ignore error in production)
	_ = godotenv.Load()

	cfg := &Config{
		Environment:        getEnv("ENVIRONMENT", "development"),
		Port:              getEnv("PORT", "8080"),
		Host:              getEnv("HOST", "0.0.0.0"),
		SMTPHost:          getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPUser:          getEnv("SMTP_USER", ""),
		SMTPPassword:      getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:          getEnv("SMTP_FROM", "ApplePrice <noreply@example.com>"),
		ScraperUserAgent:  getEnv("SCRAPER_USER_AGENT", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"),
		DataDir:           getEnv("DATA_DIR", "./data"),
		CORSOrigins:       getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000"),
	}

	// Parse integer values
	if port := getEnv("SMTP_PORT", "587"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
		}
		cfg.SMTPPort = p
	}

	// Parse duration
	if interval := getEnv("SCRAPER_INTERVAL", "5m"); interval != "" {
		d, err := time.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid SCRAPER_INTERVAL: %w", err)
		}
		cfg.ScraperInterval = d
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
