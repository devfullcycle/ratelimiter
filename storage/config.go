package storage

import (
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// DefaultRedisConfig returns a RedisConfig with default values
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Host:     getEnvOrDefault("REDIS_HOST", "localhost"),
		Port:     getEnvOrDefault("REDIS_PORT", "6379"),
		Password: getEnvOrDefault("REDIS_PASSWORD", ""),
		DB:       getEnvIntOrDefault("REDIS_DB", 0),
	}
}

// NewRedisClient creates a new Redis client from config
func NewRedisClient(cfg RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Host + ":" + cfg.Port,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}

// Helper functions for environment variables
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
