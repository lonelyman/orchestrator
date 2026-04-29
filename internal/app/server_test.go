package app

import (
	"testing"
	"time"

	"github.com/enterprise-ai/orchestrator/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConfigurePostgresPool(t *testing.T) {
	poolConfig, err := pgxpool.ParseConfig("postgres://user:pass@localhost:5432/orchestrator")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	cfg := &config.AppConfig{
		DBPoolMaxConns:          50,
		DBPoolMinConns:          5,
		DBPoolMaxConnLifetime:   45 * time.Minute,
		DBPoolMaxConnIdleTime:   10 * time.Minute,
		DBPoolHealthCheckPeriod: 30 * time.Second,
	}

	configurePostgresPool(poolConfig, cfg)

	if poolConfig.MaxConns != 50 {
		t.Fatalf("expected max conns 50, got %d", poolConfig.MaxConns)
	}
	if poolConfig.MinConns != 5 {
		t.Fatalf("expected min conns 5, got %d", poolConfig.MinConns)
	}
	if poolConfig.MaxConnLifetime != 45*time.Minute {
		t.Fatalf("expected lifetime 45m, got %s", poolConfig.MaxConnLifetime)
	}
	if poolConfig.MaxConnIdleTime != 10*time.Minute {
		t.Fatalf("expected idle time 10m, got %s", poolConfig.MaxConnIdleTime)
	}
	if poolConfig.HealthCheckPeriod != 30*time.Second {
		t.Fatalf("expected health period 30s, got %s", poolConfig.HealthCheckPeriod)
	}
}
