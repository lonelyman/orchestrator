package config

import (
	"os"
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

	// System Prompt
	SystemPrompt string
}

// Load โหลด config จาก environment variables
func Load() *AppConfig {
	return &AppConfig{
		APIPort:    getEnv("API_PORT", "50000"),
		LLMBackend: getEnv("LLM_BACKEND", "ollama"),
		LLMHost:    getEnv("LLM_HOST", "localhost"),
		LLMPort:    getEnv("LLM_PORT", "11434"),
		LLMModel:   getEnv("LLM_MODEL", "qwen2.5:7b"),
		SystemPrompt: getEnv("SYSTEM_PROMPT",
			"You are a helpful enterprise AI assistant. You must always respond in Thai language only. Never use Chinese, English, or any other language. Thai language only."),
	}
}

// getEnv อ่านค่า env หรือใช้ค่า default
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
