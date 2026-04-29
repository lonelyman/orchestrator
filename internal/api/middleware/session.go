package middleware

import (
	"log/slog"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/enterprise-ai/orchestrator/internal/domain/ports"
	"github.com/gofiber/fiber/v3"
)

// SessionMiddleware จัดการ Session อัตโนมัติ
func SessionMiddleware(sessionPort ports.SessionPort) fiber.Handler {
	return func(c fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*models.Claims)
		if !ok || claims == nil {
			return c.Next()
		}

		ctx := c.Context()
		sessionID := c.Get("X-Session-ID")
		requestID, _ := c.Locals(RequestIDKey).(string)

		var session *models.Session
		var err error

		// ถ้ามี Session ID → ดึง Session เดิม
		if sessionID != "" {
			session, err = sessionPort.GetSession(ctx, sessionID)
			if err != nil {
				slog.Warn("session not found, creating new", "request_id", requestID, "session_id", sessionID)
				session = nil
			} else if session.UserID != claims.UserID {
				slog.Warn("session owner mismatch, creating new",
					"request_id", requestID,
					"session_id", sessionID,
					"session_user_id", session.UserID,
					"claims_user_id", claims.UserID,
				)
				session = nil
			}
		}

		// ถ้าไม่มี Session หรือ Session หมดอายุ → สร้างใหม่
		if session == nil {
			session, err = sessionPort.CreateSession(ctx, claims.UserID)
			if err != nil {
				slog.Error("create session failed", "request_id", requestID, "error", err)
				return c.Next()
			}
			slog.Info("session created", "request_id", requestID, "session_id", session.ID, "user_id", claims.UserID)
		}

		// เก็บ Session ไว้ใน context
		c.Locals("session", session)
		c.Locals("session_id", session.ID)

		// ส่ง Session ID กลับใน response header
		c.Set("X-Session-ID", session.ID)

		return c.Next()
	}
}
