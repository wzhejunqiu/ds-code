package spawn

import (
	"context"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

func TestCountToolUses(t *testing.T) {
	sub := subagentstore.NewMemoryStore()
	run, err := sub.CreateRun(context.Background(), subagentstore.CreateRunParams{
		ParentSessionID:  "sess",
		ParentToolCallID: "tc1",
		AgentType:        "Explore",
		SpawnKind:        subagentstore.SpawnAsync,
		Prompt:           "work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := countToolUses(context.Background(), sub, run.ID); n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
	_ = sub.AppendMessage(context.Background(), subagentstore.Message{
		RunID: run.ID, Role: role.Tool, Content: "ok", ToolName: "read_file",
	})
	_ = sub.AppendMessage(context.Background(), subagentstore.Message{
		RunID: run.ID, Role: role.Assistant, Content: "done",
	})
	_ = sub.AppendMessage(context.Background(), subagentstore.Message{
		RunID: run.ID, Role: role.Tool, Content: "ok2", ToolName: "grep",
	})
	if n := countToolUses(context.Background(), sub, run.ID); n != 2 {
		t.Fatalf("expected 2 tool messages, got %d", n)
	}
}

func TestNotificationFormat_includesToolUses(t *testing.T) {
	n := Notification{
		AgentID:      "a1",
		Status:       "completed",
		ToolUseCount: 3,
		DurationMS:   100,
	}
	out := n.Format()
	if !strings.Contains(out, `"tool_uses":3`) {
		t.Fatalf("expected tool_uses in JSON, got %q", out)
	}
}

func TestNotificationQueue_asyncCompletionUsesPrioLater(t *testing.T) {
	q := NewNotificationQueue()
	q.Enqueue(Notification{AgentID: "async-1", Status: "completed"}, PrioLater)
	if later := q.Drain(PrioLater); len(later) != 1 {
		t.Fatalf("expected 1 PrioLater notification, got %d", len(later))
	}
	if next := q.Drain(PrioNext); len(next) != 0 {
		t.Fatalf("expected no PrioNext notifications, got %d", len(next))
	}
}

func TestNotificationPriority_activeTurn(t *testing.T) {
	ctx := agent.WithActiveTurn(context.Background())
	if got := notificationPriority(ctx); got != PrioLater {
		t.Fatalf("active turn: got %v, want PrioLater", got)
	}
	ctx = agent.WithoutActiveTurn(ctx)
	if got := notificationPriority(ctx); got != PrioNext {
		t.Fatalf("idle: got %v, want PrioNext", got)
	}
}
