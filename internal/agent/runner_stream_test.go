package agent_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

type streamEmitClient struct {
	resp *llm.Response
}

func (c *streamEmitClient) Chat(ctx context.Context, req llm.Request) (*llm.Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.OnStream != nil {
		req.OnStream(llm.StreamDelta{Reasoning: "think"})
		for _, part := range []string{"hel", "lo ", "world"} {
			req.OnStream(llm.StreamDelta{Content: part})
		}
	}
	return c.resp, nil
}

func TestRunner_streamsContentDeltasDuringOnStream(t *testing.T) {
	cfg := testConfig()
	dir := t.TempDir()
	store := session.NewMemoryStore()
	sess, err := store.NewSession("deepseek-v4-pro", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}

	var reasoning []string
	var content []string
	cb := &agent.TurnCallbacks{
		OnReasoningDelta: func(s string) { reasoning = append(reasoning, s) },
		OnContentDelta:   func(s string) { content = append(content, s) },
	}

	reg := tool.NewRegistry()
	r := &agent.Runner{
		LLM: &streamEmitClient{resp: &llm.Response{
			Content:          "hello world",
			ReasoningContent: "think",
			FinishReason:     "stop",
		}},
		Tools:    reg,
		Perm:     permission.NewEngine("auto", dir, false),
		Sessions: store,
		Context: &ctxpkg.Service{
			Cfg: cfg, Store: store, Tools: reg,
			AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: permission.NewEngine("auto", dir, false)},
		},
		Cfg:      cfg,
		MaxTurns: 5,
		Out:      &bytes.Buffer{},
	}

	res, err := r.RunTurn(context.Background(), sess.ID, "hi", cb)
	if err != nil {
		t.Fatal(err)
	}
	if res.FinalContent != "hello world" {
		t.Fatalf("final content = %q", res.FinalContent)
	}
	if strings.Join(reasoning, "") != "think" {
		t.Fatalf("reasoning deltas = %v", reasoning)
	}
	if strings.Join(content, "") != "hello world" {
		t.Fatalf("content deltas = %v", content)
	}
	if len(content) != 3 {
		t.Fatalf("want 3 streamed content chunks, got %d: %v", len(content), content)
	}
}
