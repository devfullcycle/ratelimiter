package middleware

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/devfullcycle/ratelimiter/ratelimiter"
)

// ErrorResponse represents the JSON error response
type ErrorResponse struct {
	Error        string    `json:"error"`
	Limit        int       `json:"limit"`
	RequestsMade int       `json:"requests_made"`
	RetryAfter   time.Time `json:"retry_after"`
}

// RateLimitMiddleware wraps a rate limiter with HTTP middleware functionality
type RateLimitMiddleware struct {
	limiter *ratelimiter.RateLimiter
	logger  *slog.Logger
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(limiter *ratelimiter.RateLimiter, logger *slog.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
		logger:  logger,
	}
}

// Handler wraps an HTTP handler with rate limiting
func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP
		ip := getClientIP(r)

		// Check rate limit
		resp, err := m.limiter.Allow(ip)
		if err != nil {
			m.logger.Error("rate limit check failed", 
				"error", err,
				"ip", ip,
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !resp.Allowed {
			// Calculate retry after in seconds
			retryAfterSecs := int(time.Until(resp.RetryAfter).Seconds())
			
			// Set headers
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", strconv.Itoa(retryAfterSecs))
			w.WriteHeader(http.StatusTooManyRequests)

			// Create error response
			errResp := ErrorResponse{
				Error:        "Rate limit exceeded",
				Limit:        resp.Limit,
				RequestsMade: resp.RequestsMade,
				RetryAfter:   resp.RetryAfter,
			}

			// Log rate limit exceeded
			m.logger.Info("rate limit exceeded",
				"ip", ip,
				"requests_made", resp.RequestsMade,
				"limit", resp.Limit,
				"retry_after", resp.RetryAfter,
			)

			// Send JSON response
			json.NewEncoder(w).Encode(errResp)
			return
		}

		// Request allowed, proceed
		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := net.ParseIP(xff)
		if ips != nil {
			return ips.String()
		}
	}
	
	// Extract from RemoteAddr
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}
