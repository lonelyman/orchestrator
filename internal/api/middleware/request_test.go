package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestRequestIDMiddleware_GeneratesRequestID(t *testing.T) {
	app := fiber.New()
	app.Use(RequestIDMiddleware())
	app.Get("/ok", func(c fiber.Ctx) error {
		if requestID, _ := c.Locals(RequestIDKey).(string); requestID == "" {
			t.Fatalf("expected request id in locals")
		}
		return c.SendStatus(http.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ok", nil))
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get(RequestIDHeader); got == "" {
		t.Fatalf("expected %s response header", RequestIDHeader)
	}
}

func TestRequestIDMiddleware_PreservesIncomingRequestID(t *testing.T) {
	app := fiber.New()
	app.Use(RequestIDMiddleware())
	app.Get("/ok", func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set(RequestIDHeader, "req-123")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get(RequestIDHeader); got != "req-123" {
		t.Fatalf("expected incoming request id, got %q", got)
	}
}

func TestRecoveryMiddleware_ReturnsServerError(t *testing.T) {
	app := fiber.New()
	app.Use(RequestIDMiddleware())
	app.Use(RecoveryMiddleware())
	app.Get("/panic", func(c fiber.Ctx) error {
		panic("boom")
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/panic", nil))
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, resp.StatusCode)
	}
	if got := resp.Header.Get(RequestIDHeader); got == "" {
		t.Fatalf("expected %s response header", RequestIDHeader)
	}
}
