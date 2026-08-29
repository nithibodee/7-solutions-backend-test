// Package grpc adapts the user management use-cases to a gRPC service.
package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	userv1 "github.com/nithibodee/7-solutions-backend-test/api/proto/user/v1"
	appuser "github.com/nithibodee/7-solutions-backend-test/internal/app/user"
	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
)

// Server implements userv1.UserServiceServer on top of the application service.
type Server struct {
	userv1.UnimplementedUserServiceServer
	svc appuser.Service
}

var _ userv1.UserServiceServer = (*Server)(nil)

// NewServer returns a gRPC server backed by the given application service.
func NewServer(svc appuser.Service) *Server {
	return &Server{svc: svc}
}

// CreateUser creates a user.
func (s *Server) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.UserResponse, error) {
	u, err := s.svc.Create(ctx, appuser.RegisterInput{
		Name:     req.GetName(),
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, toStatusErr(err)
	}
	return toProto(u), nil
}

// GetUser fetches a user by ID.
func (s *Server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.UserResponse, error) {
	u, err := s.svc.Get(ctx, req.GetId())
	if err != nil {
		return nil, toStatusErr(err)
	}
	return toProto(u), nil
}

func toProto(u *domain.User) *userv1.UserResponse {
	return &userv1.UserResponse{
		Id:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
}

func toStatusErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, "user not found")
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		return status.Error(codes.AlreadyExists, "email already exists")
	case errors.Is(err, domain.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, "invalid credentials")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
