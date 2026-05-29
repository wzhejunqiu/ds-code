package usageagg_test

import (
	"context"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/session/usageagg"
)

func TestTotalForSession_sumsMainAndSubagent(t *testing.T) {
	ctx := context.Background()
	main := session.NewMemoryStore()
	sub := subagentstore.NewMemoryStore()

	sess, err := main.CreateSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}
	if err := main.AddUsage(ctx, sess.ID, llm.Usage{PromptTokens: 100, CompletionTokens: 50}); err != nil {
		t.Fatal(err)
	}

	run, err := sub.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID:  sess.ID,
		ParentToolCallID: "tc-1",
		Label:            "probe",
		Prompt:           "go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sub.AddUsage(ctx, run.ID, llm.Usage{PromptTokens: 200, CompletionTokens: 80}); err != nil {
		t.Fatal(err)
	}

	total, err := usageagg.TotalForSession(ctx, main, sub, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if total.PromptTokensTotal != 300 || total.CompletionTokensTotal != 130 {
		t.Fatalf("got %+v", total)
	}
}

func TestUsageByAgentType_groupsRuns(t *testing.T) {
	ctx := context.Background()
	sub := subagentstore.NewMemoryStore()
	sessID := "parent-1"

	explore, err := sub.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID: sessID, ParentToolCallID: "tc1", AgentType: "Explore", Prompt: "a",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = sub.AddUsage(ctx, explore.ID, llm.Usage{PromptTokens: 100, CompletionTokens: 40})

	verify, err := sub.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID: sessID, ParentToolCallID: "tc2", AgentType: "verification", Prompt: "b",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = sub.AddUsage(ctx, verify.ID, llm.Usage{PromptTokens: 50, CompletionTokens: 10})

	rows, err := usageagg.UsageByAgentType(ctx, sub, sessID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 types, got %d", len(rows))
	}
	if rows[0].AgentType != "Explore" || rows[0].Snapshot.PromptTokensTotal != 100 {
		t.Fatalf("explore row: %+v", rows[0])
	}
}
