// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime settings.
type Config struct {
	HTTPPort        string
	GRPCPort        string
	MongoURI        string
	MongoDB         string
	JWTSecret       string
	JWTIssuer       string
	JWTTTL          time.Duration
	BcryptCost      int
	CountInterval   time.Duration
	ShutdownTimeout time.Duration
	GRPCAuth        bool
}

// Load reads configuration from the environment, applying defaults. It returns
// an error only when a required secret is missing.
func Load() (Config, error) {
	c := Config{
		HTTPPort:        getenv("HTTP_PORT", "8080"),
		GRPCPort:        getenv("GRPC_PORT", "9090"),
		MongoURI:        getenv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:         getenv("MONGO_DB", "usermgmt"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		JWTIssuer:       getenv("JWT_ISSUER", "user-management-api"),
		JWTTTL:          getdur("JWT_TTL", 24*time.Hour),
		BcryptCost:      getint("BCRYPT_COST", 10),
		CountInterval:   getdur("USER_COUNT_INTERVAL", 10*time.Second),
		ShutdownTimeout: getdur("SHUTDOWN_TIMEOUT", 10*time.Second),
		GRPCAuth:        getbool("GRPC_AUTH", false),
	}
	if c.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	return c, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getint(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getbool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getdur(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
