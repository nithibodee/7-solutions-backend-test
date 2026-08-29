// Package auth provides adapters for the domain's PasswordHasher, TokenIssuer,
// and TokenValidator ports.
package auth

import (
	"golang.org/x/crypto/bcrypt"

	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
)

// BcryptHasher implements domain.PasswordHasher using bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher returns a hasher. A cost <= 0 falls back to bcrypt.DefaultCost.
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

// Hash returns the bcrypt hash of a plaintext password.
func (h *BcryptHasher) Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Compare returns nil when plain matches hash, domain.ErrInvalidCredentials
// otherwise.
func (h *BcryptHasher) Compare(hash, plain string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		return domain.ErrInvalidCredentials
	}
	return nil
}
