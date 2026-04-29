package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/enterprise-ai/orchestrator/config"
	"github.com/gofiber/fiber/v3"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	app := fiber.New()
	app.Use(SecurityHeadersMiddleware(&config.AppConfig{
		CSPPolicy: "default-src 'none'",
	}))
	app.Get("/ok", func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ok", nil))
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected X-Frame-Options DENY, got %q", got)
	}
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected nosniff, got %q", got)
	}
	if got := resp.Header.Get("Content-Security-Policy"); got != "default-src 'none'" {
		t.Fatalf("unexpected CSP: %q", got)
	}
}

func TestCORSMiddleware_AllowsConfiguredOrigin(t *testing.T) {
	app := fiber.New()
	app.Use(CORSMiddleware(&config.AppConfig{
		AllowedOrigins: []string{"https://app.example.com"},
	}))
	app.Get("/ok", func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("Origin", "https://app.example.com")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("expected allowed origin header, got %q", got)
	}
}

func TestCORSMiddleware_DeniesByDefault(t *testing.T) {
	app := fiber.New()
	app.Use(CORSMiddleware(&config.AppConfig{}))
	app.Get("/ok", func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("Origin", "https://app.example.com")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allowed origin header, got %q", got)
	}
}
