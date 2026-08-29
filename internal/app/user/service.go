// Package user (app) holds the application use-cases for user management. It
// orchestrates the domain ports and enforces business rules; it knows nothing
// about HTTP, gRPC, or MongoDB.
package user

import (
	"context"
	"errors"
	"strings"
	"time"

	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
)

// Service exposes the user management use-cases. The transport adapters depend
// on this interface, which keeps them decoupled from the concrete logic and
// makes them trivial to test with a mock.
type Service interface {
	Register(ctx context.Context, in RegisterInput) (*domain.User, error)
	Authenticate(ctx context.Context, email, password string) (string, error)
	Create(ctx context.Context, in RegisterInput) (*domain.User, error)
	Get(ctx context.Context, id string) (*domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	Update(ctx context.Context, id string, in UpdateInput) (*domain.User, error)
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

// RegisterInput is the payload for creating a user.
type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

// UpdateInput is a partial update. Nil means "leave unchanged".
type UpdateInput struct {
	Name  *string
	Email *string
}

type service struct {
	repo   domain.Repository
	hasher domain.PasswordHasher
	tokens domain.TokenIssuer
	now    func() time.Time
}

var _ Service = (*service)(nil)

// NewService wires the use-cases with their ports.
func NewService(repo domain.Repository, hasher domain.PasswordHasher, tokens domain.TokenIssuer) Service {
	return &service{repo: repo, hasher: hasher, tokens: tokens, now: time.Now}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *service) create(ctx context.Context, in RegisterInput) (*domain.User, error) {
	email := normalizeEmail(in.Email)

	// Pre-check for a friendlier error; the unique index is the real guard
	// against the race between check and insert.
	switch _, err := s.repo.GetByEmail(ctx, email); {
	case err == nil:
		return nil, domain.ErrEmailAlreadyExists
	case !errors.Is(err, domain.ErrNotFound):
		return nil, err
	}

	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	u := &domain.User{
		Name:      strings.TrimSpace(in.Name),
		Email:     email,
		Password:  hash,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Register creates a user via the public sign-up flow.
func (s *service) Register(ctx context.Context, in RegisterInput) (*domain.User, error) {
	return s.create(ctx, in)
}

// Create creates a user via an authenticated admin-style call. Same rules as
// Register today; kept separate so the two flows can diverge later.
func (s *service) Create(ctx context.Context, in RegisterInput) (*domain.User, error) {
	return s.create(ctx, in)
}

// Authenticate verifies credentials and returns a signed token.
func (s *service) Authenticate(ctx context.Context, email, password string) (string, error) {
	u, err := s.repo.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", domain.ErrInvalidCredentials
		}
		return "", err
	}
	if err := s.hasher.Compare(u.Password, password); err != nil {
		return "", domain.ErrInvalidCredentials
	}
	return s.tokens.Issue(u)
}

func (s *service) Get(ctx context.Context, id string) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) List(ctx context.Context) ([]domain.User, error) {
	return s.repo.List(ctx)
}

func (s *service) Update(ctx context.Context, id string, in UpdateInput) (*domain.User, error) {
	fields := domain.UpdateFields{}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		fields.Name = &name
	}
	if in.Email != nil {
		email := normalizeEmail(*in.Email)
		fields.Email = &email

		existing, err := s.repo.GetByEmail(ctx, email)
		switch {
		case err == nil && existing.ID != id:
			return nil, domain.ErrEmailAlreadyExists
		case err != nil && !errors.Is(err, domain.ErrNotFound):
			return nil, err
		}
	}
	if fields.IsEmpty() {
		return nil, domain.ErrEmptyUpdate
	}
	return s.repo.Update(ctx, id, fields)
}

func (s *service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) Count(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}
