package storage

import (
	"sync"
	"time"
)

type requestWindow struct {
	count     int
	startTime time.Time
}

type blockInfo struct {
	until time.Time
}

// MemoryStorage implements rate limiting storage in memory
type MemoryStorage struct {
	requests map[string]requestWindow
	blocks   map[string]blockInfo
	mu       sync.RWMutex
}

// NewMemoryStorage creates a new memory-based storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		requests: make(map[string]requestWindow),
		blocks:   make(map[string]blockInfo),
	}
}

// IncrementRequests increments the request count for a key
func (s *MemoryStorage) IncrementRequests(key string, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	window, exists := s.requests[key]
	if !exists || now.Sub(window.startTime) >= time.Minute {
		// Start new window
		s.requests[key] = requestWindow{
			count:     1,
			startTime: now,
		}
		return 1, nil
	}

	// Increment existing window
	window.count++
	s.requests[key] = window
	return window.count, nil
}

// GetRequests returns the current request count for a key
func (s *MemoryStorage) GetRequests(key string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if window, exists := s.requests[key]; exists {
		return window.count, nil
	}
	return 0, nil
}

// IsBlocked checks if a key is blocked
func (s *MemoryStorage) IsBlocked(key string) (bool, time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if block, exists := s.blocks[key]; exists {
		if time.Now().Before(block.until) {
			return true, block.until, nil
		}
		// Block expired, clean up
		delete(s.blocks, key)
	}
	return false, time.Time{}, nil
}

// Block marks a key as blocked until the specified time
func (s *MemoryStorage) Block(key string, until time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.blocks[key] = blockInfo{until: until}
	return nil
}

// Reset resets all rate limit data for a key
func (s *MemoryStorage) Reset(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.requests, key)
	delete(s.blocks, key)
	return nil
}
