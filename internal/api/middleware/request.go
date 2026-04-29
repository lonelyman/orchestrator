package middleware

import (
	"fmt"
	"log/slog"
	"strings"

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
