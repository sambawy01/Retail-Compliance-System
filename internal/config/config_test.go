package config

import (
	"os"
	"strings"
	"testing"
)

// setEnv sets an env var and registers cleanup to restore it.
func setEnv(t *testing.T, key, val string) {
	t.Helper()
	old, ok := os.LookupEnv(key)
	os.Setenv(key, val)
	t.Cleanup(func() {
		if ok {
			os.Setenv(key, old)
		} else {
			os.Unsetenv(key)
		}
	})
}

// clearEnv clears all config-related env vars for a clean test baseline.
func clearEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"PORT", "HTTP_PORT", "DATABASE_URL", "JWT_PRIVATE_KEY_PATH", "JWT_PUBLIC_KEY_PATH",
		"JWT_PRIVATE_KEY_B64", "JWT_PUBLIC_KEY_B64", "REDIS_URL", "B2_BUCKET", "B2_KEY_ID",
		"B2_APP_KEY", "LOG_LEVEL", "ENV", "APP_ENV", "ALLOWED_ORIGINS",
		"TELEGRAM_BOT_TOKEN", "TELEGRAM_CHAT_ID",
	}
	for _, k := range keys {
		old, ok := os.LookupEnv(k)
		os.Unsetenv(k)
		t.Cleanup(func() {
			if ok {
				os.Setenv(k, old)
			} else {
				os.Unsetenv(k)
			}
		})
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)
	setEnv(t, "DATABASE_URL", "postgres://localhost/testdb")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port: got %q, want %q", cfg.Port, "8080")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel: got %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.Env != "development" {
		t.Errorf("Env: got %q, want %q", cfg.Env, "development")
	}
	if cfg.AllowedOrigins != "*" {
		t.Errorf("AllowedOrigins: got %q, want %q", cfg.AllowedOrigins, "*")
	}
	if cfg.DatabaseURL != "postgres://localhost/testdb" {
		t.Errorf("DatabaseURL: got %q, want %q", cfg.DatabaseURL, "postgres://localhost/testdb")
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	clearEnv(t)
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Errorf("expected DATABASE_URL error, got %q", err.Error())
	}
}

func TestLoad_PortFromHTTP_PORT(t *testing.T) {
	clearEnv(t)
	setEnv(t, "DATABASE_URL", "postgres://localhost/testdb")
	setEnv(t, "HTTP_PORT", "3000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "3000" {
		t.Errorf("Port: got %q, want %q", cfg.Port, "3000")
	}
}

func TestLoad_PortPreferredOverHTTP_PORT(t *testing.T) {
	clearEnv(t)
	setEnv(t, "DATABASE_URL", "postgres://localhost/testdb")
	setEnv(t, "HTTP_PORT", "3000")
	setEnv(t, "PORT", "9090")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port: got %q, want %q (PORT should take precedence over HTTP_PORT)", cfg.Port, "9090")
	}
}

func TestLoad_EnvFromAPP_ENV(t *testing.T) {
	clearEnv(t)
	setEnv(t, "DATABASE_URL", "postgres://localhost/testdb")
	setEnv(t, "APP_ENV", "staging")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Env != "staging" {
		t.Errorf("Env: got %q, want %q", cfg.Env, "staging")
	}
}

func TestLoad_EnvPreferredOverAPP_ENV(t *testing.T) {
	clearEnv(t)
	setEnv(t, "DATABASE_URL", "postgres://localhost/testdb")
	setEnv(t, "APP_ENV", "staging")
	setEnv(t, "ENV", "production")
	setEnv(t, "JWT_PRIVATE_KEY_B64", "base64key==")
	setEnv(t, "JWT_PUBLIC_KEY_B64", "base64key==")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Env != "production" {
		t.Errorf("Env: got %q, want %q (ENV should take precedence over APP_ENV)", cfg.Env, "production")
	}
}

func TestLoad_ProductionRequiresJWTKeys(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T)
		wantErr    string
		shouldFail bool
	}{
		{
			name: "production with no JWT keys",
			setup: func(t *testing.T) {
				setEnv(t, "DATABASE_URL", "postgres://localhost/testdb")
				setEnv(t, "ENV", "production")
			},
			wantErr:    "JWT_PRIVATE_KEY_PATH or JWT_PRIVATE_KEY_B64 is required in production",
			shouldFail: true,
		},
		{
			name: "production with private key path only",
			setup: func(t *testing.T) {
				setEnv(t, "DATABASE_URL", "postgres://localhost/testdb")
				setEnv(t, "ENV", "production")
				setEnv(t, "JWT_PRIVATE_KEY_PATH", "/keys/private.pem")
			},
			wantErr:    "JWT_PUBLIC_KEY_PATH or JWT_PUBLIC_KEY_B64 is required in production",
			shouldFail: true,
		},
		{
			name: "production with both file paths",
			setup: func(t *testing.T) {
				setEnv(t, "DATABASE_URL", "postgres://localhost/testdb")
				setEnv(t, "ENV", "production")
				setEnv(t, "JWT_PRIVATE_KEY_PATH", "/keys/private.pem")
				setEnv(t, "JWT_PUBLIC_KEY_PATH", "/keys/public.pem")
			},
			shouldFail: false,
		},
		{
			name: "production with both base64 keys",
			setup: func(t *testing.T) {
				setEnv(t, "DATABASE_URL", "postgres://localhost/testdb")
				setEnv(t, "ENV", "production")
				setEnv(t, "JWT_PRIVATE_KEY_B64", "base64key==")
				setEnv(t, "JWT_PUBLIC_KEY_B64", "base64key==")
			},
			shouldFail: false,
		},
		{
			name: "production with file path for private, base64 for public",
			setup: func(t *testing.T) {
				setEnv(t, "DATABASE_URL", "postgres://localhost/testdb")
				setEnv(t, "ENV", "production")
				setEnv(t, "JWT_PRIVATE_KEY_PATH", "/keys/private.pem")
				setEnv(t, "JWT_PUBLIC_KEY_B64", "base64key==")
			},
			shouldFail: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(t)
			tc.setup(t)
			cfg, err := Load()
			if tc.shouldFail {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("expected error containing %q, got %q", tc.wantErr, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if cfg == nil {
					t.Fatal("expected non-nil config")
				}
			}
		})
	}
}

