package usageagg_test

import (
	"context"
	"math"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/session/usageagg"
)

func TestEstimateCostForSession_sumsMainAndSub(t *testing.T) {
	ctx := context.Background()
	main := session.NewMemoryStore()
	sub := subagentstore.NewMemoryStore()
	sess, _ := main.CreateSession("deepseek-v4-pro", "max", "enabled", "auto", "agent")
	_ = main.AddUsage(ctx, sess.ID, llm.Usage{PromptTokens: 100, CompletionTokens: 50})
	run, _ := sub.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID: sess.ID, ParentToolCallID: "tc-1", Model: "deepseek-v4-flash",
	})
	_ = sub.AddUsage(ctx, run.ID, llm.Usage{PromptTokens: 200, CompletionTokens: 80})

	cost, err := usageagg.EstimateCostForSession(ctx, main, sub, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cost.TotalCNY <= 0 {
		t.Fatalf("total cny = %v", cost.TotalCNY)
	}
	if cost.SubagentCNY <= 0 {
		t.Fatalf("subagent cny = %v", cost.SubagentCNY)
	}
}

func TestEstimateCostForSession_includesTitleInSubagentCNY(t *testing.T) {
	ctx := context.Background()
	main := session.NewMemoryStore()
	sub := subagentstore.NewMemoryStore()
	sess, _ := main.CreateSession("deepseek-v4-pro", "max", "enabled", "auto", "agent")

	titleRun, _ := sub.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID:  sess.ID,
		ParentToolCallID: "session-title",
		RunKind:          subagentstore.RunKindTitle,
		Model:            "deepseek-v4-flash",
	})
	_ = sub.AddUsage(ctx, titleRun.ID, llm.Usage{PromptTokens: 1000, CompletionTokens: 500})
	_ = sub.FinishRun(ctx, titleRun.ID, subagentstore.StatusDone, "")

	taskRun, _ := sub.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID:  sess.ID,
		ParentToolCallID: "tc-task",
		RunKind:          subagentstore.RunKindTask,
		Model:            "deepseek-v4-flash",
	})
	_ = sub.AddUsage(ctx, taskRun.ID, llm.Usage{PromptTokens: 100, CompletionTokens: 50})
	_ = sub.FinishRun(ctx, taskRun.ID, subagentstore.StatusDone, "")

	cost, err := usageagg.EstimateCostForSession(ctx, main, sub, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cost.SubagentCNY <= 0 {
		t.Fatalf("subagent cny = %v", cost.SubagentCNY)
	}
	if math.Abs(cost.TotalCNY-(cost.MainCNY+cost.SubagentCNY)) > 1e-9 {
		t.Fatalf("total %v != main %v + subagent %v", cost.TotalCNY, cost.MainCNY, cost.SubagentCNY)
	}
	titleOnly, err := usageagg.SubagentOnly(ctx, sub, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if titleOnly.Billed == 0 {
		t.Fatal("expected title run tokens in subagent usage")
	}
	if cost.SubSnap.Billed != titleOnly.Billed {
		t.Fatalf("sub snap billed = %d, want %d", cost.SubSnap.Billed, titleOnly.Billed)
	}
}
