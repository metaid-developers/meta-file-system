package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"meta-file-system/conf"

	"github.com/redis/go-redis/v9"
)

var (
	RedisClient *redis.Client
	ctx         = context.Background()
)

// InitRedis initialize Redis client
func InitRedis() error {
	if !conf.Cfg.Redis.Enabled {
		log.Println("Redis cache is disabled")
		return nil
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", conf.Cfg.Redis.Host, conf.Cfg.Redis.Port),
		Password: conf.Cfg.Redis.Password,
		DB:       conf.Cfg.Redis.DB,
	})

	// Test connection
	if err := RedisClient.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️  Failed to connect to Redis: %v", err)
		log.Println("Redis cache will be disabled")
		RedisClient = nil
		return err
	}

	log.Printf("✅ Redis connected successfully: %s:%d (DB: %d, TTL: %ds)",
		conf.Cfg.Redis.Host, conf.Cfg.Redis.Port, conf.Cfg.Redis.DB, conf.Cfg.Redis.CacheTTL)
	return nil
}

// CloseRedis close Redis connection
func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}

// SetCache set cache with TTL
func SetCache(key string, value interface{}) error {
	if RedisClient == nil || !conf.Cfg.Redis.Enabled {
		return nil // Cache disabled, skip silently
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	ttl := time.Duration(conf.Cfg.Redis.CacheTTL) * time.Second
	if err := RedisClient.Set(ctx, key, data, ttl).Err(); err != nil {
		log.Printf("⚠️  Failed to set cache for key %s: %v", key, err)
		return err
	}

	return nil
}

// GetCache get cache by key
func GetCache(key string, dest interface{}) error {
	if RedisClient == nil || !conf.Cfg.Redis.Enabled {
		return redis.Nil // Cache disabled, return nil (cache miss)
	}

	data, err := RedisClient.Get(ctx, key).Result()
	if err != nil {
		return err // redis.Nil if key not found
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	return nil
}

// DeleteCache delete cache by key
func DeleteCache(key string) error {
	if RedisClient == nil || !conf.Cfg.Redis.Enabled {
		return nil // Cache disabled, skip silently
	}

	if err := RedisClient.Del(ctx, key).Err(); err != nil {
		log.Printf("⚠️  Failed to delete cache for key %s: %v", key, err)
		return err
	}

	return nil
}

// DeleteCachePattern delete cache by pattern
func DeleteCachePattern(pattern string) error {
	if RedisClient == nil || !conf.Cfg.Redis.Enabled {
		return nil // Cache disabled, skip silently
	}

	iter := RedisClient.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := RedisClient.Del(ctx, iter.Val()).Err(); err != nil {
			log.Printf("⚠️  Failed to delete cache for key %s: %v", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}

	return nil
}

// IsRedisEnabled check if Redis is enabled and connected
func IsRedisEnabled() bool {
	return RedisClient != nil && conf.Cfg.Redis.Enabled
}

// SetHashField set hash field value
func SetHashField(hashKey, field string, value interface{}) error {
	if RedisClient == nil || !conf.Cfg.Redis.Enabled {
		return nil // Cache disabled, skip silently
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal hash data: %w", err)
	}

	if err := RedisClient.HSet(ctx, hashKey, field, data).Err(); err != nil {
		log.Printf("⚠️  Failed to set hash field %s:%s: %v", hashKey, field, err)
		return err
	}

	return nil
}

// GetAllHashFields get all fields from hash
func GetAllHashFields(hashKey string) (map[string]string, error) {
	if RedisClient == nil || !conf.Cfg.Redis.Enabled {
		return nil, redis.Nil
	}

	result, err := RedisClient.HGetAll(ctx, hashKey).Result()
	if err != nil {
		return nil, err
	}

	return result, nil
}
