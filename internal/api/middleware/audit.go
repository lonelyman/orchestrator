package middleware

import (
	"log/slog"
	"strings"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/enterprise-ai/orchestrator/internal/domain/ports"
	"github.com/gofiber/fiber/v3"
)

// AuditMiddleware บันทึกทุก request อัตโนมัติ
func AuditMiddleware(auditPort ports.AuditPort, timeout ...time.Duration) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		// รัน handler ก่อน
		err := c.Next()

		// ดึงข้อมูลหลัง handler ทำงานเสร็จ
		claims, _ := c.Locals("claims").(*models.Claims)
		sessionID, _ := c.Locals("session_id").(string)

		userID := ""
		username := ""
		if claims != nil {
			userID = claims.UserID
			username = claims.Username
		}

		// ดึง intent จาก context (ถ้ามี)
		intent, _ := c.Locals("intent").(string)

		// ดึง query จาก request body (สำหรับ chat)
		query, _ := c.Locals("audit_query").(string)
		responsePreview, _ := c.Locals("audit_response").(string)

		// Truncate response preview ที่ rune boundary (Thai = 3 bytes/ตัว)
		if len([]rune(responsePreview)) > 200 {
			responsePreview = string([]rune(responsePreview)[:200]) + "..."
		}

		// สร้าง audit log
		log := models.AuditLog{
			RequestID:       requestID(c),
			UserID:          userID,
			Username:        username,
			SessionID:       sessionID,
			Method:          c.Method(),
			Path:            c.Path(),
			Intent:          intent,
			Query:           query,
			ResponsePreview: responsePreview,
			LatencyMs:       time.Since(start).Milliseconds(),
			StatusCode:      c.Response().StatusCode(),
			IPAddress:       c.IP(),
		}

		if saveErr := auditPort.Save(c.Context(), log); saveErr != nil {
			slog.Error("enqueue audit log failed",
				"error", saveErr,
				"request_id", log.RequestID,
				"path", log.Path,
			)
		}

		return err
	}
}

// SetAuditQuery helper สำหรับ handler ใส่ query ลงใน context
func SetAuditQuery(c fiber.Ctx, query string) {
	c.Locals("audit_query", query)
}

// SetAuditResponse helper สำหรับ handler ใส่ response ลงใน context
func SetAuditResponse(c fiber.Ctx, response string) {
	runes := []rune(response)
	if len(runes) > 200 {
		c.Locals("audit_response", string(runes[:200])+"...")
		return
	}
	c.Locals("audit_response", response)
}

// SetAuditIntent helper สำหรับ handler ใส่ intent ลงใน context
func SetAuditIntent(c fiber.Ctx, intent string) {
	// ลบ whitespace ออก
	c.Locals("intent", strings.TrimSpace(intent))
}

func requestID(c fiber.Ctx) string {
	requestID, _ := c.Locals(RequestIDKey).(string)
	return requestID
}
