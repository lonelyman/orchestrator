package config

import (
	"os"
	"strconv"
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
	JWTExpiry string

	// System Prompt
	SystemPrompt string

	DevMode     bool
	DevUsername string
	DevPassword string
}

// Load โหลด config จาก environment variables
func Load() *AppConfig {
	adPort, _ := strconv.Atoi(getEnv("AD_PORT", "389"))

	return &AppConfig{
		DevMode:     getEnv("DEV_MODE", "false") == "true",
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
		DBPass:      getEnv("DB_PASS", "changeme"),
		ADServer:    getEnv("AD_SERVER", "192.168.2.1"),
		ADPort:      adPort,
		ADBaseDN:    getEnv("AD_BASE_DN", "DC=nutritionprofess,DC=com"),
		ADDomain:    getEnv("AD_DOMAIN", "nutritionprofess.com"),
		JWTSecret:   getEnv("JWT_SECRET", "change-this-secret-in-production"),
		JWTExpiry:   getEnv("JWT_EXPIRY", "8h"),
		SystemPrompt: getEnv("SYSTEM_PROMPT",
			"You are a helpful enterprise AI assistant. You must always respond in Thai language only."),
	}
}

// getEnv อ่านค่า env หรือใช้ค่า default
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
