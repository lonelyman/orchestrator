package middleware

import (
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/gofiber/fiber/v3"
)

// RequireRole อนุญาตเฉพาะ user ที่มี role ตรงกับรายการที่กำหนด
func RequireRole(allowedRoles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		allowed[role] = struct{}{}
	}

	return func(c fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*models.Claims)
		if !ok || claims == nil {
			return c.Status(401).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "authentication required",
					"code":    "UNAUTHORIZED",
				},
			})
		}

		if _, ok := allowed[claims.Role]; ok {
			return c.Next()
		}

		return c.Status(403).JSON(fiber.Map{
			"error": fiber.Map{
				"message": "insufficient permissions",
				"code":    "FORBIDDEN",
			},
		})
	}
}