func TestLoad_AllOptionalFields(t *testing.T) {
	clearEnv(t)
	setEnv(t, "DATABASE_URL", "postgres://localhost/testdb")
	setEnv(t, "REDIS_URL", "redis://localhost:6379")
	setEnv(t, "B2_BUCKET", "mybucket")
	setEnv(t, "B2_KEY_ID", "key123")
	setEnv(t, "B2_APP_KEY", "appkey456")
	setEnv(t, "JWT_PRIVATE_KEY_PATH", "/keys/private.pem")
	setEnv(t, "JWT_PUBLIC_KEY_PATH", "/keys/public.pem")
	setEnv(t, "JWT_PRIVATE_KEY_B64", "privbase64==")
	setEnv(t, "JWT_PUBLIC_KEY_B64", "pubbase64==")
	setEnv(t, "TELEGRAM_BOT_TOKEN", "token123")
	setEnv(t, "TELEGRAM_CHAT_ID", "chat456")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.RedisURL != "redis://localhost:6379" {
		t.Errorf("RedisURL: got %q, want %q", cfg.RedisURL, "redis://localhost:6379")
	}
	if cfg.B2Bucket != "mybucket" {
		t.Errorf("B2Bucket: got %q, want %q", cfg.B2Bucket, "mybucket")
	}
	if cfg.B2KeyID != "key123" {
		t.Errorf("B2KeyID: got %q, want %q", cfg.B2KeyID, "key123")
	}
	if cfg.B2AppKey != "appkey456" {
		t.Errorf("B2AppKey: got %q, want %q", cfg.B2AppKey, "appkey456")
	}
	if cfg.JWTPrivateKey != "/keys/private.pem" {
		t.Errorf("JWTPrivateKey: got %q, want %q", cfg.JWTPrivateKey, "/keys/private.pem")
	}
	if cfg.JWTPublicKey != "/keys/public.pem" {
		t.Errorf("JWTPublicKey: got %q, want %q", cfg.JWTPublicKey, "/keys/public.pem")
	}
	if cfg.JWTPrivateKeyB64 != "privbase64==" {
		t.Errorf("JWTPrivateKeyB64: got %q, want %q", cfg.JWTPrivateKeyB64, "privbase64==")
	}
	if cfg.JWTPublicKeyB64 != "pubbase64==" {
		t.Errorf("JWTPublicKeyB64: got %q, want %q", cfg.JWTPublicKeyB64, "pubbase64==")
	}
	if cfg.TelegramBotToken != "token123" {
		t.Errorf("TelegramBotToken: got %q, want %q", cfg.TelegramBotToken, "token123")
	}
	if cfg.TelegramChatID != "chat456" {
		t.Errorf("TelegramChatID: got %q, want %q", cfg.TelegramChatID, "chat456")
	}
}

func TestIsProduction(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"production", true},
		{"development", false},
		{"staging", false},
		{"prod", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.env, func(t *testing.T) {
			c := &Config{Env: tc.env}
			if got := c.IsProduction(); got != tc.want {
				t.Errorf("IsProduction() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEnvOr(t *testing.T) {
	tests := []struct {
		name    string
		envVal  string
		envSet  bool
		def     string
		want    string
	}{
		{"env set", "custom-value", true, "default", "custom-value"},
		{"env not set, use default", "", false, "default", "default"},
		{"env set to empty, use default", "", true, "default", "default"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key := "TEST_ENV_OR_KEY"
			if tc.envSet {
				os.Setenv(key, tc.envVal)
				defer os.Unsetenv(key)
			} else {
				os.Unsetenv(key)
			}
			got := envOr(key, tc.def)
			if got != tc.want {
				t.Errorf("envOr() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLoad_LogLevel(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want string
	}{
		{"debug level", "debug", "debug"},
		{"warn level", "warn", "warn"},
		{"error level", "error", "error"},
		{"empty defaults to info", "", "info"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(t)
			setEnv(t, "DATABASE_URL", "postgres://localhost/testdb")
			if tc.env != "" {
				setEnv(t, "LOG_LEVEL", tc.env)
			}
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.LogLevel != tc.want {
				t.Errorf("LogLevel: got %q, want %q", cfg.LogLevel, tc.want)
			}
		})
	}
}
