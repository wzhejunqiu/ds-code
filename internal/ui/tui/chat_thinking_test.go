package tui

import (
	"strings"
	"testing"
	"time"
)

// Simulates two LLM sub-rounds: each gets its own assistant block with thinking.
func TestRenderChatMultipleThinkingBlocks(t *testing.T) {
	now := time.Date(2026, 5, 17, 12, 0, 10, 0, time.UTC)
	r1Start := now.Add(-8 * time.Second)
	r1End := now.Add(-6 * time.Second)
	r2Start := now.Add(-3 * time.Second)
	r2End := now.Add(-1 * time.Second)

	blocks := []chatBlock{
		{role: chatRoleAssistant, reasoningOpen: true},
		{role: chatRoleTool, toolName: "read_file", toolArgs: "path=a.go", toolResult: "ok"},
		{role: chatRolePlanning, planningStartedAt: r1End},
		{role: chatRoleAssistant, reasoningOpen: true},
	}
	blocks[0].reasoning.WriteString("plan round 1")
	blocks[0].reasoningStartedAt = r1Start
	blocks[0].reasoningEndedAt = r1End
	blocks[0].reasoningDuration = r1End.Sub(r1Start)
	blocks[0].content.WriteString("step 1 done")

	blocks[3].reasoning.WriteString("plan round 2")
	blocks[3].reasoningStartedAt = r2Start
	blocks[3].reasoningEndedAt = r2End
	blocks[3].reasoningDuration = r2End.Sub(r2Start)
	blocks[3].content.WriteString("final answer")

	out := renderChat(blocks, 60, now, false)
	thoughtCount := strings.Count(out, "thought for")
	if thoughtCount < 2 {
		t.Fatalf("expected at least 2 independent thought labels, got %d:\n%s", thoughtCount, out)
	}
	r1 := strings.Index(out, "plan round 1")
	r2 := strings.Index(out, "plan round 2")
	t1 := strings.Index(out, "thought for")
	t2 := strings.LastIndex(out, "thought for")
	if r1 < 0 || r2 < 0 || t1 < 0 || t2 < 0 {
		t.Fatalf("missing segments:\n%s", out)
	}
	if !(t1 < r1 && r1 < r2 && t2 > r1) {
		t.Fatalf("each thinking label should precede its own reasoning body:\n%s", out)
	}
}
