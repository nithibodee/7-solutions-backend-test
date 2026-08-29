package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
)

// ErrInvalidToken is returned when a token is malformed, expired, or signed with
// the wrong key/algorithm.
var ErrInvalidToken = errors.New("invalid token")

// JWTManager implements both domain.TokenIssuer and domain.TokenValidator using
// HMAC-SHA256 (HS256).
type JWTManager struct {
	secret []byte
	issuer string
	ttl    time.Duration
	now    func() time.Time
}

var (
	_ domain.TokenIssuer    = (*JWTManager)(nil)
	_ domain.TokenValidator = (*JWTManager)(nil)
)

// NewJWTManager returns a manager. A ttl <= 0 defaults to 24h.
func NewJWTManager(secret, issuer string, ttl time.Duration) *JWTManager {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &JWTManager{secret: []byte(secret), issuer: issuer, ttl: ttl, now: time.Now}
}

// Issue mints an HS256-signed token for the user.
func (m *JWTManager) Issue(u *domain.User) (string, error) {
	now := m.now()
	claims := jwt.MapClaims{
		"sub":   u.ID,
		"email": u.Email,
		"iss":   m.issuer,
		"iat":   now.Unix(),
		"exp":   now.Add(m.ttl).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Validate verifies the signature and expiry and returns the domain claims.
func (m *JWTManager) Validate(token string) (domain.Claims, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method %v", ErrInvalidToken, t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithTimeFunc(m.now))
	if err != nil || !parsed.Valid {
		return domain.Claims{}, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return domain.Claims{}, ErrInvalidToken
	}
	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	if sub == "" {
		return domain.Claims{}, ErrInvalidToken
	}
	return domain.Claims{UserID: sub, Email: email}, nil
}
