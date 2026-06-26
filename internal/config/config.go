// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"os"
)

// Config holds all application configuration values.
type Config struct {
	Port            string
	DatabaseURL     string
	JWTPrivateKey   string // file path
	JWTPublicKey    string // file path
	JWTPrivateKeyB64 string // base64-encoded key content
	JWTPublicKeyB64  string // base64-encoded key content
	RedisURL        string
	B2Bucket        string
	B2KeyID         string
	B2AppKey        string
	LogLevel        string
	Env             string
	AllowedOrigins  string
	TelegramBotToken string
	TelegramChatID   string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	c := &Config{
		Port:             envOr("PORT", envOr("HTTP_PORT", "8080")),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		JWTPrivateKey:    envOr("JWT_PRIVATE_KEY_PATH", ""),
		JWTPublicKey:     envOr("JWT_PUBLIC_KEY_PATH", ""),
		JWTPrivateKeyB64: os.Getenv("JWT_PRIVATE_KEY_B64"),
		JWTPublicKeyB64:  os.Getenv("JWT_PUBLIC_KEY_B64"),
		RedisURL:         os.Getenv("REDIS_URL"),
		B2Bucket:         os.Getenv("B2_BUCKET"),
		B2KeyID:          os.Getenv("B2_KEY_ID"),
		B2AppKey:         os.Getenv("B2_APP_KEY"),
		LogLevel:         envOr("LOG_LEVEL", "info"),
		Env:              envOr("ENV", envOr("APP_ENV", "development")),
		AllowedOrigins:   envOr("ALLOWED_ORIGINS", "*"),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:   os.Getenv("TELEGRAM_CHAT_ID"),
	}
	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}
	if c.Env == "production" {
		if c.JWTPrivateKey == "" && c.JWTPrivateKeyB64 == "" {
			return nil, fmt.Errorf("config: JWT_PRIVATE_KEY_PATH or JWT_PRIVATE_KEY_B64 is required in production")
		}
		if c.JWTPublicKey == "" && c.JWTPublicKeyB64 == "" {
			return nil, fmt.Errorf("config: JWT_PUBLIC_KEY_PATH or JWT_PUBLIC_KEY_B64 is required in production")
		}
	}
	return c, nil
}

// IsProduction returns true when running in production.
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}