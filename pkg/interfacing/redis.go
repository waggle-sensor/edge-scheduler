package interfacing

import (
	"context"
	"time"

	"github.com/go-redis/redis/v9"
)

type RedisClient struct {
	URI    string
	client *redis.Client
}

func NewRedisClient(uri string) *RedisClient {
	return &RedisClient{
		URI: uri,
	}
}

func (r *RedisClient) connect() error {
	if r.client == nil {
		r.client = redis.NewClient(&redis.Options{
			Addr: r.URI,
		})
	}
	return nil
}

func (r *RedisClient) Set(k string, v interface{}) error {
	err := r.connect()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return r.client.Set(ctx, k, v, 0).Err()
}
