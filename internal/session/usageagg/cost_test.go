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

func TestEstimateCostForSession_includesAllAgents(t *testing.T) {
	ctx := context.Background()
	main := session.NewMemoryStore()
	sub := subagentstore.NewMemoryStore()
	sess, _ := main.CreateSession("deepseek-v4-pro", "max", "enabled", "auto", "agent")

	exploreRun, _ := sub.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID:  sess.ID,
		ParentToolCallID: "tc-explore",
		AgentType:        "Explore",
		Model:            "deepseek-v4-flash",
	})
	_ = sub.AddUsage(ctx, exploreRun.ID, llm.Usage{PromptTokens: 1000, CompletionTokens: 500})
	_ = sub.FinishRun(ctx, exploreRun.ID, subagentstore.StatusCompleted, "")

	verifyRun, _ := sub.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID:  sess.ID,
		ParentToolCallID: "tc-verify",
		AgentType:        "verification",
		Model:            "deepseek-v4-flash",
	})
	_ = sub.AddUsage(ctx, verifyRun.ID, llm.Usage{PromptTokens: 100, CompletionTokens: 50})
	_ = sub.FinishRun(ctx, verifyRun.ID, subagentstore.StatusCompleted, "")

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
	allUsage, err := usageagg.SubagentOnly(ctx, sub, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if allUsage.Billed == 0 {
		t.Fatal("expected all agent tokens in subagent usage")
	}
	if cost.SubSnap.Billed != allUsage.Billed {
		t.Fatalf("sub snap billed = %d, want %d", cost.SubSnap.Billed, allUsage.Billed)
	}
}
