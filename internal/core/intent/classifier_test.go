package intent

import (
	"testing"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

func TestClassifier_WebSearchStrongKeyword(t *testing.T) {
	result := New().Classify("ข่าว AI ล่าสุด")
	if result.Intent != models.IntentWebSearch {
		t.Fatalf("expected web_search, got %s", result.Intent)
	}
}

func TestClassifier_MCPWinsOverRecencyKeyword(t *testing.T) {
	result := New().Classify("ยอดขายวันนี้เป็นอย่างไร")
	if result.Intent != models.IntentMCP {
		t.Fatalf("expected mcp, got %s", result.Intent)
	}
}

func TestClassifier_RecencyKeywordFallsBackToWebSearch(t *testing.T) {
	result := New().Classify("ราคาทองวันนี้")
	if result.Intent != models.IntentWebSearch {
		t.Fatalf("expected web_search, got %s", result.Intent)
	}
}
