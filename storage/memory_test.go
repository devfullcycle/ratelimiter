package storage

import (
	"testing"
	"time"
)

func TestMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()

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
