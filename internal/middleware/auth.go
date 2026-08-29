package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	domain "github.com/nithibodee/7-solutions-backend-test/internal/domain/user"
)

// ClaimsContextKey is the Gin context key under which validated claims are stored.
const ClaimsContextKey = "claims"

// Auth returns middleware that requires a valid Bearer token. On success it
// stores the domain.Claims in the Gin context under ClaimsContextKey.
func Auth(validator domain.TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token, ok := bearerToken(header)
		if !ok {
			abortUnauthorized(c, "missing or malformed Authorization header")
			return
		}
		claims, err := validator.Validate(token)
		if err != nil {
			abortUnauthorized(c, "invalid or expired token")
			return
		}
		c.Set(ClaimsContextKey, claims)
		c.Next()
	}
}

// ClaimsFromContext extracts the claims stored by Auth.
func ClaimsFromContext(c *gin.Context) (domain.Claims, bool) {
	v, ok := c.Get(ClaimsContextKey)
	if !ok {
		return domain.Claims{}, false
	}
	claims, ok := v.(domain.Claims)
	return claims, ok
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(header[len(prefix):]), true
}

func abortUnauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{"code": "unauthorized", "message": msg},
	})
}
