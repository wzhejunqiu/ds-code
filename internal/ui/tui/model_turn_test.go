package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/agent"
)

func TestFinalizeLastAssistant_onlyTouchesTrailingBlock(t *testing.T) {
	started := time.Now().Add(-30 * time.Second)
	m := model{chat: []chatBlock{
		{role: "assistant", reasoningStartedAt: started},
		{role: "tool", toolName: "read_file"},
		{role: "assistant", reasoningStartedAt: time.Now().Add(-2 * time.Second), streaming: true},
	}}
	m.finalizeLastAssistant(time.Now())

	if !m.chat[0].reasoningEndedAt.IsZero() {
		t.Fatal("historical assistant block should not be finalized")
	}
	if m.chat[2].reasoningEndedAt.IsZero() || m.chat[2].streaming {
		t.Fatal("expected last assistant block finalized and not streaming")
	}
}

func TestApplyTurnMetrics_prefersLastAssistantWithContent(t *testing.T) {
	m := model{chat: []chatBlock{
		{role: "user"},
		{role: "assistant"},
		{role: "tool", toolName: "read_file"},
		{role: "assistant"},
	}}
	m.chat[1].content.WriteString("preamble")
	// Last assistant block is empty; metrics should attach to the reply with content.
	m.chat[3].reasoning.WriteString("think")

	m.applyTurnMetrics(&agent.TurnResult{
		TurnDuration:           5*time.Second + 200*time.Millisecond,
		FinalReasoningDuration: 1200 * time.Millisecond,
	})

	if m.chat[1].turnDuration != 5*time.Second+200*time.Millisecond {
		t.Fatalf("turnDuration on content block = %v", m.chat[1].turnDuration)
	}
	if m.chat[3].turnDuration != 0 {
		t.Fatalf("empty trailing block should not get turnDuration: %v", m.chat[3].turnDuration)
	}

	out := renderChat(m.chat, 60, time.Now(), false)
	if !strings.Contains(out, "task took 5.2s") {
		t.Fatalf("expected turn duration in output:\n%s", out)
	}
}
