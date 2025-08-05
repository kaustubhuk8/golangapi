package config

import (
	"fmt"
	"os"
)

type Config struct {
	DSN      string
	RedisURL string
}

func Load() *Config {
	dsn := getEnv("DSN", "manifold:manifoldpassword@tcp(localhost:3306)/manifold?parseTime=true")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	
	// Debug logging
	fmt.Printf("DSN: %s\n", dsn)
	fmt.Printf("REDIS_URL: %s\n", redisURL)
	
	return &Config{
		DSN:      dsn,
		RedisURL: redisURL,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
} 