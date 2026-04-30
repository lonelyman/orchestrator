package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

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
	"github.com/enterprise-ai/orchestrator/internal/infrastructure/websearch"
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
	audit interface {
		Close(context.Context) error
	}
	closeOnce sync.Once
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
	asyncAuditAdapter := auditInfra.NewAsyncAdapter(auditAdapter, cfg.AuditWorkers, cfg.AuditQueueSize, cfg.AuditTimeout)

	ragEngine := rag.New(embedderAdapter, vectorAdapter)
	orch := orchestrator.New(llmAdapter, ragEngine, sessionAdapter, cfg.SystemPrompt)
	webSearchAdapter, err := buildWebSearchAdapter(cfg)
	if err != nil {
		sqlDB.Close()
		pool.Close()
		return nil, err
	}
	if webSearchAdapter != nil {
		orch.RegisterTool(orchestrator.NewWebSearchTool(webSearchAdapter, cfg.WebSearchMaxResults))
		slog.Info("web search tool registered", "provider", cfg.WebSearchProvider, "max_results", cfg.WebSearchMaxResults)
	}

	ldapAdapter := auth.NewLDAPAdapter(cfg.ADServer, cfg.ADPort, cfg.ADBaseDN, cfg.ADDomain, cfg.DevMode, cfg.DevUsername, cfg.DevPassword)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiry)

	fiberApp := api.NewRouter(api.RouterDependencies{
		Config:        cfg,
		JWTManager:    jwtManager,
		SessionStore:  sessionAdapter,
		AuditStore:    asyncAuditAdapter,
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

	return &Server{App: fiberApp, pool: pool, sqlDB: sqlDB, audit: asyncAuditAdapter}, nil
}

func buildWebSearchAdapter(cfg *config.AppConfig) (ports.WebSearchPort, error) {
	if !cfg.WebSearchEnabled {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(cfg.WebSearchProvider)) {
	case "tavily":
		return websearch.NewTavilyAdapter(cfg.WebSearchBaseURL, cfg.WebSearchAPIKey, cfg.WebSearchTimeout), nil
	default:
		return nil, fmt.Errorf("unsupported WEB_SEARCH_PROVIDER: %s", cfg.WebSearchProvider)
	}
}

func (s *Server) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.close(ctx)
}

func (s *Server) close(ctx context.Context) {
	s.closeOnce.Do(func() {
		if s.audit != nil {
			if err := s.audit.Close(ctx); err != nil {
				slog.Error("close audit worker pool failed", "error", err)
			}
		}
		if s.sqlDB != nil {
			s.sqlDB.Close()
		}
		if s.pool != nil {
			s.pool.Close()
		}
	})
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.App == nil {
		s.Close()
		return nil
	}

	err := s.App.ShutdownWithContext(ctx)
	s.close(ctx)
	if errors.Is(err, fiber.ErrNotRunning) {
		return nil
	}
	return err
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
	configurePostgresPool(poolConfig, cfg)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	slog.Info("postgresql connected")

	return pool, nil
}

func configurePostgresPool(poolConfig *pgxpool.Config, cfg *config.AppConfig) {
	poolConfig.MaxConns = int32(cfg.DBPoolMaxConns)
	poolConfig.MinConns = int32(cfg.DBPoolMinConns)
	poolConfig.MaxConnLifetime = cfg.DBPoolMaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.DBPoolMaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.DBPoolHealthCheckPeriod
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
