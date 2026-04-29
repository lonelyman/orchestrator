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
	"RATE_LIMIT_PER_MINUTE",
	"SESSION_EXPIRY",
	"CHAT_TIMEOUT",
	"RAG_INGEST_TIMEOUT",
	"AUDIT_TIMEOUT",
	"HEALTH_TIMEOUT",
	"MIGRATION_TIMEOUT",
	"MAX_UPLOAD_MB",
	"LLM_BACKEND",
	"LLM_HOST",
	"LLM_PORT",
	"LLM_MODEL",
	"LLM_API_KEY",
	"EMBED_HOST",
	"EMBED_PORT",
	"EMBED_MODEL",
	"OCR_ENGINE",
	"OCR_HOST",
	"OCR_PORT",
	"OCR_MODEL",
	"OCR_TIMEOUT",
	"OCR_MAX_PAGES",
	"OCR_PROMPT",
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
	if cfg.RateLimitPerMin != 20 {
		t.Fatalf("expected default rate limit 20, got %d", cfg.RateLimitPerMin)
	}
	if cfg.SessionExpiry != 30*time.Minute {
		t.Fatalf("expected default session expiry 30m, got %s", cfg.SessionExpiry)
	}
	if cfg.ChatTimeout != 120*time.Second {
		t.Fatalf("expected default chat timeout 120s, got %s", cfg.ChatTimeout)
	}
	if cfg.MaxUploadMegabyte != 10 || cfg.MaxUploadBytes != 10<<20 {
		t.Fatalf("expected default upload 10MB, got %d MB / %d bytes", cfg.MaxUploadMegabyte, cfg.MaxUploadBytes)
	}
	if cfg.EmbedHost != cfg.LLMHost || cfg.EmbedPort != cfg.LLMPort {
		t.Fatalf("expected embed endpoint to default to llm endpoint, got %s:%s vs %s:%s", cfg.EmbedHost, cfg.EmbedPort, cfg.LLMHost, cfg.LLMPort)
	}
	if cfg.OCREngine != "tesseract" {
		t.Fatalf("expected default OCR engine tesseract, got %q", cfg.OCREngine)
	}
	if cfg.OCRTimeout != 180*time.Second {
		t.Fatalf("expected default OCR timeout 180s, got %s", cfg.OCRTimeout)
	}
	if cfg.OCRMaxPages != 20 {
		t.Fatalf("expected default OCR max pages 20, got %d", cfg.OCRMaxPages)
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

func TestLoad_RejectsInvalidOperationalLimit(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DEV_MODE", "true")
	t.Setenv("RATE_LIMIT_PER_MINUTE", "0")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "RATE_LIMIT_PER_MINUTE") {
		t.Fatalf("expected RATE_LIMIT_PER_MINUTE error, got %v", err)
	}
}

func TestLoad_RejectsUnsupportedLLMBackend(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DEV_MODE", "true")
	t.Setenv("LLM_BACKEND", "unsupported")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "LLM_BACKEND") {
		t.Fatalf("expected LLM_BACKEND error, got %v", err)
	}
}

func TestLoad_RejectsUnsupportedOCREngine(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DEV_MODE", "true")
	t.Setenv("OCR_ENGINE", "unsupported")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "OCR_ENGINE") {
		t.Fatalf("expected OCR_ENGINE error, got %v", err)
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
	t.Setenv("SESSION_EXPIRY", "45m")
	t.Setenv("CHAT_TIMEOUT", "90s")
	t.Setenv("MAX_UPLOAD_MB", "25")
	t.Setenv("LLM_BACKEND", "vllm")
	t.Setenv("LLM_API_KEY", "test-key")
	t.Setenv("EMBED_HOST", "embed.example.com")
	t.Setenv("EMBED_PORT", "11435")
	t.Setenv("OCR_ENGINE", "ollama")
	t.Setenv("OCR_HOST", "ocr.example.com")
	t.Setenv("OCR_PORT", "11436")
	t.Setenv("OCR_MODEL", "scb10x/typhoon-ocr1.5-3b:latest")
	t.Setenv("OCR_TIMEOUT", "240s")
	t.Setenv("OCR_MAX_PAGES", "30")
	t.Setenv("OCR_PROMPT", "OCR only")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.JWTExpiry != 30*time.Minute {
		t.Fatalf("expected JWT expiry 30m, got %s", cfg.JWTExpiry)
	}
	if cfg.SessionExpiry != 45*time.Minute {
		t.Fatalf("expected session expiry 45m, got %s", cfg.SessionExpiry)
	}
	if cfg.ChatTimeout != 90*time.Second {
		t.Fatalf("expected chat timeout 90s, got %s", cfg.ChatTimeout)
	}
	if cfg.MaxUploadBytes != 25<<20 {
		t.Fatalf("expected max upload 25MB, got %d", cfg.MaxUploadBytes)
	}
	if cfg.LLMBackend != "vllm" || cfg.LLMAPIKey != "test-key" {
		t.Fatalf("unexpected llm config: backend=%q key=%q", cfg.LLMBackend, cfg.LLMAPIKey)
	}
	if cfg.EmbedHost != "embed.example.com" || cfg.EmbedPort != "11435" {
		t.Fatalf("unexpected embed endpoint: %s:%s", cfg.EmbedHost, cfg.EmbedPort)
	}
	if cfg.OCREngine != "ollama" || cfg.OCRHost != "ocr.example.com" || cfg.OCRPort != "11436" {
		t.Fatalf("unexpected OCR endpoint: engine=%q endpoint=%s:%s", cfg.OCREngine, cfg.OCRHost, cfg.OCRPort)
	}
	if cfg.OCRModel != "scb10x/typhoon-ocr1.5-3b:latest" || cfg.OCRPrompt != "OCR only" {
		t.Fatalf("unexpected OCR model/prompt: model=%q prompt=%q", cfg.OCRModel, cfg.OCRPrompt)
	}
	if cfg.OCRTimeout != 240*time.Second || cfg.OCRMaxPages != 30 {
		t.Fatalf("unexpected OCR limits: timeout=%s maxPages=%d", cfg.OCRTimeout, cfg.OCRMaxPages)
	}
}
