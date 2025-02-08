package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/devfullcycle/ratelimiter/ratelimiter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func NewGrpcRateLimitInterceptor(limiter *ratelimiter.RateLimiter, logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Internal, "failed to get metadata")
		}

		// Get client IP from metadata
		clientIP := ""
		if ips := md.Get("x-forwarded-for"); len(ips) > 0 {
			clientIP = ips[0]
		}
		if clientIP == "" {
			if ips := md.Get("x-real-ip"); len(ips) > 0 {
				clientIP = ips[0]
			}
		}

		resp, err := limiter.Allow(clientIP)
		if err != nil {
			logger.Error("rate limit check failed", "error", err)
			return nil, status.Error(codes.Internal, "rate limit check failed")
		}

		if !resp.Allowed {
			header := metadata.New(map[string]string{
				"retry-after": resp.RetryAfter.Format(time.RFC3339),
			})
			grpc.SetHeader(ctx, header)
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}
