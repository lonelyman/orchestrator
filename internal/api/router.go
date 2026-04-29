package api

import (
	"time"

	"github.com/enterprise-ai/orchestrator/config"
	"github.com/enterprise-ai/orchestrator/internal/api/handlers"
	"github.com/enterprise-ai/orchestrator/internal/api/middleware"
	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/enterprise-ai/orchestrator/internal/domain/ports"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/auth"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

type RouterDependencies struct {
	Config          *config.AppConfig
	JWTManager      *auth.JWTManager
	SessionStore    ports.SessionPort
	AuditStore      ports.AuditPort
	AuthHandler     *handlers.AuthHandler
	HealthHandler   *handlers.HealthHandler
	ChatHandler     *handlers.ChatHandler
	RAGHandler      *handlers.RAGHandler
	DocumentHandler *handlers.DocumentHandler
	AuditHandler    *handlers.AuditHandler
}

func NewRouter(deps RouterDependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:   "Enterprise AI Orchestrator v0.1",
		BodyLimit: deps.Config.BodyLimit,
	})

	app.Use(middleware.RequestIDMiddleware())
	app.Use(middleware.RecoveryMiddleware())
	app.Use(middleware.SecurityHeadersMiddleware(deps.Config))
	app.Use(middleware.CORSMiddleware(deps.Config))
	app.Use(middleware.RequestLogMiddleware())

	RegisterRoutes(app, deps)
	return app
}

func RegisterRoutes(app *fiber.App, deps RouterDependencies) {
	app.Post("/auth/login", deps.AuthHandler.Login)
	app.Get("/live", deps.HealthHandler.Live)
	app.Get("/ready", deps.HealthHandler.Ready)
	app.Get("/health", deps.HealthHandler.Check)
	app.Get("/v1/models", deps.ChatHandler.Models)

	protected := app.Group("/", middleware.JWTMiddleware(deps.JWTManager))
	protected.Use(middleware.SessionMiddleware(deps.SessionStore))
	protected.Use(middleware.AuditMiddleware(deps.AuditStore, deps.Config.AuditTimeout))
	protected.Use(rateLimiter(deps.Config.RateLimitPerMin))

	protected.Get("/auth/me", deps.AuthHandler.Me)
	protected.Post("/v1/chat/completions", deps.ChatHandler.Completions)
	protected.Post("/v1/rag/ingest", deps.RAGHandler.Ingest)
	protected.Post("/v1/rag/upload", deps.RAGHandler.Upload)
	protected.Get("/v1/rag/documents", deps.DocumentHandler.List)
	protected.Delete("/v1/rag/documents/:source", deps.DocumentHandler.Delete)
	protected.Delete("/v1/rag/documents", deps.DocumentHandler.Delete)

	admin := app.Group("/v1/admin", middleware.JWTMiddleware(deps.JWTManager))
	admin.Use(middleware.RequireRole("admin"))
	admin.Get("/logs", deps.AuditHandler.List)
}

func rateLimiter(maxRequests int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        maxRequests,
		Expiration: time.Minute,
		KeyGenerator: func(c fiber.Ctx) string {
			if claims, ok := c.Locals("claims").(*models.Claims); ok && claims != nil {
				return claims.UserID
			}
			return c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "too many requests, please slow down",
					"code":    "RATE_LIMIT_EXCEEDED",
				},
			})
		},
	})
}
