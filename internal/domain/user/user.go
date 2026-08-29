// Package user contains the core domain: the User entity, domain-level errors,
// and the ports (interfaces) that outer layers must implement. It has no
// dependency on any framework, driver, or transport.
package user

import (
	"errors"
	"time"
)

// Domain errors. Adapters translate infrastructure errors into these, and the
// transport layer maps these onto protocol responses (HTTP status, gRPC code).
var (
	// ErrNotFound is returned when a user does not exist.
	ErrNotFound = errors.New("user not found")
	// ErrEmailAlreadyExists is returned when an email is already taken.
	ErrEmailAlreadyExists = errors.New("email already exists")
	// ErrInvalidCredentials is returned when authentication fails.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrEmptyUpdate is returned when an update carries no changes.
	ErrEmptyUpdate = errors.New("update contains no fields")
)

// User is the core entity. Password always holds a hash, never a plaintext
// value, and is never serialised out over any transport.
type User struct {
	ID        string
	Name      string
	Email     string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UpdateFields describes a partial update to a user. A nil pointer means "leave
// unchanged"; a non-nil pointer means "set to this value".
type UpdateFields struct {
	Name  *string
	Email *string
}

// IsEmpty reports whether the update would change nothing.
func (u UpdateFields) IsEmpty() bool {
	return u.Name == nil && u.Email == nil
}
