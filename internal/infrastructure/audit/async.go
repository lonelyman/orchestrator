package audit

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/enterprise-ai/orchestrator/internal/domain/ports"
)

var ErrAsyncAuditClosed = errors.New("async audit store is closed")

type AsyncAdapter struct {
	delegate ports.AuditPort
	timeout  time.Duration
	queue    chan models.AuditLog

	mu        sync.RWMutex
	closed    bool
	closeOnce sync.Once
	wg        sync.WaitGroup
	dropped   atomic.Uint64
}

func NewAsyncAdapter(delegate ports.AuditPort, workers int, queueSize int, timeout time.Duration) *AsyncAdapter {
	if workers <= 0 {
		workers = 4
	}
	if queueSize <= 0 {
		queueSize = 1000
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	a := &AsyncAdapter{
		delegate: delegate,
		timeout:  timeout,
		queue:    make(chan models.AuditLog, queueSize),
	}
	for i := 0; i < workers; i++ {
		a.wg.Add(1)
		go a.worker(i + 1)
	}
	slog.Info("audit worker pool started", "workers", workers, "queue_size", queueSize)
	return a
}

func (a *AsyncAdapter) Save(ctx context.Context, log models.AuditLog) error {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return ErrAsyncAuditClosed
	}

	select {
	case a.queue <- log:
		return nil
	default:
		dropped := a.dropped.Add(1)
		slog.Warn("audit queue full, dropping log",
			"dropped", dropped,
			"request_id", log.RequestID,
			"path", log.Path,
			"status", log.StatusCode,
		)
		return nil
	}
}

func (a *AsyncAdapter) List(ctx context.Context, filter models.AuditLogFilter) ([]models.AuditLog, error) {
	return a.delegate.List(ctx, filter)
}

func (a *AsyncAdapter) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	a.closeOnce.Do(func() {
		a.mu.Lock()
		a.closed = true
		close(a.queue)
		a.mu.Unlock()
	})

	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("audit worker pool stopped", "dropped", a.dropped.Load())
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *AsyncAdapter) worker(workerID int) {
	defer a.wg.Done()
	for log := range a.queue {
		ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
		err := a.delegate.Save(ctx, log)
		cancel()
		if err != nil {
			slog.Error("save audit log failed",
				"error", err,
				"worker_id", workerID,
				"request_id", log.RequestID,
				"path", log.Path,
			)
		}
	}
}
