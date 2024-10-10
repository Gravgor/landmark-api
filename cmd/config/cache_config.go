package config

import (
	"os"
	"time"
)

type CacheConfig struct {
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	DefaultTTL    time.Duration
}

func NewCacheConfig() *CacheConfig {
	return &CacheConfig{
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,
		DefaultTTL:    15 * time.Minute,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
