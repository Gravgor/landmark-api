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
		RedisHost:     getEnv("REDISHOST", "localhost"),
		RedisPort:     getEnv("REDISPORT", "6379"),
		RedisPassword: getEnv("REDISPASSWORD", ""),
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
