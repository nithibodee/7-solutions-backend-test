package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nithibodee/7-solutions-backend-test/internal/platform/config"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "s")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.HTTPPort)
	assert.Equal(t, "9090", cfg.GRPCPort)
	assert.Equal(t, 10*time.Second, cfg.CountInterval)
	assert.Equal(t, 24*time.Hour, cfg.JWTTTL)
	assert.False(t, cfg.GRPCAuth)
}

func TestLoad_MissingSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	_, err := config.Load()
	assert.Error(t, err)
}

func TestLoad_Overrides(t *testing.T) {
	t.Setenv("JWT_SECRET", "s")
	t.Setenv("HTTP_PORT", "3000")
	t.Setenv("USER_COUNT_INTERVAL", "30s")
	t.Setenv("GRPC_AUTH", "true")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "3000", cfg.HTTPPort)
	assert.Equal(t, 30*time.Second, cfg.CountInterval)
	assert.True(t, cfg.GRPCAuth)
}
