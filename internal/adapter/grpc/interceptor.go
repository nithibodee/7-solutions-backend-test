package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
)

// AuthUnaryInterceptor validates a Bearer token supplied in the "authorization"
// metadata key. It is only installed when GRPC_AUTH is enabled.
func AuthUnaryInterceptor(validator domain.TokenValidator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		values := md.Get("authorization")
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}
		token := values[0]
		if len(token) > 7 && (token[:7] == "Bearer " || token[:7] == "bearer ") {
			token = token[7:]
		}
		if _, err := validator.Validate(token); err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		return handler(ctx, req)
	}
}
