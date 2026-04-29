package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/enterprise-ai/orchestrator/config"
	"github.com/enterprise-ai/orchestrator/internal/api"
	"github.com/enterprise-ai/orchestrator/internal/api/handlers"
	"github.com/enterprise-ai/orchestrator/internal/core/orchestrator"
	"github.com/enterprise-ai/orchestrator/internal/core/rag"
	"github.com/enterprise-ai/orchestrator/internal/domain/ports"
	auditInfra "github.com/enterprise-ai/orchestrator/internal/infrastructure/audit"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/auth"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/embedder"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/llm"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/migrations"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/session"
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/vector"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	pgxvector "github.com/pgvector/pgvector-go/pgx"
)

type Server struct {
	App   *fiber.App
	pool  *pgxpool.Pool
	sqlDB *sql.DB
}

func New(ctx context.Context, cfg *config.AppConfig) (*Server, error) {
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)

	migrationDB, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("open migration db: %w", err)
	}
	defer migrationDB.Close()

	migrationCtx, cancel := context.WithTimeout(ctx, cfg.MigrationTimeout)
	defer cancel()
	if err := migrations.Up(migrationCtx, migrationDB); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	slog.Info("database migrations applied")

	pool, err := connectPostgres(ctx, cfg)
	if err != nil {
		return nil, err
	}

	sqlDB := stdlib.OpenDBFromPool(pool)

	llmURL := fmt.Sprintf("http://%s:%s", cfg.LLMHost, cfg.LLMPort)
	llmAdapter, err := buildLLMAdapter(cfg, llmURL)
	if err != nil {
		sqlDB.Close()
		pool.Close()
		return nil, err
	}

	embedURL := fmt.Sprintf("http://%s:%s", cfg.EmbedHost, cfg.EmbedPort)
	embedderAdapter := embedder.NewNomicAdapter(embedURL, cfg.EmbedModel)
	vectorAdapter := vector.NewPgvectorAdapter(pool)
	sessionAdapter := session.NewPostgresAdapter(pool, cfg.SessionExpiry)
	auditAdapter := auditInfra.NewPostgresAdapter(pool)

	ragEngine := rag.New(embedderAdapter, vectorAdapter)
	orch := orchestrator.New(llmAdapter, ragEngine, sessionAdapter, cfg.SystemPrompt)

	ldapAdapter := auth.NewLDAPAdapter(cfg.ADServer, cfg.ADPort, cfg.ADBaseDN, cfg.ADDomain, cfg.DevMode, cfg.DevUsername, cfg.DevPassword)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiry)

	fiberApp := api.NewRouter(api.RouterDependencies{
		Config:        cfg,
		JWTManager:    jwtManager,
		SessionStore:  sessionAdapter,
		AuditStore:    auditAdapter,
		AuthHandler:   handlers.NewAuthHandler(ldapAdapter, jwtManager),
		HealthHandler: handlers.NewHealthHandlerWithTimeout(orch, vectorAdapter, cfg.HealthTimeout),
		ChatHandler:   handlers.NewChatHandlerWithTimeout(orch, cfg.ChatTimeout, cfg.LLMModel, cfg.LLMBackend),
		RAGHandler: handlers.NewRAGHandlerWithOptions(ragEngine, rag.ParseOptions{
			MaxBytes:    cfg.MaxUploadBytes,
			OCREngine:   cfg.OCREngine,
			OCRBaseURL:  fmt.Sprintf("http://%s:%s", cfg.OCRHost, cfg.OCRPort),
			OCRModel:    cfg.OCRModel,
			OCRPrompt:   cfg.OCRPrompt,
			OCRTimeout:  cfg.OCRTimeout,
			OCRMaxPages: cfg.OCRMaxPages,
		}, cfg.RAGIngestTimeout),
		DocumentHandler: handlers.NewDocumentHandler(vectorAdapter),
		AuditHandler:    handlers.NewAuditHandler(auditAdapter),
	})

	return &Server{App: fiberApp, pool: pool, sqlDB: sqlDB}, nil
}

func (s *Server) Close() {
	if s.sqlDB != nil {
		s.sqlDB.Close()
	}
	if s.pool != nil {
		s.pool.Close()
	}
}

func connectPostgres(ctx context.Context, cfg *config.AppConfig) (*pgxpool.Pool, error) {
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvector.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	slog.Info("postgresql connected")

	return pool, nil
}

func buildLLMAdapter(cfg *config.AppConfig, baseURL string) (ports.LLMPort, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.LLMBackend)) {
	case "ollama":
		return llm.NewOllamaAdapter(baseURL, cfg.LLMModel), nil
	case "vllm", "openai-compatible":
		return llm.NewOpenAICompatibleAdapter(baseURL, cfg.LLMModel, cfg.LLMAPIKey), nil
	default:
		return nil, fmt.Errorf("unsupported LLM_BACKEND: %s", cfg.LLMBackend)
	}
}
