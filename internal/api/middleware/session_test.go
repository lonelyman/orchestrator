package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/gofiber/fiber/v3"
)

type fakeSessionPort struct {
	sessions map[string]*models.Session
	created  []*models.Session
}

func (f *fakeSessionPort) CreateSession(_ context.Context, userID string) (*models.Session, error) {
	session := &models.Session{
		ID:        "new-session",
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		IsActive:  true,
	}
	f.created = append(f.created, session)
	return session, nil
}

func (f *fakeSessionPort) GetSession(_ context.Context, sessionID string) (*models.Session, error) {
	return f.sessions[sessionID], nil
}

func (f *fakeSessionPort) GetHistory(_ context.Context, _ string, _ int) ([]models.Message, error) {
	return nil, nil
}

func (f *fakeSessionPort) SaveMessage(_ context.Context, _ models.Message) error {
	return nil
}

func (f *fakeSessionPort) ExpireSession(_ context.Context, _ string) error {
	return nil
}

func TestSessionMiddleware_ReusesSessionOwnedByUser(t *testing.T) {
	sessionPort := &fakeSessionPort{
		sessions: map[string]*models.Session{
			"existing-session": {ID: "existing-session", UserID: "user-1", IsActive: true},
		},
	}
	app := newSessionMiddlewareTestApp(sessionPort, "user-1")

	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	req.Header.Set("X-Session-ID", "existing-session")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("X-Session-ID"); got != "existing-session" {
		t.Fatalf("expected existing session, got %q", got)
	}
	if len(sessionPort.created) != 0 {
		t.Fatalf("expected no new session, got %d", len(sessionPort.created))
	}
}

func TestSessionMiddleware_ReplacesSessionOwnedByAnotherUser(t *testing.T) {
	sessionPort := &fakeSessionPort{
		sessions: map[string]*models.Session{
			"other-user-session": {ID: "other-user-session", UserID: "user-2", IsActive: true},
		},
	}
	app := newSessionMiddlewareTestApp(sessionPort, "user-1")

	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	req.Header.Set("X-Session-ID", "other-user-session")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("X-Session-ID"); got != "new-session" {
		t.Fatalf("expected replacement session, got %q", got)
	}
	if len(sessionPort.created) != 1 {
		t.Fatalf("expected one new session, got %d", len(sessionPort.created))
	}
	if sessionPort.created[0].UserID != "user-1" {
		t.Fatalf("expected new session for user-1, got %q", sessionPort.created[0].UserID)
	}
}

func newSessionMiddlewareTestApp(sessionPort *fakeSessionPort, userID string) *fiber.App {
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("claims", &models.Claims{UserID: userID, Username: userID})
		return c.Next()
	})
	app.Use(SessionMiddleware(sessionPort))
	app.Get("/chat", func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})
	return app
}
