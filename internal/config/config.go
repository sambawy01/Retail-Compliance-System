// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"os"
)

// Config holds all application configuration values.
type Config struct {
	HTTPPort        string
	DatabaseURL     string
	JWTPrivateKey   string
	JWTPublicKey    string
	B2Bucket        string
	B2Key           string
	LogLevel        string
	Env             string
	AllowedOrigins  string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	c := &Config{
		HTTPPort:       envOr("HTTP_PORT", "8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		JWTPrivateKey:  envOr("JWT_PRIVATE_KEY_PATH", "keys/private.pem"),
		JWTPublicKey:   envOr("JWT_PUBLIC_KEY_PATH", "keys/public.pem"),
		B2Bucket:       os.Getenv("B2_BUCKET"),
		B2Key:          os.Getenv("B2_KEY"),
		LogLevel:       envOr("LOG_LEVEL", "info"),
		Env:            envOr("APP_ENV", "dev"),
		AllowedOrigins: os.Getenv("ALLOWED_ORIGINS"),
	}
	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}
	return c, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}