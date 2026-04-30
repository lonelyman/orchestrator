package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultJWTSecret    = "change-this-secret-in-production"
	exampleJWTSecret    = "change-this-to-random-string-in-production"
	defaultDBPass       = "changeme"
	defaultSystemPrompt = "You are a helpful enterprise AI assistant. You must always respond in Thai language only."
)

// AppConfig เก็บ configuration ทั้งหมดของระบบ
type AppConfig struct {
	// Server
	APIPort              string
	BodyLimit            int
	HealthTimeout        time.Duration
	ShutdownTimeout      time.Duration
	AllowedOrigins       []string
	CSPPolicy            string
	CORSAllowCredentials bool

	// LLM
	LLMBackend string
	LLMHost    string
	LLMPort    string
	LLMModel   string
	LLMAPIKey  string

	// Embedding
	EmbedModel string
	EmbedHost  string
	EmbedPort  string

	// Web Search
	WebSearchEnabled    bool
	WebSearchProvider   string
	WebSearchAPIKey     string
	WebSearchBaseURL    string
	WebSearchTimeout    time.Duration
	WebSearchMaxResults int

	// Database
	DBHost                  string
	DBPort                  string
	DBName                  string
	DBUser                  string
	DBPass                  string
	DBPoolMaxConns          int
	DBPoolMinConns          int
	DBPoolMaxConnLifetime   time.Duration
	DBPoolMaxConnIdleTime   time.Duration
	DBPoolHealthCheckPeriod time.Duration

	// Active Directory
	ADServer string
	ADPort   int
	ADBaseDN string
	ADDomain string

	// JWT
	JWTSecret string
	JWTExpiry time.Duration

	// Runtime Limits
	SessionExpiry     time.Duration
	ChatTimeout       time.Duration
	RAGIngestTimeout  time.Duration
	AuditTimeout      time.Duration
	AuditWorkers      int
	AuditQueueSize    int
	MigrationTimeout  time.Duration
	RateLimitPerMin   int
	MaxUploadBytes    int64
	MaxUploadMegabyte int
	OCREngine         string
	OCRHost           string
	OCRPort           string
	OCRModel          string
	OCRPrompt         string
	OCRTimeout        time.Duration
	OCRMaxPages       int

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
	if _, err := parsePort("EMBED_PORT", getEnv("LLM_PORT", "11434")); err != nil {
		return nil, err
	}
	if _, err := parsePort("OCR_PORT", getEnv("LLM_PORT", "11434")); err != nil {
		return nil, err
	}
	if _, err := parsePort("DB_PORT", "5432"); err != nil {
		return nil, err
	}

	jwtExpiry, err := time.ParseDuration(getEnv("JWT_EXPIRY", "8h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY: %w", err)
	}
	sessionExpiry, err := parseDuration("SESSION_EXPIRY", "30m")
	if err != nil {
		return nil, err
	}
	chatTimeout, err := parseDuration("CHAT_TIMEOUT", "120s")
	if err != nil {
		return nil, err
	}
	ragIngestTimeout, err := parseDuration("RAG_INGEST_TIMEOUT", "120s")
	if err != nil {
		return nil, err
	}
	auditTimeout, err := parseDuration("AUDIT_TIMEOUT", "5s")
	if err != nil {
		return nil, err
	}
	auditWorkers, err := parsePositiveInt("AUDIT_WORKERS", "4")
	if err != nil {
		return nil, err
	}
	auditQueueSize, err := parsePositiveInt("AUDIT_QUEUE_SIZE", "1000")
	if err != nil {
		return nil, err
	}
	healthTimeout, err := parseDuration("HEALTH_TIMEOUT", "5s")
	if err != nil {
		return nil, err
	}
	shutdownTimeout, err := parseDuration("SHUTDOWN_TIMEOUT", "10s")
	if err != nil {
		return nil, err
	}
	migrationTimeout, err := parseDuration("MIGRATION_TIMEOUT", "30s")
	if err != nil {
		return nil, err
	}
	dbPoolMaxConns, err := parsePositiveInt("DB_POOL_MAX_CONNS", "20")
	if err != nil {
		return nil, err
	}
	dbPoolMinConns, err := parsePositiveInt("DB_POOL_MIN_CONNS", "2")
	if err != nil {
		return nil, err
	}
	dbPoolMaxConnLifetime, err := parseDuration("DB_POOL_MAX_CONN_LIFETIME", "30m")
	if err != nil {
		return nil, err
	}
	dbPoolMaxConnIdleTime, err := parseDuration("DB_POOL_MAX_CONN_IDLE_TIME", "5m")
	if err != nil {
		return nil, err
	}
	dbPoolHealthCheckPeriod, err := parseDuration("DB_POOL_HEALTH_CHECK_PERIOD", "1m")
	if err != nil {
		return nil, err
	}
	rateLimitPerMin, err := parsePositiveInt("RATE_LIMIT_PER_MINUTE", "20")
	if err != nil {
		return nil, err
	}
	maxUploadMB, err := parsePositiveInt("MAX_UPLOAD_MB", "10")
	if err != nil {
		return nil, err
	}
	maxUploadBytes := int64(maxUploadMB) << 20
	ocrTimeout, err := parseDuration("OCR_TIMEOUT", "180s")
	if err != nil {
		return nil, err
	}
	ocrMaxPages, err := parsePositiveInt("OCR_MAX_PAGES", "20")
	if err != nil {
		return nil, err
	}
	webSearchEnabled, err := parseBool("WEB_SEARCH_ENABLED", "false")
	if err != nil {
		return nil, err
	}
	webSearchTimeout, err := parseDuration("WEB_SEARCH_TIMEOUT", "10s")
	if err != nil {
		return nil, err
	}
	webSearchMaxResults, err := parsePositiveInt("WEB_SEARCH_MAX_RESULTS", "5")
	if err != nil {
		return nil, err
	}
	systemPrompt, err := loadSystemPrompt()
	if err != nil {
		return nil, err
	}
	allowedOrigins, err := parseAllowedOrigins("ALLOWED_ORIGINS")
	if err != nil {
		return nil, err
	}
	corsAllowCredentials, err := parseBool("CORS_ALLOW_CREDENTIALS", "false")
	if err != nil {
		return nil, err
	}

	cfg := &AppConfig{
		DevMode:                 devMode,
		DevUsername:             getEnv("DEV_USERNAME", "dev"),
		DevPassword:             getEnv("DEV_PASSWORD", "dev1234"),
		APIPort:                 getEnv("API_PORT", "50000"),
		BodyLimit:               int(maxUploadBytes + (1 << 20)),
		AllowedOrigins:          allowedOrigins,
		CSPPolicy:               getEnv("CSP_POLICY", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"),
		CORSAllowCredentials:    corsAllowCredentials,
		LLMBackend:              getEnv("LLM_BACKEND", "ollama"),
		LLMHost:                 getEnv("LLM_HOST", "localhost"),
		LLMPort:                 getEnv("LLM_PORT", "11434"),
		LLMModel:                getEnv("LLM_MODEL", "qwen2.5:7b"),
		LLMAPIKey:               getEnv("LLM_API_KEY", ""),
		EmbedModel:              getEnv("EMBED_MODEL", "nomic-embed-text"),
		EmbedHost:               getEnv("EMBED_HOST", getEnv("LLM_HOST", "localhost")),
		EmbedPort:               getEnv("EMBED_PORT", getEnv("LLM_PORT", "11434")),
		WebSearchEnabled:        webSearchEnabled,
		WebSearchProvider:       getEnv("WEB_SEARCH_PROVIDER", "tavily"),
		WebSearchAPIKey:         getEnv("WEB_SEARCH_API_KEY", ""),
		WebSearchBaseURL:        strings.TrimRight(getEnv("WEB_SEARCH_BASE_URL", "https://api.tavily.com"), "/"),
		WebSearchTimeout:        webSearchTimeout,
		WebSearchMaxResults:     webSearchMaxResults,
		DBHost:                  getEnv("DB_HOST", "localhost"),
		DBPort:                  getEnv("DB_PORT", "5432"),
		DBName:                  getEnv("DB_NAME", "orchestrator"),
		DBUser:                  getEnv("DB_USER", "orchestrator"),
		DBPass:                  getEnv("DB_PASS", defaultDBPass),
		DBPoolMaxConns:          dbPoolMaxConns,
		DBPoolMinConns:          dbPoolMinConns,
		DBPoolMaxConnLifetime:   dbPoolMaxConnLifetime,
		DBPoolMaxConnIdleTime:   dbPoolMaxConnIdleTime,
		DBPoolHealthCheckPeriod: dbPoolHealthCheckPeriod,
		ADServer:                getEnv("AD_SERVER", "192.168.2.1"),
		ADPort:                  adPort,
		ADBaseDN:                getEnv("AD_BASE_DN", "DC=nutritionprofess,DC=com"),
		ADDomain:                getEnv("AD_DOMAIN", "nutritionprofess.com"),
		JWTSecret:               getEnv("JWT_SECRET", defaultJWTSecret),
		JWTExpiry:               jwtExpiry,
		// Runtime Limits
		SessionExpiry:     sessionExpiry,
		ChatTimeout:       chatTimeout,
		RAGIngestTimeout:  ragIngestTimeout,
		AuditTimeout:      auditTimeout,
		AuditWorkers:      auditWorkers,
		AuditQueueSize:    auditQueueSize,
		HealthTimeout:     healthTimeout,
		ShutdownTimeout:   shutdownTimeout,
		MigrationTimeout:  migrationTimeout,
		RateLimitPerMin:   rateLimitPerMin,
		MaxUploadBytes:    maxUploadBytes,
		MaxUploadMegabyte: maxUploadMB,
		OCREngine:         getEnv("OCR_ENGINE", "tesseract"),
		OCRHost:           getEnv("OCR_HOST", getEnv("LLM_HOST", "localhost")),
		OCRPort:           getEnv("OCR_PORT", getEnv("LLM_PORT", "11434")),
		OCRModel:          getEnv("OCR_MODEL", "scb10x/typhoon-ocr1.5-3b:latest"),
		OCRPrompt:         getEnv("OCR_PROMPT", "Extract all readable text from this image. Preserve Thai and English text. Return only the extracted text."),
		OCRTimeout:        ocrTimeout,
		OCRMaxPages:       ocrMaxPages,
		SystemPrompt:      systemPrompt,
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
		"LLM_BACKEND":   c.LLMBackend,
		"EMBED_HOST":    c.EmbedHost,
		"EMBED_PORT":    c.EmbedPort,
		"EMBED_MODEL":   c.EmbedModel,
		"OCR_ENGINE":    c.OCREngine,
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
	if c.SessionExpiry <= 0 {
		errs = append(errs, fmt.Errorf("SESSION_EXPIRY must be greater than zero"))
	}
	if c.ChatTimeout <= 0 {
		errs = append(errs, fmt.Errorf("CHAT_TIMEOUT must be greater than zero"))
	}
	if c.RAGIngestTimeout <= 0 {
		errs = append(errs, fmt.Errorf("RAG_INGEST_TIMEOUT must be greater than zero"))
	}
	if c.AuditTimeout <= 0 {
		errs = append(errs, fmt.Errorf("AUDIT_TIMEOUT must be greater than zero"))
	}
	if c.AuditWorkers <= 0 {
		errs = append(errs, fmt.Errorf("AUDIT_WORKERS must be greater than zero"))
	}
	if c.AuditQueueSize <= 0 {
		errs = append(errs, fmt.Errorf("AUDIT_QUEUE_SIZE must be greater than zero"))
	}
	if c.HealthTimeout <= 0 {
		errs = append(errs, fmt.Errorf("HEALTH_TIMEOUT must be greater than zero"))
	}
	if c.ShutdownTimeout <= 0 {
		errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT must be greater than zero"))
	}
	if c.MigrationTimeout <= 0 {
		errs = append(errs, fmt.Errorf("MIGRATION_TIMEOUT must be greater than zero"))
	}
	if c.DBPoolMaxConns <= 0 {
		errs = append(errs, fmt.Errorf("DB_POOL_MAX_CONNS must be greater than zero"))
	}
	if c.DBPoolMinConns <= 0 {
		errs = append(errs, fmt.Errorf("DB_POOL_MIN_CONNS must be greater than zero"))
	}
	if c.DBPoolMinConns > c.DBPoolMaxConns {
		errs = append(errs, fmt.Errorf("DB_POOL_MIN_CONNS must be less than or equal to DB_POOL_MAX_CONNS"))
	}
	if c.DBPoolMaxConnLifetime <= 0 {
		errs = append(errs, fmt.Errorf("DB_POOL_MAX_CONN_LIFETIME must be greater than zero"))
	}
	if c.DBPoolMaxConnIdleTime <= 0 {
		errs = append(errs, fmt.Errorf("DB_POOL_MAX_CONN_IDLE_TIME must be greater than zero"))
	}
	if c.DBPoolHealthCheckPeriod <= 0 {
		errs = append(errs, fmt.Errorf("DB_POOL_HEALTH_CHECK_PERIOD must be greater than zero"))
	}
	if c.RateLimitPerMin <= 0 {
		errs = append(errs, fmt.Errorf("RATE_LIMIT_PER_MINUTE must be greater than zero"))
	}
	if c.WebSearchEnabled {
		if !isSupportedWebSearchProvider(c.WebSearchProvider) {
			errs = append(errs, fmt.Errorf("WEB_SEARCH_PROVIDER must be one of: tavily"))
		}
		if strings.TrimSpace(c.WebSearchAPIKey) == "" {
			errs = append(errs, fmt.Errorf("WEB_SEARCH_API_KEY is required when WEB_SEARCH_ENABLED=true"))
		}
		if strings.TrimSpace(c.WebSearchBaseURL) == "" {
			errs = append(errs, fmt.Errorf("WEB_SEARCH_BASE_URL is required when WEB_SEARCH_ENABLED=true"))
		} else if _, err := url.ParseRequestURI(c.WebSearchBaseURL); err != nil {
			errs = append(errs, fmt.Errorf("WEB_SEARCH_BASE_URL must be a valid URL"))
		}
	}
	if c.WebSearchTimeout <= 0 {
		errs = append(errs, fmt.Errorf("WEB_SEARCH_TIMEOUT must be greater than zero"))
	}
	if c.WebSearchMaxResults <= 0 {
		errs = append(errs, fmt.Errorf("WEB_SEARCH_MAX_RESULTS must be greater than zero"))
	}
	if c.WebSearchMaxResults > 10 {
		errs = append(errs, fmt.Errorf("WEB_SEARCH_MAX_RESULTS must be less than or equal to 10"))
	}
	if c.CORSAllowCredentials && containsWildcardOrigin(c.AllowedOrigins) {
		errs = append(errs, fmt.Errorf("CORS_ALLOW_CREDENTIALS cannot be true when ALLOWED_ORIGINS contains *"))
	}
	if c.MaxUploadBytes <= 0 || c.MaxUploadMegabyte <= 0 {
		errs = append(errs, fmt.Errorf("MAX_UPLOAD_MB must be greater than zero"))
	}
	if !isSupportedOCREngine(c.OCREngine) {
		errs = append(errs, fmt.Errorf("OCR_ENGINE must be one of: tesseract, ollama, disabled"))
	}
	if strings.EqualFold(strings.TrimSpace(c.OCREngine), "ollama") {
		if strings.TrimSpace(c.OCRHost) == "" {
			errs = append(errs, fmt.Errorf("OCR_HOST is required when OCR_ENGINE=ollama"))
		}
		if strings.TrimSpace(c.OCRPort) == "" {
			errs = append(errs, fmt.Errorf("OCR_PORT is required when OCR_ENGINE=ollama"))
		}
		if strings.TrimSpace(c.OCRModel) == "" {
			errs = append(errs, fmt.Errorf("OCR_MODEL is required when OCR_ENGINE=ollama"))
		}
		if strings.TrimSpace(c.OCRPrompt) == "" {
			errs = append(errs, fmt.Errorf("OCR_PROMPT is required when OCR_ENGINE=ollama"))
		}
	}
	if c.OCRTimeout <= 0 {
		errs = append(errs, fmt.Errorf("OCR_TIMEOUT must be greater than zero"))
	}
	if c.OCRMaxPages <= 0 {
		errs = append(errs, fmt.Errorf("OCR_MAX_PAGES must be greater than zero"))
	}
	if !isSupportedLLMBackend(c.LLMBackend) {
		errs = append(errs, fmt.Errorf("LLM_BACKEND must be one of: ollama, vllm, openai-compatible"))
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

func parsePositiveInt(key, defaultVal string) (int, error) {
	value := getEnv(key, defaultVal)
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("invalid %s: must be greater than zero", key)
	}
	return parsed, nil
}

func parseDuration(key, defaultVal string) (time.Duration, error) {
	value := getEnv(key, defaultVal)
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("invalid %s: must be greater than zero", key)
	}
	return parsed, nil
}

func parseAllowedOrigins(key string) ([]string, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin == "" {
			continue
		}
		if origin == "*" {
			origins = append(origins, origin)
			continue
		}
		u, err := url.Parse(origin)
		if err != nil || u.Scheme == "" || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
			return nil, fmt.Errorf("invalid %s origin: %s", key, origin)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("invalid %s origin scheme: %s", key, origin)
		}
		origins = append(origins, strings.ToLower(origin))
	}
	return origins, nil
}

func containsWildcardOrigin(origins []string) bool {
	for _, origin := range origins {
		if origin == "*" {
			return true
		}
	}
	return false
}

func loadSystemPrompt() (string, error) {
	promptFile := strings.TrimSpace(os.Getenv("SYSTEM_PROMPT_FILE"))
	if promptFile == "" {
		return getEnv("SYSTEM_PROMPT", defaultSystemPrompt), nil
	}

	b, err := os.ReadFile(promptFile)
	if err != nil {
		return "", fmt.Errorf("read SYSTEM_PROMPT_FILE: %w", err)
	}
	prompt := strings.TrimSpace(string(b))
	if prompt == "" {
		return "", fmt.Errorf("SYSTEM_PROMPT_FILE must not be empty")
	}

	return prompt, nil
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

func isSupportedLLMBackend(backend string) bool {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "ollama", "vllm", "openai-compatible":
		return true
	default:
		return false
	}
}

func isSupportedOCREngine(engine string) bool {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "tesseract", "ollama", "disabled":
		return true
	default:
		return false
	}
}

func isSupportedWebSearchProvider(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "tavily":
		return true
	default:
		return false
	}
}
