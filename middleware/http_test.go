package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devfullcycle/ratelimiter/ratelimiter"
	"log/slog"
	"os"
)

func TestRateLimitMiddleware(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	storage := &mockStorage{}
	limiter := ratelimiter.New(storage)
	middleware := NewRateLimitMiddleware(limiter, logger)

	// Test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test successful request
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	middleware.Handler(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Test rate limit exceeded
	storage.count = 101
	req = httptest.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()
	middleware.Handler(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status code %d, got %d", http.StatusTooManyRequests, rec.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if resp.Error != "Rate limit exceeded" {
		t.Errorf("Expected error message 'Rate limit exceeded', got '%s'", resp.Error)
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
