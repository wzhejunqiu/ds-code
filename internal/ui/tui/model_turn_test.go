package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
)

func TestFinalizeLastAssistant_onlyTouchesTrailingBlock(t *testing.T) {
	started := time.Now().Add(-30 * time.Second)
	m := model{chat: []chat.Block{
		{Role: chat.RoleAssistant, ReasoningStartedAt: started},
		{Role: chat.RoleTool, ToolName: "read_file"},
		{Role: chat.RoleAssistant, ReasoningStartedAt: time.Now().Add(-2 * time.Second), Streaming: true},
	}}
	m.finalizeLastAssistant(time.Now())

	if !m.chat[0].ReasoningEndedAt.IsZero() {
		t.Fatal("historical assistant block should not be finalized")
	}
	if m.chat[2].ReasoningEndedAt.IsZero() || m.chat[2].Streaming {
		t.Fatal("expected last assistant block finalized and not streaming")
	}
}

func TestApplyTurnMetrics_prefersLastAssistantWithContent(t *testing.T) {
	m := model{chat: []chat.Block{
		{Role: chat.RoleUser},
		{Role: chat.RoleAssistant},
		{Role: chat.RoleTool, ToolName: "read_file"},
		{Role: chat.RoleAssistant},
	}}
	m.chat[1].Content.WriteString("preamble")
	// Last assistant block is empty; metrics should attach to the reply with content.
	m.chat[3].Reasoning.WriteString("think")

	m.applyTurnMetrics(&agent.TurnResult{
		TurnDuration:           5*time.Second + 200*time.Millisecond,
		FinalReasoningDuration: 1200 * time.Millisecond,
	})

	if m.chat[1].TurnDuration != 5*time.Second+200*time.Millisecond {
		t.Fatalf("turnDuration on content block = %v", m.chat[1].TurnDuration)
	}
	if m.chat[3].TurnDuration != 0 {
		t.Fatalf("empty trailing block should not get turnDuration: %v", m.chat[3].TurnDuration)
	}

	out := chat.Render(m.chat, 60, time.Now(), false)
	if !strings.Contains(out, "task took 5.2s") {
		t.Fatalf("expected turn duration in output:\n%s", out)
	}
}
