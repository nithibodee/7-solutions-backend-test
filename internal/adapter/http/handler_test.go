package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	httpadapter "github.com/nithibodee/7-solutions-backend-test/internal/adapter/http"
	appuser "github.com/nithibodee/7-solutions-backend-test/internal/app/user"
	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
	"github.com/nithibodee/7-solutions-backend-test/test/mocks"
)

func init() { gin.SetMode(gin.TestMode) }

// stubValidator lets protected-route tests inject an identity without real JWTs.
type stubValidator struct{ err error }

func (s stubValidator) Validate(string) (domain.Claims, error) {
	if s.err != nil {
		return domain.Claims{}, s.err
	}
	return domain.Claims{UserID: "caller", Email: "caller@example.com"}, nil
}

func newTestServer(t *testing.T) (*mocks.MockService, http.Handler) {
	t.Helper()
	svc := mocks.NewMockService(t)
	r := httpadapter.NewRouter(httpadapter.NewHandler(svc), stubValidator{}, testLogger())
	return svc, r
}

func doJSON(h http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRegister_Created(t *testing.T) {
	svc, srv := newTestServer(t)
	now := time.Now().UTC()
	svc.EXPECT().Register(mock.Anything, appuser.RegisterInput{
		Name: "Alice", Email: "alice@example.com", Password: "password123",
	}).Return(&domain.User{ID: "1", Name: "Alice", Email: "alice@example.com", CreatedAt: now, UpdatedAt: now}, nil)

	rec := doJSON(srv, http.MethodPost, "/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"password123"}`, "")

	require.Equal(t, http.StatusCreated, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "alice@example.com", body["email"])
	_, hasPassword := body["password"]
	assert.False(t, hasPassword, "password must never be serialised")
}

func TestRegister_ValidationError(t *testing.T) {
	_, srv := newTestServer(t)

	rec := doJSON(srv, http.MethodPost, "/auth/register",
		`{"name":"Alice","email":"not-an-email","password":"short"}`, "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_request")
}

func TestRegister_Conflict(t *testing.T) {
	svc, srv := newTestServer(t)
	svc.EXPECT().Register(mock.Anything, mock.Anything).Return(nil, domain.ErrEmailAlreadyExists)

	rec := doJSON(srv, http.MethodPost, "/auth/register",
		`{"name":"Bob","email":"bob@example.com","password":"password123"}`, "")

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "email_taken")
}

func TestLogin_Success(t *testing.T) {
	svc, srv := newTestServer(t)
	svc.EXPECT().Authenticate(mock.Anything, "alice@example.com", "password123").Return("jwt-abc", nil)

	rec := doJSON(srv, http.MethodPost, "/auth/login",
		`{"email":"alice@example.com","password":"password123"}`, "")

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "jwt-abc", body["token"])
}

func TestLogin_BadCredentials(t *testing.T) {
	svc, srv := newTestServer(t)
	svc.EXPECT().Authenticate(mock.Anything, mock.Anything, mock.Anything).Return("", domain.ErrInvalidCredentials)

	rec := doJSON(srv, http.MethodPost, "/auth/login",
		`{"email":"alice@example.com","password":"nope"}`, "")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestProtectedRoute_RequiresToken(t *testing.T) {
	_, srv := newTestServer(t)

	rec := doJSON(srv, http.MethodGet, "/api/users", "", "")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "unauthorized")
}

func TestListUsers_OK(t *testing.T) {
	svc, srv := newTestServer(t)
	svc.EXPECT().List(mock.Anything).Return([]domain.User{
		{ID: "1", Name: "A", Email: "a@example.com"},
		{ID: "2", Name: "B", Email: "b@example.com"},
	}, nil)

	rec := doJSON(srv, http.MethodGet, "/api/users", "", "any-token")

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Users []map[string]any `json:"users"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Len(t, body.Users, 2)
}

func TestGetUser_NotFound(t *testing.T) {
	svc, srv := newTestServer(t)
	svc.EXPECT().Get(mock.Anything, "missing").Return(nil, domain.ErrNotFound)

	rec := doJSON(srv, http.MethodGet, "/api/users/missing", "", "any-token")

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateUser_OK(t *testing.T) {
	svc, srv := newTestServer(t)
	svc.EXPECT().Update(mock.Anything, "1", mock.MatchedBy(func(in appuser.UpdateInput) bool {
		return in.Name != nil && *in.Name == "Renamed" && in.Email == nil
	})).Return(&domain.User{ID: "1", Name: "Renamed"}, nil)

	rec := doJSON(srv, http.MethodPatch, "/api/users/1", `{"name":"Renamed"}`, "any-token")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Renamed")
}

func TestUpdateUser_EmptyBody(t *testing.T) {
	svc, srv := newTestServer(t)
	svc.EXPECT().Update(mock.Anything, "1", appuser.UpdateInput{}).Return(nil, domain.ErrEmptyUpdate)

	rec := doJSON(srv, http.MethodPatch, "/api/users/1", `{}`, "any-token")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteUser_NoContent(t *testing.T) {
	svc, srv := newTestServer(t)
	svc.EXPECT().Delete(mock.Anything, "1").Return(nil)

	rec := doJSON(srv, http.MethodDelete, "/api/users/1", "", "any-token")

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestCreateUser_ViaAPI(t *testing.T) {
	svc, srv := newTestServer(t)
	svc.EXPECT().Create(mock.Anything, mock.Anything).Return(&domain.User{ID: "9", Email: "new@example.com"}, nil)

	rec := doJSON(srv, http.MethodPost, "/api/users",
		`{"name":"New","email":"new@example.com","password":"password123"}`, "any-token")

	assert.Equal(t, http.StatusCreated, rec.Code)
}
