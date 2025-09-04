package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	AppBaseURL      string
	SessionSecret   string
	GoogleOAuthClientID string
	GoogleOAuthClientSecret string
	AllowedDomains  []string
	JPEGQuality     int
	JPEGProgressive bool
	PNGStrip        bool
	R2AccountID     string
	R2AccessKeyID   string
	R2SecretAccessKey string
	R2Bucket        string
	R2PublicBaseURL string
	R2S3Endpoint    string
}

func Load() *Config {
	// Try to load .env file from project root (one level up from backend/)
	envPath := filepath.Join("..", ".env")
	godotenv.Load(envPath)
	
	// Also try loading from current directory
	godotenv.Load(".env")
	
	return &Config{
		Port:            getEnv("PORT", "8080"),
		AppBaseURL:      getEnv("APP_BASE_URL", "http://localhost:3000"),
		SessionSecret:   getEnv("SESSION_SECRET", ""),
		GoogleOAuthClientID: getEnv("GOOGLE_OAUTH_CLIENT_ID", ""),
		GoogleOAuthClientSecret: getEnv("GOOGLE_OAUTH_CLIENT_SECRET", ""),
		AllowedDomains:  strings.Split(getEnv("ALLOWED_DOMAINS", "hackclub.com"), ","),
		JPEGQuality:     getEnvInt("JPEG_QUALITY", 84),
		JPEGProgressive: getEnvBool("JPEG_PROGRESSIVE", true),
		PNGStrip:        getEnvBool("PNG_STRIP", true),
		R2AccountID:     getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:   getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2Bucket:        getEnv("R2_BUCKET", "format-assets"),
		R2PublicBaseURL: getEnv("R2_PUBLIC_BASE_URL", "https://i.format.hackclub.com"),
		R2S3Endpoint:    getEnv("R2_S3_ENDPOINT", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
