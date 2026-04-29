package middleware

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const (
	RequestIDHeader = "X-Request-ID"
	RequestIDKey    = "request_id"
)

func RequestIDMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		requestID := strings.TrimSpace(c.Get(RequestIDHeader))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Locals(RequestIDKey, requestID)
		c.Set(RequestIDHeader, requestID)

		return c.Next()
	}
}

func RequestLogMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		claims, _ := c.Locals("claims").(*models.Claims)
		userID := ""
		if claims != nil {
			userID = claims.UserID
		}
		sessionID, _ := c.Locals("session_id").(string)
		requestID, _ := c.Locals(RequestIDKey).(string)

		attrs := []any{
			"request_id", requestID,
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"latency_ms", time.Since(start).Milliseconds(),
			"ip", c.IP(),
			"user_id", userID,
			"session_id", sessionID,
		}
		if err != nil {
			attrs = append(attrs, "error", RedactLogValue(err.Error()))
			slog.Warn("request completed with error", attrs...)
			return err
		}

		status := c.Response().StatusCode()
		if status >= 500 {
			slog.Error("request completed", attrs...)
		} else if status >= 400 {
			slog.Warn("request completed", attrs...)
		} else {
			slog.Info("request completed", attrs...)
		}
		return nil
	}
}

func RecoveryMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		defer func() {
			if recovered := recover(); recovered != nil {
				requestID, _ := c.Locals(RequestIDKey).(string)
				slog.Error("request panic recovered",
					"request_id", requestID,
					"method", c.Method(),
					"path", c.Path(),
					"panic", fmt.Sprint(recovered),
				)

				_ = c.Status(500).JSON(fiber.Map{
					"error": fiber.Map{
						"message": "internal server error",
						"code":    "SERVER_ERROR",
					},
				})
			}
		}()

		return c.Next()
	}
}

func RedactLogValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}

	lower := strings.ToLower(value)
	sensitiveMarkers := []string{
		"authorization",
		"bearer ",
		"jwt",
		"token",
		"password",
		"secret",
	}
	for _, marker := range sensitiveMarkers {
		if strings.Contains(lower, marker) {
			return "[redacted]"
		}
	}
	return value
}
