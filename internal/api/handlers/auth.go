package handlers

import (
	"log/slog"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/auth"
	"github.com/gofiber/fiber/v3"
)

// AuthHandler จัดการ Authentication
type AuthHandler struct {
	ldap *auth.LDAPAdapter
	jwt  *auth.JWTManager
}

// NewAuthHandler สร้าง AuthHandler ใหม่
func NewAuthHandler(ldap *auth.LDAPAdapter, jwt *auth.JWTManager) *AuthHandler {
	return &AuthHandler{ldap: ldap, jwt: jwt}
}

// Login รับ username/password แล้วตรวจสอบกับ AD
func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.Bind().JSON(&req); err != nil {
		return Fail(c, 400, "invalid request", "BAD_REQUEST")
	}

	if req.Username == "" || req.Password == "" {
		return Fail(c, 400, "username and password required", "BAD_REQUEST")
	}

	// ตรวจสอบกับ AD
	user, err := h.ldap.Authenticate(req.Username, req.Password)
	if err != nil {
		slog.Warn("login failed", "username", req.Username, "error", err)
		return Fail(c, 401, "invalid credentials", "UNAUTHORIZED")
	}

	// สร้าง JWT
	token, expiresAt, err := h.jwt.Generate(user)
	if err != nil {
		slog.Error("generate token failed", "error", err)
		return Fail(c, 500, "token generation failed", "SERVER_ERROR")
	}

	slog.Info("login success", "username", user.Username, "department", user.Department)

	return OK(c, models.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      *user,
	})
}

// Me ดึงข้อมูล user จาก JWT
func (h *AuthHandler) Me(c fiber.Ctx) error {
	claims := c.Locals("claims").(*models.Claims)
	return OK(c, claims)
}