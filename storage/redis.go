package storage

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStorage implements rate limiting storage using Redis
type RedisStorage struct {
	client redisClient
}

// redisClient interface defines the Redis operations we need
type redisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Incr(ctx context.Context, key string) *redis.IntCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	ExpireAt(ctx context.Context, key string, tm time.Time) *redis.BoolCmd
}

// NewRedisStorage creates a new Redis-based storage
func NewRedisStorage(client redisClient) *RedisStorage {
	return &RedisStorage{
		client: client,
	}
}

// IncrementRequests increments the request count for a key
func (s *RedisStorage) IncrementRequests(key string, now time.Time) (int, error) {
	ctx := context.Background()
	windowKey := fmt.Sprintf("ratelimit:req:%s", key)
	
	// Increment the counter
	count := s.client.Incr(ctx, windowKey)
	if err := count.Err(); err != nil {
		return 0, fmt.Errorf("failed to increment requests: %w", err)
	}

	// Set expiration if this is the first request in the window
	if count.Val() == 1 {
		expireCmd := s.client.ExpireAt(ctx, windowKey, now.Add(time.Minute))
		if err := expireCmd.Err(); err != nil {
			return 0, fmt.Errorf("failed to set expiration: %w", err)
		}
	}

	return int(count.Val()), nil
}

// GetRequests returns the current request count for a key
func (s *RedisStorage) GetRequests(key string) (int, error) {
	ctx := context.Background()
	windowKey := fmt.Sprintf("ratelimit:req:%s", key)
	
	val := s.client.Get(ctx, windowKey)
	if err := val.Err(); err != nil {
		// Key doesn't exist means no requests yet
		if err.Error() == "redis: nil" {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get requests: %w", err)
	}

	count, err := strconv.ParseInt(val.Val(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse request count: %w", err)
	}

	return int(count), nil
}

// IsBlocked checks if a key is blocked
func (s *RedisStorage) IsBlocked(key string) (bool, time.Time, error) {
	ctx := context.Background()
	blockKey := fmt.Sprintf("ratelimit:block:%s", key)
	
	val := s.client.Get(ctx, blockKey)
	if err := val.Err(); err != nil {
		// Key doesn't exist means not blocked
		if err.Error() == "redis: nil" {
			return false, time.Time{}, nil
		}
		return false, time.Time{}, fmt.Errorf("failed to check block status: %w", err)
	}

	// Parse the stored time
	unixTime, err := strconv.ParseInt(val.Val(), 10, 64)
	if err != nil {
		return false, time.Time{}, fmt.Errorf("failed to parse block time: %w", err)
	}

	retryAfter := time.Unix(unixTime, 0)
	
	// Check if still blocked
	if time.Now().Before(retryAfter) {
		return true, retryAfter, nil
	}

	// Block expired, clean up
	_ = s.client.Del(ctx, blockKey)
	return false, time.Time{}, nil
}

// Block marks a key as blocked until the specified time
func (s *RedisStorage) Block(key string, until time.Time) error {
	ctx := context.Background()
	blockKey := fmt.Sprintf("ratelimit:block:%s", key)
	
	// Store the block expiration time
	cmd := s.client.Set(ctx, blockKey, until.Unix(), time.Until(until))
	if err := cmd.Err(); err != nil {
		return fmt.Errorf("failed to set block: %w", err)
	}

	return nil
}

// Reset resets all rate limit data for a key
func (s *RedisStorage) Reset(key string) error {
	ctx := context.Background()
	windowKey := fmt.Sprintf("ratelimit:req:%s", key)
	blockKey := fmt.Sprintf("ratelimit:block:%s", key)
	
	cmd := s.client.Del(ctx, windowKey, blockKey)
	if err := cmd.Err(); err != nil {
		return fmt.Errorf("failed to reset rate limit data: %w", err)
	}

	return nil
}
