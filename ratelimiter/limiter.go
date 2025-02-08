package ratelimiter

import (
	"time"
)

// Options configures the rate limiter behavior
type Options struct {
	MaxRequests   int           // Maximum requests allowed in the time window
	TimeWindow    time.Duration // Time window for counting requests
	BlockDuration time.Duration // Duration to block after limit exceeded
}

// Option is a function that configures Options
type Option func(*Options)

// WithMaxRequests sets the maximum requests allowed in the time window
func WithMaxRequests(max int) Option {
	return func(o *Options) {
		o.MaxRequests = max
	}
}

// WithTimeWindow sets the time window for counting requests
func WithTimeWindow(d time.Duration) Option {
	return func(o *Options) {
		o.TimeWindow = d
	}
}

// WithBlockDuration sets the duration to block after limit exceeded
func WithBlockDuration(d time.Duration) Option {
	return func(o *Options) {
		o.BlockDuration = d
	}
}

// Response contains the rate limit check result
type Response struct {
	Allowed      bool      `json:"allowed"`
	RetryAfter   time.Time `json:"retry_after,omitempty"`
	RequestsLeft int       `json:"requests_left"`
	RequestsMade int       `json:"requests_made"`
	Limit        int       `json:"limit"`
}

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	opts    Options
	storage Storage
}

// New creates a new RateLimiter with the given options
func New(storage Storage, opts ...Option) *RateLimiter {
	options := Options{
		MaxRequests:   100,          // Default: 100 requests
		TimeWindow:    time.Minute,  // Default: per minute
		BlockDuration: time.Minute,  // Default: 1 minute block
	}

	for _, opt := range opts {
		opt(&options)
	}

	return &RateLimiter{
		opts:    options,
		storage: storage,
	}
}

// Allow checks if a request is allowed for the given key
func (rl *RateLimiter) Allow(key string) (Response, error) {
	// Check if key is blocked first
	blocked, retryAfter, err := rl.storage.IsBlocked(key)
	if err != nil {
		return Response{}, err
	}

	if blocked {
		return Response{
			Allowed:      false,
			RetryAfter:   retryAfter,
			RequestsLeft: 0,
			RequestsMade: rl.opts.MaxRequests,
			Limit:        rl.opts.MaxRequests,
		}, nil
	}

	// Get current count first
	count, err := rl.storage.GetRequests(key)
	if err != nil {
		return Response{}, err
	}

	// Allow requests until MaxRequests is reached
	if count < rl.opts.MaxRequests {
		// Increment only if we're under the limit
		count, err = rl.storage.IncrementRequests(key, time.Now())
		if err != nil {
			return Response{}, err
		}
		return Response{
			Allowed:      true,
			RequestsLeft: rl.opts.MaxRequests - count,
			RequestsMade: count,
			Limit:        rl.opts.MaxRequests,
		}, nil
	}

	// Block only after MaxRequests exceeded
	blockUntil := time.Now().Add(rl.opts.BlockDuration)
	if err := rl.storage.Block(key, blockUntil); err != nil {
		return Response{}, err
	}

	return Response{
		Allowed:      false,
		RetryAfter:   blockUntil,
		RequestsLeft: 0,
		RequestsMade: count,
		Limit:        rl.opts.MaxRequests,
	}, nil
}

// Reset resets the rate limit for a given key
func (rl *RateLimiter) Reset(key string) error {
	return rl.storage.Reset(key)
}
