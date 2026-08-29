package http

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
	"github.com/nithibodee/7-solutions-backend-test/internal/middleware"
)

// NewRouter builds the Gin engine with all middleware and routes wired.
func NewRouter(h *Handler, validator domain.TokenValidator, log *slog.Logger) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.Logging(log))

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	auth := r.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
	}

	api := r.Group("/api", middleware.Auth(validator))
	{
		api.POST("/users", h.CreateUser)
		api.GET("/users", h.ListUsers)
		api.GET("/users/:id", h.GetUser)
		api.PATCH("/users/:id", h.UpdateUser)
		api.DELETE("/users/:id", h.DeleteUser)
	}

	return r
}
