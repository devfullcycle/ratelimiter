package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupRedisContainer(t *testing.T) (testcontainers.Container, *redis.Client) {
	ctx := context.Background()
	
	// Container request
	req := testcontainers.ContainerRequest{
		Image:        "redis:7.2",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	// Start container
	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:         true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}

	// Get mapped port
	mappedPort, err := redisC.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("failed to get container external port: %s", err)
	}

	// Get host
	host, err := redisC.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %s", err)
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, mappedPort.Port()),
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("failed to connect to Redis: %s", err)
	}

	return redisC, client
}

func TestRedisStorage(t *testing.T) {
	t.Parallel()
	redisC, client := setupRedisContainer(t)
	defer func() {
		client.Close()
		if err := redisC.Terminate(context.Background()); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

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
	t.Parallel()
	redisC, client := setupRedisContainer(t)
	defer func() {
		client.Close()
		if err := redisC.Terminate(context.Background()); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

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
