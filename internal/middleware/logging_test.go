package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nithibodee/7-solutions-backend-test/internal/middleware"
)

func TestLogging_RecordsMethodPathStatusDuration(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	r := gin.New()
	r.Use(middleware.Logging(log))
	r.GET("/things", func(c *gin.Context) { c.Status(http.StatusTeapot) })

	req := httptest.NewRequest(http.MethodGet, "/things?q=1", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry))
	assert.Equal(t, "http request", entry["msg"])
	assert.Equal(t, "GET", entry["method"])
	assert.Equal(t, "/things?q=1", entry["path"])
	assert.Equal(t, float64(http.StatusTeapot), entry["status"])
	assert.Contains(t, entry, "duration")
}
