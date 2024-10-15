package services

import (
	"context"
	"encoding/json"
	"fmt"
	"landmark-api/internal/config"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheService interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	DeleteByPattern(ctx context.Context, pattern string) error
}

type RedisCacheService struct {
	client *redis.Client
}

func NewRedisCacheService(cfg *config.CacheConfig) (*RedisCacheService, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &RedisCacheService{client: client}, nil
}

func (c *RedisCacheService) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *RedisCacheService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %v", err)
	}
	return c.client.Set(ctx, key, jsonData, expiration).Err()
}

func (c *RedisCacheService) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *RedisCacheService) DeleteByPattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		err := c.client.Del(ctx, iter.Val()).Err()
		if err != nil {
			return err
		}
	}
	return iter.Err()
}
