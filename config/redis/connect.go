package redis

import (
	"context"
	"fmt"
	"sync"

	"project-srv/config"
	"project-srv/pkg/redis"
)

var (
	instance redis.IRedis
	once     sync.Once
	mu       sync.RWMutex
	initErr  error
)

// Connect initializes and connects to Redis using singleton pattern.
func Connect(ctx context.Context, cfg config.RedisConfig) (redis.IRedis, error) {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		return instance, nil
	}

	if initErr != nil {
		once = sync.Once{}
		initErr = nil
	}

	var err error
	once.Do(func() {
		client, e := redis.NewRedis(redis.RedisConfig{
			Host:     cfg.Host,
			Port:     cfg.Port,
			Password: cfg.Password,
			DB:       cfg.DB,
		})
		if e != nil {
			err = fmt.Errorf("failed to create Redis client: %w", e)
			initErr = err
			return
		}
		if e := client.Ping(ctx); e != nil {
			err = fmt.Errorf("failed to ping Redis: %w", e)
			initErr = err
			return
		}
		instance = client
	})

	return instance, err
}

// GetClient returns the singleton Redis client instance.
func GetClient() redis.IRedis {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("Redis client not initialized. Call Connect() first")
	}
	return instance
}

// HealthCheck checks if Redis connection is healthy
func HealthCheck(ctx context.Context) error {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		return fmt.Errorf("Redis client not initialized")
	}

	return instance.Ping(ctx)
}

// Disconnect closes the Redis client and resets the singleton.
func Disconnect() error {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		if err := instance.Close(); err != nil {
			return err
		}
		instance = nil
		once = sync.Once{}
		initErr = nil
	}
	return nil
}
