package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/devfullcycle/ratelimiter/middleware"
	"github.com/devfullcycle/ratelimiter/ratelimiter"
	"github.com/devfullcycle/ratelimiter/storage"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create memory storage
	store := storage.NewMemoryStorage()

	// Create rate limiter with default options
	limiter := ratelimiter.New(store)

	// Create middleware
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(limiter, logger)

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Hello, World!"}`))
	})

	// Wrap handler with rate limiting middleware
	http.Handle("/", rateLimitMiddleware.Handler(handler))

	// Start server
	logger.Info("starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
