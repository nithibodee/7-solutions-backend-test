package user_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	appuser "github.com/nithibodee/7-solutions-backend-test/internal/app/user"
	"github.com/nithibodee/7-solutions-backend-test/test/mocks"
)

func TestRunUserCountLogger_LogsAndStops(t *testing.T) {
	svc := mocks.NewMockService(t)
	var calls atomic.Int32
	svc.EXPECT().Count(mock.Anything).RunAndReturn(func(context.Context) (int64, error) {
		calls.Add(1)
		return 7, nil
	})

	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		appuser.RunUserCountLogger(ctx, svc, log, 10*time.Millisecond)
		close(done)
	}()

	require.Eventually(t, func() bool { return calls.Load() >= 2 }, time.Second, 5*time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("logger did not stop after context cancellation")
	}

	assert.Contains(t, buf.String(), `"total":7`)

	var stopped bool
	for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		var entry map[string]any
		if json.Unmarshal(line, &entry) == nil && entry["msg"] == "user count logger stopped" {
			stopped = true
		}
	}
	assert.True(t, stopped)
}
