package ratelimiter

import "time"

// Storage defines the interface for rate limit data storage
type Storage interface {
	// IncrementRequests increments the request count for a key and returns the new count
	IncrementRequests(key string, now time.Time) (int, error)

	// GetRequests returns the current request count for a key
	GetRequests(key string) (int, error)

	// IsBlocked checks if a key is blocked and returns when it can retry
	IsBlocked(key string) (blocked bool, retryAfter time.Time, err error)

	// Block marks a key as blocked until the specified time
	Block(key string, until time.Time) error

	// Reset resets all rate limit data for a key
	Reset(key string) error
}
