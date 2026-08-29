package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	appuser "github.com/nithibodee/7-solutions-backend-test/internal/app/user"
	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
	"github.com/nithibodee/7-solutions-backend-test/test/mocks"
)

func newService(t *testing.T) (*mocks.MockRepository, *mocks.MockPasswordHasher, *mocks.MockTokenIssuer, appuser.Service) {
	t.Helper()
	repo := mocks.NewMockRepository(t)
	hasher := mocks.NewMockPasswordHasher(t)
	tokens := mocks.NewMockTokenIssuer(t)
	return repo, hasher, tokens, appuser.NewService(repo, hasher, tokens)
}

func TestRegister_Success(t *testing.T) {
	repo, hasher, _, svc := newService(t)

	repo.EXPECT().GetByEmail(mock.Anything, "alice@example.com").Return(nil, domain.ErrNotFound)
	hasher.EXPECT().Hash("password123").Return("hashed", nil)
	repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
		return u.Email == "alice@example.com" && u.Password == "hashed" && u.Name == "Alice"
	})).RunAndReturn(func(_ context.Context, u *domain.User) error {
		u.ID = "generated-id"
		return nil
	})

	got, err := svc.Register(context.Background(), appuser.RegisterInput{
		Name: "Alice", Email: "  ALICE@example.com ", Password: "password123",
	})

	require.NoError(t, err)
	assert.Equal(t, "generated-id", got.ID)
	assert.Equal(t, "alice@example.com", got.Email)
	assert.False(t, got.CreatedAt.IsZero())
}

func TestCreate_Success(t *testing.T) {
	repo, hasher, _, svc := newService(t)
	repo.EXPECT().GetByEmail(mock.Anything, "carol@example.com").Return(nil, domain.ErrNotFound)
	hasher.EXPECT().Hash(mock.Anything).Return("h", nil)
	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	got, err := svc.Create(context.Background(), appuser.RegisterInput{
		Name: "Carol", Email: "carol@example.com", Password: "password123",
	})

	require.NoError(t, err)
	assert.Equal(t, "carol@example.com", got.Email)
}

func TestRegister_HasherError(t *testing.T) {
	repo, hasher, _, svc := newService(t)
	repo.EXPECT().GetByEmail(mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound)
	hasher.EXPECT().Hash(mock.Anything).Return("", errors.New("hash failed"))

	_, err := svc.Register(context.Background(), appuser.RegisterInput{
		Name: "X", Email: "x@example.com", Password: "password123",
	})

	assert.Error(t, err)
}

func TestRegister_EmailAlreadyExists(t *testing.T) {
	repo, _, _, svc := newService(t)
	repo.EXPECT().GetByEmail(mock.Anything, "bob@example.com").Return(&domain.User{ID: "1"}, nil)

	_, err := svc.Register(context.Background(), appuser.RegisterInput{
		Name: "Bob", Email: "bob@example.com", Password: "password123",
	})

	assert.ErrorIs(t, err, domain.ErrEmailAlreadyExists)
}

func TestRegister_RepositoryErrorOnLookup(t *testing.T) {
	repo, _, _, svc := newService(t)
	boom := errors.New("db down")
	repo.EXPECT().GetByEmail(mock.Anything, mock.Anything).Return(nil, boom)

	_, err := svc.Register(context.Background(), appuser.RegisterInput{
		Name: "X", Email: "x@example.com", Password: "password123",
	})

	assert.ErrorIs(t, err, boom)
}

func TestAuthenticate_Success(t *testing.T) {
	repo, hasher, tokens, svc := newService(t)
	u := &domain.User{ID: "1", Email: "alice@example.com", Password: "hashed"}

	repo.EXPECT().GetByEmail(mock.Anything, "alice@example.com").Return(u, nil)
	hasher.EXPECT().Compare("hashed", "secret").Return(nil)
	tokens.EXPECT().Issue(u).Return("jwt-token", nil)

	token, err := svc.Authenticate(context.Background(), "Alice@example.com", "secret")

	require.NoError(t, err)
	assert.Equal(t, "jwt-token", token)
}

func TestAuthenticate_UnknownEmail(t *testing.T) {
	repo, _, _, svc := newService(t)
	repo.EXPECT().GetByEmail(mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound)

	_, err := svc.Authenticate(context.Background(), "ghost@example.com", "secret")

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	repo, hasher, _, svc := newService(t)
	u := &domain.User{ID: "1", Email: "alice@example.com", Password: "hashed"}
	repo.EXPECT().GetByEmail(mock.Anything, mock.Anything).Return(u, nil)
	hasher.EXPECT().Compare("hashed", "wrong").Return(domain.ErrInvalidCredentials)

	_, err := svc.Authenticate(context.Background(), "alice@example.com", "wrong")

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestGet_NotFound(t *testing.T) {
	repo, _, _, svc := newService(t)
	repo.EXPECT().GetByID(mock.Anything, "missing").Return(nil, domain.ErrNotFound)

	_, err := svc.Get(context.Background(), "missing")

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdate_EmptyPayload(t *testing.T) {
	_, _, _, svc := newService(t)

	_, err := svc.Update(context.Background(), "1", appuser.UpdateInput{})

	assert.ErrorIs(t, err, domain.ErrEmptyUpdate)
}

func TestUpdate_NameOnly(t *testing.T) {
	repo, _, _, svc := newService(t)
	newName := "  New Name "
	updated := &domain.User{ID: "1", Name: "New Name"}
	repo.EXPECT().Update(mock.Anything, "1", mock.MatchedBy(func(f domain.UpdateFields) bool {
		return f.Name != nil && *f.Name == "New Name" && f.Email == nil
	})).Return(updated, nil)

	got, err := svc.Update(context.Background(), "1", appuser.UpdateInput{Name: &newName})

	require.NoError(t, err)
	assert.Equal(t, "New Name", got.Name)
}

func TestUpdate_EmailTakenByAnotherUser(t *testing.T) {
	repo, _, _, svc := newService(t)
	email := "taken@example.com"
	repo.EXPECT().GetByEmail(mock.Anything, email).Return(&domain.User{ID: "other"}, nil)

	_, err := svc.Update(context.Background(), "1", appuser.UpdateInput{Email: &email})

	assert.ErrorIs(t, err, domain.ErrEmailAlreadyExists)
}

func TestUpdate_EmailOwnedBySameUser(t *testing.T) {
	repo, _, _, svc := newService(t)
	email := "mine@example.com"
	repo.EXPECT().GetByEmail(mock.Anything, email).Return(&domain.User{ID: "1"}, nil)
	repo.EXPECT().Update(mock.Anything, "1", mock.Anything).Return(&domain.User{ID: "1", Email: email}, nil)

	got, err := svc.Update(context.Background(), "1", appuser.UpdateInput{Email: &email})

	require.NoError(t, err)
	assert.Equal(t, email, got.Email)
}

func TestDelete_Delegates(t *testing.T) {
	repo, _, _, svc := newService(t)
	repo.EXPECT().Delete(mock.Anything, "1").Return(nil)

	assert.NoError(t, svc.Delete(context.Background(), "1"))
}

func TestCount_Delegates(t *testing.T) {
	repo, _, _, svc := newService(t)
	repo.EXPECT().Count(mock.Anything).Return(int64(42), nil)

	n, err := svc.Count(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(42), n)
}
