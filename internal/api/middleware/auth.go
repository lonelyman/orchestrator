package middleware

import (
	"log/slog"
	"strings"

	"github.com/enterprise-ai/orchestrator/internal/infrastructure/auth"
	"github.com/gofiber/fiber/v3"
)

// JWTMiddleware ตรวจสอบ JWT token ในทุก request
func JWTMiddleware(jwt *auth.JWTManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		// ดึง token จาก Header Authorization: Bearer <token>
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "authorization header required",
					"code":    "UNAUTHORIZED",
				},
			})
		}

		// ตรวจสอบ format "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(401).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "invalid authorization format",
					"code":    "UNAUTHORIZED",
				},
			})
		}

		// Verify JWT
		claims, err := jwt.Verify(parts[1])
		if err != nil {
			requestID, _ := c.Locals(RequestIDKey).(string)
			slog.Warn("invalid token",
				"request_id", requestID,
				"error", RedactLogValue(err.Error()),
			)
			return c.Status(401).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "invalid or expired token",
					"code":    "UNAUTHORIZED",
				},
			})
		}

		// เก็บ claims ไว้ใน context
		c.Locals("claims", claims)
		c.Locals("user_id", claims.UserID)
		c.Locals("username", claims.Username)

		slog.Debug("authenticated", "username", claims.Username, "role", claims.Role)

		return c.Next()
	}
}
