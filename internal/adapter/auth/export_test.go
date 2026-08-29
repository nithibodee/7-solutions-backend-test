package auth

import "time"

// SetClock overrides the manager's time source. Test-only.
func (m *JWTManager) SetClock(now func() time.Time) {
	m.now = now
}
