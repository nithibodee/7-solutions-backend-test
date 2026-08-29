package user

import "context"

// Repository is the persistence port for users. The MongoDB adapter implements
// it; tests use a generated mock.
type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context) ([]User, error)
	Update(ctx context.Context, id string, fields UpdateFields) (*User, error)
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

// PasswordHasher abstracts password hashing so the domain never depends on a
// concrete algorithm (bcrypt in production).
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error
}

// Claims is the authenticated identity carried by a validated token.
type Claims struct {
	UserID string
	Email  string
}

// TokenIssuer mints authentication tokens for a user.
type TokenIssuer interface {
	Issue(u *User) (string, error)
}

// TokenValidator verifies a token string and extracts its claims.
type TokenValidator interface {
	Validate(token string) (Claims, error)
}
