package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

type fakeHealthOrchestrator struct {
	healthErr   error
	embedderErr error
}

func (f fakeHealthOrchestrator) HealthCheck(context.Context) error {
	return f.healthErr
}

func (f fakeHealthOrchestrator) EmbedderCheck(context.Context) error {
	return f.embedderErr
}

type fakeHealthDB struct {
	err error
}

func (f fakeHealthDB) Ping(context.Context) error {
	return f.err
}

func TestHealthHandlerLive_DoesNotRequireDependencies(t *testing.T) {
	handler := NewHealthHandler(
		fakeHealthOrchestrator{healthErr: errors.New("llm down"), embedderErr: errors.New("embedder down")},
		fakeHealthDB{err: errors.New("db down")},
	)
	app := fiber.New()
	app.Get("/live", handler.Live)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/live", nil))
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestHealthHandlerReady_Healthy(t *testing.T) {
	handler := NewHealthHandler(fakeHealthOrchestrator{}, fakeHealthDB{})
	app := fiber.New()
	app.Get("/ready", handler.Ready)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ready", nil))
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestHealthHandlerReady_UnhealthyDependency(t *testing.T) {
	handler := NewHealthHandler(fakeHealthOrchestrator{}, fakeHealthDB{err: errors.New("db down")})
	app := fiber.New()
	app.Get("/ready", handler.Ready)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ready", nil))
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, resp.StatusCode)
	}
}
