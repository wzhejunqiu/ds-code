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

func TestResultStatusFromStore(t *testing.T) {
	tests := []struct {
		status subagentstore.Status
		want   ResultStatus
		err    bool
	}{
		{subagentstore.StatusCompleted, ResultCompleted, false},
		{subagentstore.StatusError, ResultFailed, false},
		{subagentstore.StatusKilled, ResultKilled, false},
		{subagentstore.StatusRunning, "", true},
		{subagentstore.Status("unknown"), "", true},
	}
	for _, tt := range tests {
		got, err := resultStatusFromStore(tt.status)
		if tt.err {
			if err == nil {
				t.Fatalf("status %q: expected error", tt.status)
			}
			continue
		}
		if err != nil {
			t.Fatalf("status %q: unexpected error: %v", tt.status, err)
		}
		if got != tt.want {
			t.Fatalf("status %q: got %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestNotificationFormat_inlineXML(t *testing.T) {
	n := Notification{
		AgentID:   "a1",
		ToolUseID: "tc1",
		Status:    ResultCompleted,
		Summary:   `Agent "x" completed`,
		Result:    "short <result>",
	}
	out := n.Format()
	if !strings.Contains(out, "<task-notification>") {
		t.Fatal("expected wrapper")
	}
	if !strings.Contains(out, "<result>short &lt;result&gt;</result>") {
		t.Fatalf("expected escaped result, got %q", out)
	}
	if strings.Contains(out, "<usage>") || strings.Contains(out, "total_tokens") {
		t.Fatal("usage should not appear in notification")
	}
	if strings.Contains(out, "<output-file>") {
		t.Fatal("inline should not have output-file")
	}
}

func TestNotificationFormat_spillIncludesHint(t *testing.T) {
	n := Notification{
		AgentID:    "a1",
		ToolUseID:  "tc1",
		OutputFile: "/tmp/out.output",
		Status:     ResultCompleted,
		Summary:    `Agent "x" completed`,
	}
	out := n.Format()
	if !strings.Contains(out, "<output-file>/tmp/out.output</output-file>") {
		t.Fatalf("expected output-file, got %q", out)
	}
	if !strings.Contains(out, "read_file") || !strings.Contains(out, "/tmp/out.output") {
		t.Fatalf("expected SavedResultHint after notification: %q", out)
	}
	if strings.Contains(out, "<result>") {
		t.Fatal("spill should not have result")
	}
}

func TestNotificationQueue_asyncCompletionUsesPrioLater(t *testing.T) {
	q := NewNotificationQueue()
	q.Enqueue(Notification{AgentID: "async-1", Status: ResultCompleted}, PrioLater)
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
	if got := notificationPriority(ctx); got != PrioNow {
		t.Fatalf("idle: got %v, want PrioNow", got)
	}
}
