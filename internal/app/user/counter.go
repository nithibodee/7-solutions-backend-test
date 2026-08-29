package user

import (
	"context"
	"log/slog"
	"time"
)

// RunUserCountLogger logs the total number of users every interval until ctx is
// cancelled. It recovers from panics so a transient failure never kills the
// goroutine.
func RunUserCountLogger(ctx context.Context, svc Service, log *slog.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("user count logger stopped")
			return
		case <-ticker.C:
			logCount(ctx, svc, log)
		}
	}
}

func logCount(ctx context.Context, svc Service, log *slog.Logger) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("user count logger recovered from panic", slog.Any("panic", r))
		}
	}()

	countCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	n, err := svc.Count(countCtx)
	if err != nil {
		log.Error("failed to count users", slog.Any("error", err))
		return
	}
	log.Info("user count", slog.Int64("total", n))
}
