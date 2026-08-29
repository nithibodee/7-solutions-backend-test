package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	appuser "github.com/nithibodee/7-solutions-backend-test/internal/app/user"
)

// Handler holds the HTTP handlers for the user management API.
type Handler struct {
	svc appuser.Service
}

// NewHandler returns a Handler backed by the given application service.
func NewHandler(svc appuser.Service) *Handler {
	return &Handler{svc: svc}
}

// Register handles POST /auth/register.
func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}
	u, err := h.svc.Register(c.Request.Context(), appuser.RegisterInput(req))
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toUserResponse(u))
}

// Login handles POST /auth/login.
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}
	token, err := h.svc.Authenticate(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, tokenResponse{Token: token})
}

// CreateUser handles POST /api/users.
func (h *Handler) CreateUser(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}
	u, err := h.svc.Create(c.Request.Context(), appuser.RegisterInput(req))
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toUserResponse(u))
}

// ListUsers handles GET /api/users.
func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.svc.List(c.Request.Context())
	if err != nil {
		respondDomainError(c, err)
		return
	}
	out := make([]userResponse, len(users))
	for i := range users {
		out[i] = toUserResponse(&users[i])
	}
	c.JSON(http.StatusOK, gin.H{"users": out})
}

// GetUser handles GET /api/users/:id.
func (h *Handler) GetUser(c *gin.Context) {
	u, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserResponse(u))
}

// UpdateUser handles PATCH /api/users/:id.
func (h *Handler) UpdateUser(c *gin.Context) {
	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err)
		return
	}
	u, err := h.svc.Update(c.Request.Context(), c.Param("id"), appuser.UpdateInput{Name: req.Name, Email: req.Email})
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserResponse(u))
}

// DeleteUser handles DELETE /api/users/:id.
func (h *Handler) DeleteUser(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		respondDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
