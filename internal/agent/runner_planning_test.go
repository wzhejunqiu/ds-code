package agent_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hejunqiu/ds-code/internal/agent"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/llm/mock"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/builtin/read_file"
)

// getFailMemoryStore fails Get once tool messages exist (simulates round>0 prepare failure).
type getFailMemoryStore struct {
	*session.MemoryStore
}

func (s *getFailMemoryStore) Get(ctx context.Context, id string) (session.Session, error) {
	msgs, err := s.MemoryStore.ListMessages(ctx, id)
	if err != nil {
		return session.Session{}, err
	}
	for _, m := range msgs {
		if m.Role == role.Tool {
			return session.Session{}, fmt.Errorf("simulated get failure after tools")
		}
	}
	return s.MemoryStore.Get(ctx, id)
}

func TestRunTurn_planningStartSkippedOnRound0(t *testing.T) {
	cfg := testConfig()
	dir := t.TempDir()
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}

	var starts, ends int
	cb := &agent.TurnCallbacks{
		OnPlanningStart: func() { starts++ },
		OnPlanningEnd:   func() { ends++ },
	}

	reg := tool.NewRegistry()
	r := &agent.Runner{
		LLM: &streamEmitClient{resp: &llm.Response{
			Content:          "ok",
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

	if _, err := r.RunTurn(context.Background(), sess.ID, "hi", cb); err != nil {
		t.Fatal(err)
	}
	if starts != 0 {
		t.Fatalf("OnPlanningStart on round 0: got %d, want 0", starts)
	}
	if ends != 1 {
		t.Fatalf("OnPlanningEnd: got %d, want 1", ends)
	}
}

func TestRunTurn_planningStartOnLaterRounds(t *testing.T) {
	cfg := testConfig()
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/foo.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}

	var starts, ends int
	cb := &agent.TurnCallbacks{
		OnPlanningStart: func() { starts++ },
		OnPlanningEnd:   func() { ends++ },
	}

	perm := permission.NewEngine("auto", dir, false)
	reg := tool.NewRegistry()
	reg.Register(&read_file.ReadFileTool{Cfg: cfg, Perm: perm, Strict: false})

	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{
				ToolCalls: []llm.ToolCall{{
					ID: "c1", Name: "read_file", Arguments: `{"path":"foo.txt"}`,
				}},
				ReasoningContent: "think",
				FinishReason:     "tool_calls",
			},
			{Content: "done", FinishReason: "stop"},
		},
	}

	ctxSvc := &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg, AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}}
	r := &agent.Runner{
		LLM: mockLLM, Tools: reg, Perm: perm, Sessions: store, Context: ctxSvc,
		Cfg: cfg, MaxTurns: 5, Out: &bytes.Buffer{},
	}

	if _, err := r.RunTurn(context.Background(), sess.ID, "read", cb); err != nil {
		t.Fatal(err)
	}
	if starts != 1 {
		t.Fatalf("OnPlanningStart after round 0: got %d, want 1", starts)
	}
	if ends != 2 {
		t.Fatalf("OnPlanningEnd: got %d, want 2 (one per LLM round)", ends)
	}
}

func TestRunTurn_prepareFailureEndsPlanning(t *testing.T) {
	cfg := testConfig()
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/foo.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	inner := session.NewMemoryStore()
	store := &getFailMemoryStore{MemoryStore: inner}
	sess, err := inner.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}

	var starts, ends int
	cb := &agent.TurnCallbacks{
		OnPlanningStart: func() { starts++ },
		OnPlanningEnd:   func() { ends++ },
	}

	perm := permission.NewEngine("auto", dir, false)
	reg := tool.NewRegistry()
	reg.Register(&read_file.ReadFileTool{Cfg: cfg, Perm: perm, Strict: false})

	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{
				ToolCalls: []llm.ToolCall{{
					ID: "c1", Name: "read_file", Arguments: `{"path":"foo.txt"}`,
				}},
				FinishReason: "tool_calls",
			},
		},
	}

	ctxSvc := &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg, AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}}
	r := &agent.Runner{
		LLM: mockLLM, Tools: reg, Perm: perm, Sessions: store, Context: ctxSvc,
		Cfg: cfg, MaxTurns: 5, Out: &bytes.Buffer{},
	}

	_, err = r.RunTurn(context.Background(), sess.ID, "read", cb)
	if err == nil {
		t.Fatal("expected prepare failure on round 1")
	}
	if starts != 1 {
		t.Fatalf("OnPlanningStart before failed prepare: got %d, want 1", starts)
	}
	if ends != 2 {
		t.Fatalf("OnPlanningEnd (round 0 + prepare cleanup): got %d, want 2", ends)
	}
}
