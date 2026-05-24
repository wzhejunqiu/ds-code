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
