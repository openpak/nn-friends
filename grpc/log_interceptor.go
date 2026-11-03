package grpc

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// logInterceptor writes one line per unary call: full method, status code and duration. Never the
// request, response or metadata, which carry the API key and user data.
func logInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Printf("grpc %s -> %s %dms", info.FullMethod, status.Code(err), time.Since(start).Milliseconds())
	return resp, err
}
