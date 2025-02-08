package ratelimiter

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	storage := &mockStorage{}
	limiter := New(storage)

	// Test successful request
	resp, err := limiter.Allow("test-ip")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !resp.Allowed {
		t.Error("Expected request to be allowed")
	}

	// Test rate limit exceeded
	storage.count = 101
	resp, err = limiter.Allow("test-ip")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if resp.Allowed {
		t.Error("Expected request to be blocked")
	}
}

// Mock storage for testing
type mockStorage struct {
	count int
}

func (m *mockStorage) IncrementRequests(key string, now time.Time) (int, error) {
	m.count++
	return m.count, nil
}

func (m *mockStorage) GetRequests(key string) (int, error) {
	return m.count, nil
}

func (m *mockStorage) IsBlocked(key string) (bool, time.Time, error) {
	if m.count > 100 {
		return true, time.Now().Add(time.Minute), nil
	}
	return false, time.Time{}, nil
}

func (m *mockStorage) Block(key string, until time.Time) error {
	return nil
}

func (m *mockStorage) Reset(key string) error {
	m.count = 0
	return nil
}
