package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"time"

	pb "github.com/devfullcycle/ratelimiter/proto"
	"github.com/devfullcycle/ratelimiter/middleware"
	"github.com/devfullcycle/ratelimiter/ratelimiter"
	"github.com/devfullcycle/ratelimiter/storage"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedRateLimiterServer
	limiter *ratelimiter.RateLimiter
}

func (s *server) Allow(ctx context.Context, req *pb.AllowRequest) (*pb.AllowResponse, error) {
	resp, err := s.limiter.Allow(req.ClientId)
	if err != nil {
		return nil, err
	}

	return &pb.AllowResponse{
		Allowed:      resp.Allowed,
		RequestsMade: int32(resp.RequestsMade),
		Limit:        int32(resp.Limit),
		RetryAfter:   resp.RetryAfter.Format(time.RFC3339),
	}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	store := storage.NewMemoryStorage()
	limiter := ratelimiter.New(store)

	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		logger.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.NewGrpcRateLimitInterceptor(limiter, logger)),
	)
	pb.RegisterRateLimiterServer(s, &server{limiter: limiter})

	logger.Info("starting gRPC server on :9090")
	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}
