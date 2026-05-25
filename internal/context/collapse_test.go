package context

import (
	"context"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
)

func TestApplyCollapseIfNeeded_replacesOldTurns(t *testing.T) {
	svc := &Service{
		Cfg: &config.Config{
			Context: config.ContextConfig{
				WindowTokens:           1000,
				CollapseThresholdRatio: 0.50,
				KeepRecentTurns:        1,
			},
		},
		collapse: newCollapseTracker(),
	}
	big := strings.Repeat("a", 2000)
	msgs := []llm.Message{
		{Role: role.User, Content: big},
		{Role: role.Assistant, Content: "old reply 1"},
		{Role: role.User, Content: strings.Repeat("c", 2000)},
		{Role: role.Assistant, Content: "old reply 2"},
		{Role: role.User, Content: strings.Repeat("b", 2000)},
		{Role: role.Assistant, Content: "new reply"},
	}
	view := &APIContextView{Messages: msgs, WindowTokens: 1000}
	svc.applyCollapseIfNeeded(context.Background(), "sess-1", view)
	if len(view.Messages) < 3 {
		t.Fatalf("expected collapsed view, got %d messages", len(view.Messages))
	}
	if !strings.Contains(view.Messages[0].Content, "<conversation-summary>") {
		t.Fatalf("expected summary message, got %q", view.Messages[0].Content)
	}
}

func TestApplyCollapseIfNeeded_afterCompactRebuild(t *testing.T) {
	svc := &Service{
		Cfg: &config.Config{
			Context: config.ContextConfig{
				WindowTokens:           1000,
				CollapseThresholdRatio: 0.50,
				KeepRecentTurns:        1,
			},
		},
		collapse: newCollapseTracker(),
	}
	big := strings.Repeat("x", 2000)
	view := &APIContextView{
		Messages: []llm.Message{
			{Role: role.User, Content: big},
			{Role: role.Assistant, Content: "old 1"},
			{Role: role.User, Content: strings.Repeat("y", 2000)},
			{Role: role.Assistant, Content: "old 2"},
			{Role: role.User, Content: strings.Repeat("z", 2000)},
			{Role: role.Assistant, Content: "new"},
		},
		WindowTokens: 1000,
	}
	svc.applyCollapseIfNeeded(context.Background(), "sess-2", view)
	if !strings.Contains(view.Messages[0].Content, "<conversation-summary>") {
		t.Fatalf("expected collapse after rebuild, got %q", view.Messages[0].Content)
	}
}

func TestApplyCollapseIfNeeded_secondPassAfterCollapsedView(t *testing.T) {
	svc := &Service{
		Cfg: &config.Config{
			Context: config.ContextConfig{
				WindowTokens:           1000,
				CollapseThresholdRatio: 0.50,
				KeepRecentTurns:        1,
			},
		},
		collapse: newCollapseTracker(),
		LLM:      nil,
	}
	big := strings.Repeat("q", 2000)
	view := &APIContextView{
		Messages: []llm.Message{
			{Role: role.User, Content: big},
			{Role: role.Assistant, Content: "old 1"},
			{Role: role.User, Content: strings.Repeat("r", 2000)},
			{Role: role.Assistant, Content: "old 2"},
			{Role: role.User, Content: strings.Repeat("s", 2000)},
			{Role: role.Assistant, Content: "new"},
		},
		WindowTokens: 1000,
	}
	svc.applyCollapseIfNeeded(context.Background(), "sess-pass2", view)
	if !strings.Contains(view.Messages[0].Content, "<conversation-summary>") {
		t.Fatalf("expected first collapse, got %q", view.Messages[0].Content)
	}
	firstLen := len(view.Messages)
	// Second pass: still over threshold; must not panic and should keep a valid view.
	svc.applyCollapseIfNeeded(context.Background(), "sess-pass2", view)
	if len(view.Messages) == 0 {
		t.Fatal("expected non-empty view after second collapse pass")
	}
	if !strings.Contains(view.Messages[0].Content, "<conversation-summary>") {
		t.Fatal("expected summary message to remain at head")
	}
	if len(view.Messages) > firstLen+3 {
		t.Fatalf("unexpected message growth: before=%d after=%d", firstLen, len(view.Messages))
	}
}

func TestApplyCollapseIfNeeded_fallbackWithoutLLM(t *testing.T) {
	svc := &Service{
		Cfg: &config.Config{
			Context: config.ContextConfig{
				WindowTokens:           1000,
				CollapseThresholdRatio: 0.50,
				KeepRecentTurns:        1,
			},
		},
		collapse: newCollapseTracker(),
		LLM:      nil,
	}
	big := strings.Repeat("z", 5000)
	view := &APIContextView{
		Messages: []llm.Message{
			{Role: role.User, Content: big},
			{Role: role.Assistant, Content: "old 1"},
			{Role: role.User, Content: strings.Repeat("y", 2000)},
			{Role: role.Assistant, Content: "old 2"},
			{Role: role.User, Content: strings.Repeat("w", 2000)},
			{Role: role.Assistant, Content: "new"},
		},
		WindowTokens: 1000,
	}
	svc.applyCollapseIfNeeded(context.Background(), "sess-3", view)
	if len(view.Messages) < 3 {
		t.Fatal("expected collapsed view")
	}
	summary := view.Messages[0].Content
	if len(summary) > 8500 {
		t.Fatalf("fallback summary too large: %d chars", len(summary))
	}
	if !strings.Contains(summary, "<conversation-summary>") {
		t.Fatalf("expected summary wrapper, got %q", summary)
	}
}
