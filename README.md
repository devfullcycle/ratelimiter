# Rate Limiter

A flexible and modular rate limiting solution for REST APIs in Go.

## Overview

This rate limiter provides a robust solution for controlling API request rates in Go applications. It features:

- IP-based rate limiting with configurable limits
- Atomic operations for thread-safe request counting
- Temporary blocking of clients exceeding limits
- Structured error responses with retry information
- Modular design for easy extension
- Memory storage with future support for distributed storage

## Installation

```bash
go get github.com/devfullcycle/ratelimiter
```

## Quick Start

```go
package main

import (
    "github.com/devfullcycle/ratelimiter/middleware"
    "github.com/devfullcycle/ratelimiter/ratelimiter"
    "github.com/devfullcycle/ratelimiter/storage"
    "log/slog"
    "net/http"
    "os"
)

func main() {
    // Initialize logger
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    // Create memory storage
    store := storage.NewMemoryStorage()

    // Create rate limiter with default options (100 req/min)
    limiter := ratelimiter.New(store)

    // Create middleware
    rateLimitMiddleware := middleware.NewRateLimitMiddleware(limiter, logger)

    // Apply to your handlers
    http.Handle("/", rateLimitMiddleware.Handler(
        http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("Hello, World!"))
        }),
    ))

    http.ListenAndServe(":8080", nil)
}
```

## Configuration

The rate limiter supports the following configuration options:

### Default Values
- Maximum requests: 100 per minute
- Block duration: 1 minute
- Storage: Memory-based
- Client identification: IP-based

### Customization Options
```go
limiter := ratelimiter.New(
    store,
    ratelimiter.WithMaxRequests(200),           // Change request limit
    ratelimiter.WithTimeWindow(time.Hour),      // Change time window
    ratelimiter.WithBlockDuration(time.Minute), // Change block duration
)
```

## Rate Limit Response

When a client exceeds the rate limit:

### HTTP Response
```
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 60
```

### JSON Response
```json
{
    "error": "Rate limit exceeded",
    "limit": 100,
    "requests_made": 120,
    "retry_after": "2024-02-08T14:30:00.000Z"
}
```

## Development and Testing

### Prerequisites
- Docker and Docker Compose
- Go 1.23 or higher (if running locally)

### Running the Example

1. Start the development container:
   ```bash
   docker compose up -d
   ```

2. Run the example server:
   ```bash
   docker compose exec app sh -c "cd /app && go run examples/memory/memory.go"
   ```

3. Test rate limiting with hey:
   ```bash
   # Test normal requests (should succeed)
   docker compose exec app hey -n 80 -c 10 http://localhost:8080/

   # Test rate limit exceeded
   docker compose exec app hey -n 150 -c 50 http://localhost:8080/
   ```

### Running Tests
```bash
docker compose exec app sh -c "cd /app && go test ./..."
```

## Architecture

The rate limiter follows a modular design with three main components:

1. Core Rate Limiter (`ratelimiter`)
   - Handles rate limiting logic
   - Configurable via options pattern
   - Thread-safe operations

2. Storage Interface (`storage`)
   - Pluggable storage backends
   - Default memory implementation
   - Atomic operations for thread safety

3. HTTP Middleware (`middleware`)
   - Easy integration with HTTP servers
   - Structured logging with slog
   - Standard error responses

## Continuous Integration

The project uses GitHub Actions for CI/CD with the following checks:

1. Unit Tests
   - All tests are run with race detection enabled
   - Tests must pass before merging

2. Load Testing
   - HTTP endpoints tested with hey tool
   - gRPC endpoints tested with ghz tool
   - Verifies rate limiting behavior under load
   - Tests both normal operation and rate limit enforcement

3. Integration Tests
   - Tests run in containers via docker-compose
   - Both HTTP and gRPC implementations tested
   - Validates end-to-end functionality

## License

MIT License
