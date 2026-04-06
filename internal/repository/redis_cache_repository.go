package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCacheRepository struct {
	client *redis.Client
}

func NewRedisCacheRepository(client *redis.Client) *RedisCacheRepository {
	return &RedisCacheRepository{client: client}
}

func (r *RedisCacheRepository) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisCacheRepository) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisCacheRepository) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
