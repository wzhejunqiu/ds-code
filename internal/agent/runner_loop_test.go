package agent

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/llm/mock"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
)

func loopTestConfig() *config.Config {
	return &config.Config{
		LLM:     config.LLMConfig{MaxTokens: 4096},
		Context: config.ContextConfig{ToolResultMaxChars: 100000},
		Agent:   config.AgentConfig{MaxTurns: 25},
	}
}

func TestAttachStreamHandlers_contentAndReasoning(t *testing.T) {
	var content, reasoning []string
	planningEnded := false
	cb := &TurnCallbacks{
		OnPlanningEnd: func() { planningEnded = true },
		OnContentDelta: func(s string) {
			content = append(content, s)
		},
		OnReasoningDelta: func(s string) {
			reasoning = append(reasoning, s)
		},
	}
	r := &Runner{}
	stream := &subRoundStream{}
	handler := r.attachStreamHandlers(cb, 0, stream)
	if handler == nil {
		t.Fatal("expected handler")
	}
	handler(llm.StreamDelta{Reasoning: "think"})
	handler(llm.StreamDelta{Content: "hi"})
	if !planningEnded {
		t.Fatal("expected planning end after stream content")
	}
	if strings.Join(content, "") != "hi" || strings.Join(reasoning, "") != "think" {
		t.Fatalf("content=%v reasoning=%v", content, reasoning)
	}
	if !stream.streamedContent {
		t.Fatal("expected streamedContent")
	}
}

func TestFinishTerminalRound_writesOutWhenNoCallback(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	r := &Runner{Sessions: store, Out: &out}
	stream := &subRoundStream{}
	result := &TurnResult{}
	got, err := r.finishTerminalRound(
		context.Background(),
		sess.ID,
		sess.Model,
		&llm.Response{Content: "final"},
		stream,
		time.Now(),
		result,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.FinalContent != "final" || out.String() != "final" {
		t.Fatalf("content=%q out=%q", got.FinalContent, out.String())
	}
	msgs, _ := store.ListMessages(context.Background(), sess.ID)
	if len(msgs) != 1 || msgs[0].Role != role.Assistant {
		t.Fatalf("messages = %+v", msgs)
	}
}

func TestChatWithCompactRetry_retriesAfterCompact(t *testing.T) {
	cfg := loopTestConfig()
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}
	mockLLM := &mock.Client{
		Errors:    []error{fmt.Errorf("context length exceeded")},
		Responses: []*llm.Response{{Content: "ok"}},
	}
	perm := permission.NewEngine("auto", t.TempDir(), false)
	reg := tool.NewRegistry()
	ctxSvc := &ctxpkg.Service{Cfg: cfg, Store: store, Tools: reg, LLM: mockLLM, AtExpander: &ctxpkg.AtExpander{Cfg: cfg, Perm: perm}}
	r := &Runner{LLM: mockLLM, Context: ctxSvc, Cfg: cfg, Sessions: store}

	req := llm.Request{Messages: []llm.Message{{Role: role.User, Content: "hi"}}}
	resp, err := r.chatWithCompactRetry(context.Background(), sess.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ok" || len(mockLLM.Calls) != 2 {
		t.Fatalf("resp=%+v calls=%d", resp, len(mockLLM.Calls))
	}
}

func TestAppendAssistantWithTools_persistsToolCallsJSON(t *testing.T) {
	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "auto", "agent")
	if err != nil {
		t.Fatal(err)
	}
	r := &Runner{Sessions: store}
	resp := &llm.Response{
		Content: "seg",
		ToolCalls: []llm.ToolCall{{
			ID: "c1", Name: "read_file", Arguments: `{"path":"x"}`,
		}},
	}
	if err := r.appendAssistantWithTools(context.Background(), sess.ID, sess.Model, resp, &subRoundStream{}); err != nil {
		t.Fatal(err)
	}
	msgs, _ := store.ListMessages(context.Background(), sess.ID)
	if len(msgs) != 1 || !strings.Contains(msgs[0].ToolCallsJSON, "read_file") {
		t.Fatalf("msg = %+v", msgs[0])
	}
}
