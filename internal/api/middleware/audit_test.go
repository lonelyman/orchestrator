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

type fakeAuditPort struct {
	saved chan auditSaveCall
}

type auditSaveCall struct {
	log            models.AuditLog
	contextErrSoon error
}

func (f *fakeAuditPort) Save(ctx context.Context, log models.AuditLog) error {
	f.saved <- auditSaveCall{
		log:            log,
		contextErrSoon: ctx.Err(),
	}
	return nil
}

func (f *fakeAuditPort) List(_ context.Context, _ models.AuditLogFilter) ([]models.AuditLog, error) {
	return nil, nil
}

func TestAuditMiddleware_SavesWithUsableBackgroundContext(t *testing.T) {
	auditPort := &fakeAuditPort{saved: make(chan auditSaveCall, 1)}
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("claims", &models.Claims{UserID: "u1", Username: "user1"})
		c.Locals("session_id", "s1")
		SetAuditQuery(c, "hello")
		SetAuditResponse(c, "world")
		SetAuditIntent(c, "direct")
		return c.Next()
	})
	app.Use(AuditMiddleware(auditPort))
	app.Post("/chat", func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusAccepted)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/chat", nil))
	if err != nil {
		t.Fatalf("app test: %v", err)
	}
	defer resp.Body.Close()

	select {
	case call := <-auditPort.saved:
		if call.contextErrSoon != nil {
			t.Fatalf("expected usable context, got %v", call.contextErrSoon)
		}
		if call.log.UserID != "u1" || call.log.Username != "user1" || call.log.SessionID != "s1" {
			t.Fatalf("unexpected audit identity: %+v", call.log)
		}
		if call.log.Query != "hello" || call.log.ResponsePreview != "world" || call.log.Intent != "direct" {
			t.Fatalf("unexpected audit content: %+v", call.log)
		}
		if call.log.StatusCode != http.StatusAccepted {
			t.Fatalf("expected status %d, got %d", http.StatusAccepted, call.log.StatusCode)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for audit save")
	}
}
