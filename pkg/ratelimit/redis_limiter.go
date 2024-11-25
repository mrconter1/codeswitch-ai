package ratelimit

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisRateLimiter struct {
	client     *redis.Client
	key        string
	maxTokens  int
	refillRate time.Duration
}

func NewRedisRateLimiter(client *redis.Client, key string, maxTokens int, refillRate time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		client:     client,
		key:        key,
		maxTokens:  maxTokens,
		refillRate: refillRate,
	}
}

func (r *RedisRateLimiter) Allow(ctx context.Context) bool {
	script := `
		local tokens = tonumber(redis.call('get', KEYS[1]) or ARGV[1])
		if tokens > 0 then
			redis.call('set', KEYS[1], tokens - 1)
			return 1
		end
		return 0
	`

	result, err := r.client.Eval(ctx, script, []string{r.key}, r.maxTokens).Result()
	if err != nil {
		return false
	}

	return result.(int64) == 1
}

func (r *RedisRateLimiter) WaitForQuota(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if r.Allow(ctx) {
				return nil
			}
			time.Sleep(r.refillRate)
		}
	}
}
