package audit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

type fakeAuditDelegate struct {
	mu      sync.Mutex
	saved   []models.AuditLog
	block   chan struct{}
	started chan struct{}
}

func (f *fakeAuditDelegate) Save(ctx context.Context, log models.AuditLog) error {
	if f.started != nil {
		select {
		case f.started <- struct{}{}:
		default:
		}
	}
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.saved = append(f.saved, log)
	return nil
}

func (f *fakeAuditDelegate) List(_ context.Context, _ models.AuditLogFilter) ([]models.AuditLog, error) {
	return nil, nil
}

func (f *fakeAuditDelegate) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.saved)
}

func TestAsyncAdapter_DrainsOnClose(t *testing.T) {
	delegate := &fakeAuditDelegate{}
	adapter := NewAsyncAdapter(delegate, 1, 10, time.Second)

	for i := 0; i < 3; i++ {
		if err := adapter.Save(context.Background(), models.AuditLog{RequestID: "req"}); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := adapter.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if delegate.count() != 3 {
		t.Fatalf("expected 3 saved logs, got %d", delegate.count())
	}
}

func TestAsyncAdapter_DropsWhenQueueFull(t *testing.T) {
	block := make(chan struct{})
	started := make(chan struct{}, 1)
	delegate := &fakeAuditDelegate{block: block, started: started}
	adapter := NewAsyncAdapter(delegate, 1, 1, 100*time.Millisecond)

	if err := adapter.Save(context.Background(), models.AuditLog{RequestID: "first"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatalf("worker did not start saving first log")
	}
	if err := adapter.Save(context.Background(), models.AuditLog{RequestID: "second"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := adapter.Save(context.Background(), models.AuditLog{RequestID: "third"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if got := adapter.dropped.Load(); got != 1 {
		t.Fatalf("expected one dropped log, got %d", got)
	}

	close(block)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := adapter.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestAsyncAdapter_RejectsAfterClose(t *testing.T) {
	adapter := NewAsyncAdapter(&fakeAuditDelegate{}, 1, 1, time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := adapter.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := adapter.Save(context.Background(), models.AuditLog{}); err != ErrAsyncAuditClosed {
		t.Fatalf("expected ErrAsyncAuditClosed, got %v", err)
	}
}
