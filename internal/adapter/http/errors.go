package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
)

// errorBody is the single error envelope used by every endpoint.
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

// respondDomainError maps a domain error onto an HTTP response.
func respondDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		respondError(c, http.StatusNotFound, "not_found", "user not found")
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		respondError(c, http.StatusConflict, "email_taken", "email already exists")
	case errors.Is(err, domain.ErrInvalidCredentials):
		respondError(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	case errors.Is(err, domain.ErrEmptyUpdate):
		respondError(c, http.StatusBadRequest, "empty_update", "no updatable fields provided")
	default:
		respondError(c, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}

func respondValidationError(c *gin.Context, err error) {
	respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
}
