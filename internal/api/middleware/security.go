package middleware

import (
	"strings"

	"github.com/enterprise-ai/orchestrator/config"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
)

func SecurityHeadersMiddleware(cfg *config.AppConfig) fiber.Handler {
	return helmet.New(helmet.Config{
		XFrameOptions:         "DENY",
		ContentSecurityPolicy: strings.TrimSpace(cfg.CSPPolicy),
		ReferrerPolicy:        "no-referrer",
		PermissionPolicy:      "camera=(), microphone=(), geolocation=()",
		HSTSMaxAge:            31536000,
		HSTSPreloadEnabled:    true,
	})
}

func CORSMiddleware(cfg *config.AppConfig) fiber.Handler {
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	allowAll := false
	for _, origin := range cfg.AllowedOrigins {
		origin = strings.TrimSpace(strings.ToLower(origin))
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAll = true
			continue
		}
		allowed[origin] = struct{}{}
	}

	return cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			if allowAll {
				return true
			}
			_, ok := allowed[strings.ToLower(strings.TrimSpace(origin))]
			return ok
		},
		AllowMethods: []string{
			fiber.MethodGet,
			fiber.MethodPost,
			fiber.MethodDelete,
			fiber.MethodOptions,
		},
		AllowHeaders: []string{
			fiber.HeaderAuthorization,
			fiber.HeaderContentType,
			RequestIDHeader,
			"X-Session-ID",
		},
		ExposeHeaders: []string{
			RequestIDHeader,
			"X-Session-ID",
		},
		AllowCredentials: cfg.CORSAllowCredentials,
		MaxAge:           600,
	})
}
