package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	grpcadapter "github.com/nithibodee/7-solutions-backend-test/internal/adapter/grpc"
	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
)

type validator struct{ err error }

func (v validator) Validate(string) (domain.Claims, error) {
	return domain.Claims{UserID: "u1"}, v.err
}

func invoke(t *testing.T, v domain.TokenValidator, ctx context.Context) error {
	t.Helper()
	interceptor := grpcadapter.AuthUnaryInterceptor(v)
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"},
		func(context.Context, any) (any, error) { return "ok", nil })
	return err
}

func TestAuthInterceptor_NoMetadata(t *testing.T) {
	err := invoke(t, validator{}, context.Background())
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_MissingToken(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	err := invoke(t, validator{}, ctx)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_InvalidToken(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer bad"))
	err := invoke(t, validator{err: errors.New("bad")}, ctx)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_ValidToken(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer good"))
	err := invoke(t, validator{}, ctx)
	require.NoError(t, err)
}
