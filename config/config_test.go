package config

import (
	"strings"
	"testing"
	"time"
)

var configEnvKeys = []string{
	"DEV_MODE",
	"DEV_USERNAME",
	"DEV_PASSWORD",
	"API_PORT",
	"LLM_BACKEND",
	"LLM_HOST",
	"LLM_PORT",
	"LLM_MODEL",
	"EMBED_MODEL",
	"DB_HOST",
	"DB_PORT",
	"DB_NAME",
	"DB_USER",
	"DB_PASS",
	"AD_SERVER",
	"AD_PORT",
	"AD_BASE_DN",
	"AD_DOMAIN",
	"JWT_SECRET",
	"JWT_EXPIRY",
	"SYSTEM_PROMPT",
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range configEnvKeys {
		t.Setenv(key, "")
	}
}

func TestLoad_AllowsDevDefaults(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DEV_MODE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.DevMode {
		t.Fatalf("expected dev mode")
	}
	if cfg.JWTExpiry != 8*time.Hour {
		t.Fatalf("expected default JWT expiry 8h, got %s", cfg.JWTExpiry)
	}
}

func TestLoad_RejectsInvalidPort(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DEV_MODE", "true")
	t.Setenv("API_PORT", "not-a-port")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "API_PORT") {
		t.Fatalf("expected API_PORT error, got %v", err)
	}
}

func TestLoad_RejectsWeakProductionConfig(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DEV_MODE", "false")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error")
	}
	for _, want := range []string{"JWT_SECRET", "DB_PASS"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %s error, got %v", want, err)
		}
	}
}

func TestLoad_AcceptsProductionConfig(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DEV_MODE", "false")
	t.Setenv("DB_PASS", "correct-horse-battery-staple")
	t.Setenv("AD_SERVER", "ad.example.com")
	t.Setenv("AD_BASE_DN", "DC=example,DC=com")
	t.Setenv("AD_DOMAIN", "example.com")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("JWT_EXPIRY", "30m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.JWTExpiry != 30*time.Minute {
		t.Fatalf("expected JWT expiry 30m, got %s", cfg.JWTExpiry)
	}
}
