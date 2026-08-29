// Package http exposes the user management use-cases over a Gin HTTP API.
package http

import (
	"time"

	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
)

type registerRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// updateRequest uses pointers so an omitted field is distinguishable from an
// empty value. At least one must be present.
type updateRequest struct {
	Name  *string `json:"name" binding:"omitempty,min=1"`
	Email *string `json:"email" binding:"omitempty,email"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toUserResponse(u *domain.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
