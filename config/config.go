package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultJWTSecret = "change-this-secret-in-production"
	exampleJWTSecret = "change-this-to-random-string-in-production"
	defaultDBPass    = "changeme"
)

// AppConfig เก็บ configuration ทั้งหมดของระบบ
type AppConfig struct {
	// Server
	APIPort string

	// LLM
	LLMBackend string
	LLMHost    string
	LLMPort    string
	LLMModel   string

	// Embedding
	EmbedModel string

	// Database
	DBHost string
	DBPort string
	DBName string
	DBUser string
	DBPass string

	// Active Directory
	ADServer string
	ADPort   int
	ADBaseDN string
	ADDomain string

	// JWT
	JWTSecret string
	JWTExpiry time.Duration

	// System Prompt
	SystemPrompt string

	DevMode     bool
	DevUsername string
	DevPassword string
}

// Load โหลด config จาก environment variables และตรวจค่าที่เสี่ยงก่อนเริ่มระบบ
func Load() (*AppConfig, error) {
	devMode, err := parseBool("DEV_MODE", "false")
	if err != nil {
		return nil, err
	}

	adPort, err := parsePort("AD_PORT", "389")
	if err != nil {
		return nil, err
	}
	if _, err := parsePort("API_PORT", "50000"); err != nil {
		return nil, err
	}
	if _, err := parsePort("LLM_PORT", "11434"); err != nil {
		return nil, err
	}
	if _, err := parsePort("DB_PORT", "5432"); err != nil {
		return nil, err
	}

	jwtExpiry, err := time.ParseDuration(getEnv("JWT_EXPIRY", "8h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY: %w", err)
	}

	cfg := &AppConfig{
		DevMode:     devMode,
		DevUsername: getEnv("DEV_USERNAME", "dev"),
		DevPassword: getEnv("DEV_PASSWORD", "dev1234"),
		APIPort:     getEnv("API_PORT", "50000"),
		LLMBackend:  getEnv("LLM_BACKEND", "ollama"),
		LLMHost:     getEnv("LLM_HOST", "localhost"),
		LLMPort:     getEnv("LLM_PORT", "11434"),
		LLMModel:    getEnv("LLM_MODEL", "qwen2.5:7b"),
		EmbedModel:  getEnv("EMBED_MODEL", "nomic-embed-text"),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBName:      getEnv("DB_NAME", "orchestrator"),
		DBUser:      getEnv("DB_USER", "orchestrator"),
		DBPass:      getEnv("DB_PASS", defaultDBPass),
		ADServer:    getEnv("AD_SERVER", "192.168.2.1"),
		ADPort:      adPort,
		ADBaseDN:    getEnv("AD_BASE_DN", "DC=nutritionprofess,DC=com"),
		ADDomain:    getEnv("AD_DOMAIN", "nutritionprofess.com"),
		JWTSecret:   getEnv("JWT_SECRET", defaultJWTSecret),
		JWTExpiry:   jwtExpiry,
		SystemPrompt: getEnv("SYSTEM_PROMPT",
			"You are a helpful enterprise AI assistant. You must always respond in Thai language only."),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate ตรวจ config ที่ทำให้ runtime เสี่ยงหรือทำงานผิดทันที
func (c *AppConfig) Validate() error {
	var errs []error

	required := map[string]string{
		"API_PORT":      c.APIPort,
		"LLM_HOST":      c.LLMHost,
		"LLM_PORT":      c.LLMPort,
		"LLM_MODEL":     c.LLMModel,
		"EMBED_MODEL":   c.EmbedModel,
		"DB_HOST":       c.DBHost,
		"DB_PORT":       c.DBPort,
		"DB_NAME":       c.DBName,
		"DB_USER":       c.DBUser,
		"DB_PASS":       c.DBPass,
		"JWT_SECRET":    c.JWTSecret,
		"SYSTEM_PROMPT": c.SystemPrompt,
	}
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			errs = append(errs, fmt.Errorf("%s is required", key))
		}
	}

	if c.JWTExpiry <= 0 {
		errs = append(errs, fmt.Errorf("JWT_EXPIRY must be greater than zero"))
	}

	if !c.DevMode {
		if isWeakJWTSecret(c.JWTSecret) {
			errs = append(errs, fmt.Errorf("JWT_SECRET must be replaced with a strong random value when DEV_MODE=false"))
		}
		if c.DBPass == defaultDBPass {
			errs = append(errs, fmt.Errorf("DB_PASS must be changed when DEV_MODE=false"))
		}
		if strings.Contains(c.ADServer, "x.x") || strings.Contains(c.ADBaseDN, "your-domain") || strings.Contains(c.ADDomain, "your-domain") {
			errs = append(errs, fmt.Errorf("AD_* placeholder values must be changed when DEV_MODE=false"))
		}
	}

	return errors.Join(errs...)
}

// getEnv อ่านค่า env หรือใช้ค่า default
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func parseBool(key, defaultVal string) (bool, error) {
	value := getEnv(key, defaultVal)
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid %s: %w", key, err)
	}
	return parsed, nil
}

func parsePort(key, defaultVal string) (int, error) {
	value := getEnv(key, defaultVal)
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid %s: must be between 1 and 65535", key)
	}
	return port, nil
}

func isWeakJWTSecret(secret string) bool {
	secret = strings.TrimSpace(secret)
	if len(secret) < 32 {
		return true
	}
	if secret == defaultJWTSecret || secret == exampleJWTSecret {
		return true
	}
	return strings.Contains(strings.ToLower(secret), "change-this")
}
