package context

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/role"
)

func TestSnipToolResults_keepRounds(t *testing.T) {
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

func TestSnipToolResults_aggressive(t *testing.T) {
	msgs := []llm.Message{
		{Role: role.User, Content: "q"},
		{Role: role.Assistant, Content: "a"},
		{Role: role.Tool, Content: strings.Repeat("z", 500)},
	}
	out := SnipToolResults(msgs, 0)
	if !strings.Contains(out[2].Content, "snipped") {
		t.Fatalf("expected snip with keepRounds=0")
	}
}
