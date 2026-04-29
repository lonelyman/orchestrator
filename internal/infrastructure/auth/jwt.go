package auth

import (
	"fmt"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/golang-jwt/jwt/v5"
)

// JWTManager จัดการการสร้างและตรวจสอบ JWT
type JWTManager struct {
	secretKey []byte
	expiry    time.Duration
}

// NewJWTManager สร้าง JWTManager ใหม่
func NewJWTManager(secretKey string, expiry time.Duration) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secretKey),
		expiry:    expiry,
	}
}

// jwtClaims คือ claims ที่เก็บใน JWT
type jwtClaims struct {
	models.Claims
	jwt.RegisteredClaims
}

// Generate สร้าง JWT token จาก user
func (j *JWTManager) Generate(user *models.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(j.expiry)

	claims := jwtClaims{
		Claims: models.Claims{
			UserID:      user.ID,
			Username:    user.Username,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Department:  user.Department,
			Role:        user.Role,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "enterprise-ai-orchestrator",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// Verify ตรวจสอบ JWT token และคืน claims
func (j *JWTManager) Verify(tokenString string) (*models.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return &claims.Claims, nil
}
