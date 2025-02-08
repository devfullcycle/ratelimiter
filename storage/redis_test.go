package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func setupRedisClient(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("failed to connect to Redis: %s", err)
	}

	return client
}

func TestRedisStorage(t *testing.T) {
	client := setupRedisClient(t)
	defer client.Close()

	// Clean up any existing data
	ctx := context.Background()
	client.FlushAll(ctx)

	storage := NewRedisStorage(client)

	// Test increment requests
	count, err := storage.IncrementRequests("test-ip", time.Now())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Test get requests
	count, err = storage.GetRequests("test-ip")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Test block
	blockUntil := time.Now().Add(time.Minute)
	err = storage.Block("test-ip", blockUntil)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test is blocked
	blocked, retryAfter, err := storage.IsBlocked("test-ip")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !blocked {
		t.Error("Expected IP to be blocked")
	}
	if retryAfter.Unix() != blockUntil.Unix() {
		t.Errorf("Expected retry after %v, got %v", blockUntil, retryAfter)
	}

	// Test reset
	err = storage.Reset("test-ip")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	count, err = storage.GetRequests("test-ip")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 after reset, got %d", count)
	}
}

func TestRedisStorageExpiration(t *testing.T) {
	client := setupRedisClient(t)
	defer client.Close()

	// Clean up any existing data
	ctx := context.Background()
	client.FlushAll(ctx)

	storage := NewRedisStorage(client)

	// Test that requests expire after window
	now := time.Now()
	_, err := storage.IncrementRequests("test-ip", now)
	if err != nil {
		t.Fatalf("Failed to increment requests: %v", err)
	}

	// Verify key exists
	count, err := storage.GetRequests("test-ip")
	if err != nil {
		t.Errorf("Expected no error checking count, got %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1 before expiration, got %d", count)
	}

	// Force key expiration by setting a short TTL
	windowKey := fmt.Sprintf("ratelimit:req:%s", "test-ip")
	err = client.Expire(ctx, windowKey, 1*time.Second).Err()
	if err != nil {
		t.Fatalf("Failed to set expiration: %v", err)
	}

	// Wait for expiration
	time.Sleep(1100 * time.Millisecond)

	// Verify key is gone
	count, err = storage.GetRequests("test-ip")
	if err != nil {
		t.Errorf("Expected no error after expiration, got %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 after expiration, got %d", count)
	}
}
