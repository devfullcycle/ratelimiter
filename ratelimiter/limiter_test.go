package ratelimiter

import (
	"sync"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	storage := &mockStorage{
		mu: &sync.Mutex{},
	}
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

func TestConcurrentRequests(t *testing.T) {
	storage := &mockStorage{
		mu: &sync.Mutex{},
	}
	limiter := New(storage)

	var wg sync.WaitGroup
	requests := 150
	wg.Add(requests)

	responses := make([]Response, requests)
	for i := 0; i < requests; i++ {
		go func(idx int) {
			defer wg.Done()
			resp, err := limiter.Allow("test-ip")
			if err != nil {
				t.Errorf("Error on request %d: %v", idx, err)
				return
			}
			responses[idx] = resp
		}(i)
	}

	wg.Wait()

	allowed := 0
	blocked := 0
	for _, resp := range responses {
		if resp.Allowed {
			allowed++
		} else {
			blocked++
		}
	}

	if allowed != 100 {
		t.Errorf("Expected exactly 100 allowed requests, got %d", allowed)
	}

	if blocked != 50 {
		t.Errorf("Expected exactly 50 blocked requests, got %d", blocked)
	}
}

// Mock storage for testing
type mockStorage struct {
	count int
	mu    *sync.Mutex
}

func (m *mockStorage) IncrementRequests(key string, now time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.count++
	return m.count, nil
}

func (m *mockStorage) GetRequests(key string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.count, nil
}

func (m *mockStorage) IsBlocked(key string) (bool, time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.count > 100 {
		return true, time.Now().Add(time.Minute), nil
	}
	return false, time.Time{}, nil
}

func (m *mockStorage) Block(key string, until time.Time) error {
	return nil
}

func (m *mockStorage) Reset(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.count = 0
	return nil
}
