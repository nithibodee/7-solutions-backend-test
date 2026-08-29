package http_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var logBuf bytes.Buffer

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestLoggingMiddleware_RecordsRequest(t *testing.T) {
	logBuf.Reset()
	_, srv := newTestServer(t)

	doJSON(srv, http.MethodGet, "/healthz", "", "")

	var found bool
	for _, line := range bytes.Split(bytes.TrimSpace(logBuf.Bytes()), []byte("\n")) {
		var entry map[string]any
		require.NoError(t, json.Unmarshal(line, &entry))
		if entry["msg"] == "http request" {
			found = true
			assert.Equal(t, "GET", entry["method"])
			assert.Equal(t, "/healthz", entry["path"])
			assert.Equal(t, float64(200), entry["status"])
			assert.Contains(t, entry, "duration")
		}
	}
	assert.True(t, found, "expected an 'http request' log entry")
}
