package grpc_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	userv1 "github.com/nithibodee/7-solutions-backend-test/api/proto/user/v1"
	grpcadapter "github.com/nithibodee/7-solutions-backend-test/internal/adapter/grpc"
	appuser "github.com/nithibodee/7-solutions-backend-test/internal/app/user"
	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
	"github.com/nithibodee/7-solutions-backend-test/test/mocks"
)

func newClient(t *testing.T, svc appuser.Service) userv1.UserServiceClient {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := googlegrpc.NewServer()
	userv1.RegisterUserServiceServer(srv, grpcadapter.NewServer(svc))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := googlegrpc.NewClient("passthrough:///bufnet",
		googlegrpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		googlegrpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return userv1.NewUserServiceClient(conn)
}

func TestGRPC_CreateUser(t *testing.T) {
	svc := mocks.NewMockService(t)
	svc.EXPECT().Create(mock.Anything, appuser.RegisterInput{
		Name: "Alice", Email: "alice@example.com", Password: "password123",
	}).Return(&domain.User{ID: "1", Name: "Alice", Email: "alice@example.com"}, nil)

	client := newClient(t, svc)
	resp, err := client.CreateUser(context.Background(), &userv1.CreateUserRequest{
		Name: "Alice", Email: "alice@example.com", Password: "password123",
	})

	require.NoError(t, err)
	assert.Equal(t, "1", resp.GetId())
	assert.Equal(t, "alice@example.com", resp.GetEmail())
}

func TestGRPC_GetUser_NotFound(t *testing.T) {
	svc := mocks.NewMockService(t)
	svc.EXPECT().Get(mock.Anything, "missing").Return(nil, domain.ErrNotFound)

	client := newClient(t, svc)
	_, err := client.GetUser(context.Background(), &userv1.GetUserRequest{Id: "missing"})

	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPC_CreateUser_Conflict(t *testing.T) {
	svc := mocks.NewMockService(t)
	svc.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, domain.ErrEmailAlreadyExists)

	client := newClient(t, svc)
	_, err := client.CreateUser(context.Background(), &userv1.CreateUserRequest{Email: "dup@example.com"})

	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}
