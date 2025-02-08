package storage

import (
	"sync"
	"sync/atomic"
	"time"
)

type requestWindow struct {
	count     int64
	startTime atomic.Value // stores time.Time
}

type blockInfo struct {
	until time.Time
}

// MemoryStorage implements rate limiting storage in memory
type MemoryStorage struct {
	requests sync.Map
	blocks   sync.Map
}

// NewMemoryStorage creates a new memory-based storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{}
}

// IncrementRequests increments the request count for a key if under limit
func (s *MemoryStorage) IncrementRequests(key string, now time.Time) (int, error) {
	// Load or initialize window
	value, loaded := s.requests.LoadOrStore(key, &requestWindow{
		count: 0,
	})
	window := value.(*requestWindow)

	// Initialize startTime if new window
	if !loaded {
		window.startTime.Store(now)
	}

	// Get current window start time
	windowStart := window.startTime.Load().(time.Time)

	// Check if window needs reset
	if now.Sub(windowStart) >= time.Minute {
		// Try to reset window atomically
		if atomic.CompareAndSwapInt64(&window.count, atomic.LoadInt64(&window.count), 0) {
			window.startTime.Store(now)
		}
	}

	// Get current count
	currentCount := atomic.LoadInt64(&window.count)
	if currentCount >= 100 {
		return int(currentCount), nil
	}

	// Try to increment atomically only if under limit
	if atomic.CompareAndSwapInt64(&window.count, currentCount, currentCount+1) {
		return int(currentCount + 1), nil
	}

	// If CAS failed, return current count
	return int(atomic.LoadInt64(&window.count)), nil
}

// GetRequests returns the current request count for a key
func (s *MemoryStorage) GetRequests(key string) (int, error) {
	if value, ok := s.requests.Load(key); ok {
		window := value.(*requestWindow)
		windowStart := window.startTime.Load().(time.Time)
		
		// Check if window needs reset
		if time.Since(windowStart) >= time.Minute {
			atomic.StoreInt64(&window.count, 0)
			window.startTime.Store(time.Now())
			return 0, nil
		}
		return int(atomic.LoadInt64(&window.count)), nil
	}
	return 0, nil
}

// IsBlocked checks if a key is blocked
func (s *MemoryStorage) IsBlocked(key string) (bool, time.Time, error) {
	if value, ok := s.blocks.Load(key); ok {
		block := value.(blockInfo)
		if time.Now().Before(block.until) {
			return true, block.until, nil
		}
		// Block expired, clean up
		s.blocks.Delete(key)
	}
	return false, time.Time{}, nil
}

// Block marks a key as blocked until the specified time
func (s *MemoryStorage) Block(key string, until time.Time) error {
	s.blocks.Store(key, blockInfo{until: until})
	return nil
}

// Reset resets all rate limit data for a key
func (s *MemoryStorage) Reset(key string) error {
	s.requests.Delete(key)
	s.blocks.Delete(key)
	return nil
}
