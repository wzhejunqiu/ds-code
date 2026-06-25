package context

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
)

func TestSnipToolResults_keepUserTurns(t *testing.T) {
	var msgs []llm.Message
	for range 4 {
		msgs = append(msgs,
			llm.Message{Role: role.User, Content: "q"},
			llm.Message{Role: role.Assistant, Content: "a"},
			llm.Message{Role: role.Tool, Content: strings.Repeat("x", 500)},
		)
	}
	out := SnipToolResults(msgs, 1)
	if !strings.Contains(out[2].Content, "snipped") {
		t.Fatalf("old tool should be snipped: %q", out[2].Content)
	}
	if strings.Contains(out[len(out)-1].Content, "snipped") {
		t.Fatalf("recent tool should remain: %q", out[len(out)-1].Content)
	}
}

func TestSnipToolResults_aggressiveSnipsOldTurn(t *testing.T) {
	msgs := []llm.Message{
		{Role: role.User, Content: "old q"},
		{Role: role.Assistant, Content: "old a"},
		{Role: role.Tool, Content: strings.Repeat("z", 500)},
		{Role: role.User, Content: "current q"},
		{Role: role.Assistant, Content: "current a"},
		{Role: role.Tool, Content: strings.Repeat("y", 500)},
	}
	out := SnipToolResults(msgs, 0)
	if !strings.Contains(out[2].Content, "snipped") {
		t.Fatalf("old turn tool should be snipped: %q", out[2].Content)
	}
	if strings.Contains(out[len(out)-1].Content, "snipped") {
		t.Fatalf("current turn tool should remain: %q", out[len(out)-1].Content)
	}
}

func TestSnipToolResults_protectsCurrentTurnMultiSubRound(t *testing.T) {
	msgs := []llm.Message{
		{Role: role.User, Content: "analyze grep"},
	}
	for range 5 {
		msgs = append(msgs,
			llm.Message{Role: role.Assistant, Content: "thinking"},
			llm.Message{Role: role.Tool, Content: strings.Repeat("m", 500)},
		)
	}
	out := SnipToolResults(msgs, 0)
	for i, m := range out {
		if m.Role != role.Tool {
			continue
		}
		if strings.Contains(m.Content, "snipped") {
			t.Fatalf("current turn tool at %d should not be snipped: %q", i, m.Content)
		}
	}
}

func TestSnipToolResults_skipsShortToolResults(t *testing.T) {
	msgs := []llm.Message{
		{Role: role.User, Content: "old"},
		{Role: role.Assistant, Content: "a"},
		{Role: role.Tool, Content: "ok"},
		{Role: role.User, Content: "new"},
	}
	out := SnipToolResults(msgs, 0)
	if strings.Contains(out[2].Content, "snipped") {
		t.Fatalf("short tool result should remain: %q", out[2].Content)
	}
}

func TestSnipProtectedFrom_noUserMessages(t *testing.T) {
	msgs := []llm.Message{
		{Role: role.Assistant, Content: "a"},
		{Role: role.Tool, Content: strings.Repeat("t", 500)},
	}
	if got := snipProtectedFrom(msgs, 0); got != len(msgs) {
		t.Fatalf("protectedFrom = %d, want %d", got, len(msgs))
	}
}
