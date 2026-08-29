package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
	"github.com/nithibodee/7-solutions-backend-test/internal/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

type fakeValidator struct {
	claims domain.Claims
	err    error
}

func (f fakeValidator) Validate(string) (domain.Claims, error) { return f.claims, f.err }

func run(v domain.TokenValidator, authHeader string) (*httptest.ResponseRecorder, domain.Claims, bool) {
	r := gin.New()
	var gotClaims domain.Claims
	var gotOK bool
	r.GET("/x", middleware.Auth(v), func(c *gin.Context) {
		gotClaims, gotOK = middleware.ClaimsFromContext(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec, gotClaims, gotOK
}

func TestAuth_ValidToken(t *testing.T) {
	rec, claims, ok := run(fakeValidator{claims: domain.Claims{UserID: "u1", Email: "u1@example.com"}}, "Bearer good")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, ok)
	assert.Equal(t, "u1", claims.UserID)
}

func TestAuth_MissingHeader(t *testing.T) {
	rec, _, _ := run(fakeValidator{}, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing or malformed")
}

func TestAuth_WrongScheme(t *testing.T) {
	rec, _, _ := run(fakeValidator{}, "Basic abc123")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_InvalidToken(t *testing.T) {
	rec, _, _ := run(fakeValidator{err: errors.New("bad")}, "Bearer bad")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid or expired token")
}
