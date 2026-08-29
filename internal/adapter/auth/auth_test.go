package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nithibodee/7-solutions-backend-test/internal/adapter/auth"
	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
)

func TestBcryptHasher_HashAndCompare(t *testing.T) {
	h := auth.NewBcryptHasher(4)

	hash, err := h.Hash("s3cret-password")
	require.NoError(t, err)
	assert.NotEqual(t, "s3cret-password", hash)

	assert.NoError(t, h.Compare(hash, "s3cret-password"))
	assert.ErrorIs(t, h.Compare(hash, "wrong"), domain.ErrInvalidCredentials)
}

func TestJWTManager_IssueThenValidate(t *testing.T) {
	m := auth.NewJWTManager("test-secret", "test-iss", time.Hour)
	u := &domain.User{ID: "user-1", Email: "alice@example.com"}

	token, err := m.Issue(u)
	require.NoError(t, err)

	claims, err := m.Validate(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "alice@example.com", claims.Email)
}

func TestJWTManager_RejectsWrongSecret(t *testing.T) {
	issuer := auth.NewJWTManager("secret-a", "iss", time.Hour)
	verifier := auth.NewJWTManager("secret-b", "iss", time.Hour)

	token, err := issuer.Issue(&domain.User{ID: "1"})
	require.NoError(t, err)

	_, err = verifier.Validate(token)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestJWTManager_RejectsExpiredToken(t *testing.T) {
	m := auth.NewJWTManager("secret", "iss", time.Minute)
	m.SetClock(func() time.Time { return time.Now().Add(-2 * time.Minute) }) // issue in the past
	token, err := m.Issue(&domain.User{ID: "1"})
	require.NoError(t, err)

	m.SetClock(time.Now) // validate in the present -> token is expired
	_, err = m.Validate(token)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestJWTManager_RejectsNoneAlgorithm(t *testing.T) {
	m := auth.NewJWTManager("secret", "iss", time.Hour)
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "1"}).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = m.Validate(unsigned)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestJWTManager_RejectsGarbage(t *testing.T) {
	m := auth.NewJWTManager("secret", "iss", time.Hour)
	_, err := m.Validate("not-a-jwt")
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}
