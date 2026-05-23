package subagent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/agent/subagent"
	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/llm/mock"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/role"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/session/subagentstore"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/register"
)

func TestRun_persistsSubagentMessagesNotMainSession(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot: dir,
		LLM:         config.LLMConfig{Model: "m", MaxTokens: 4096},
		Context:     config.ContextConfig{ToolResultMaxChars: 50_000},
		Tools: config.ToolsConfig{
			ReadFile: config.ReadFileToolConfig{MaxLines: 500},
			Task:     config.TaskToolConfig{SummaryMaxChars: 8000},
		},
		Agent: config.AgentConfig{MaxTurns: 5},
	}

	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{Content: "summary", FinishReason: "stop"},
		},
	}

	main := session.NewMemoryStore()
	parent, _ := main.CreateSession("m", "max", "enabled", "auto", "agent")
	sub := subagentstore.NewMemoryStore()
	run, err := sub.CreateRun(context.Background(), subagentstore.CreateRunParams{
		ParentSessionID:  parent.ID,
		ParentToolCallID: "call-1",
		Label:            "x",
		Prompt:           "say hi",
		Model:            "m",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = subagent.Run(context.Background(), cfg, mockLLM, "say hi", func(reg *tool.Registry) {
		register.ExploreTools(reg, cfg, permission.NewEngine("readonly", dir, false), nil, false)
	}, sub, run, nil)
	if err != nil {
		t.Fatal(err)
	}

	subMsgs, err := sub.ListMessages(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subMsgs) == 0 {
		t.Fatal("expected subagent messages")
	}

	mainMsgs, err := main.ListMessages(context.Background(), parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range mainMsgs {
		if m.Role == role.Tool && m.ToolName == "read_file" {
			t.Fatalf("main session leaked subagent tool row: %+v", m)
		}
	}

	sess, _ := main.Get(context.Background(), parent.ID)
	if sess.PromptTokensTotal != 0 {
		t.Fatalf("main session totals should not include subagent usage, got %d", sess.PromptTokensTotal)
	}
}

func TestBuildAPIContext_ignoresSubagentTable(t *testing.T) {
	ctx := context.Background()
	main := session.NewMemoryStore()
	sub := subagentstore.NewMemoryStore()
	parent, _ := main.CreateSession("m", "max", "enabled", "auto", "agent")

	_ = main.AppendMessage(ctx, session.Message{
		SessionID: parent.ID, Role: role.User, Content: "user q",
	})
	_ = main.AppendMessage(ctx, session.Message{
		SessionID: parent.ID, Role: role.Assistant,
		ToolCallsJSON: `[{"id":"c1","name":"task","arguments":"{}"}]`,
	})
	_ = main.AppendMessage(ctx, session.Message{
		SessionID: parent.ID, Role: role.Tool, ToolCallID: "c1", ToolName: "task", Content: "task summary",
	})

	run, _ := sub.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID: parent.ID, ParentToolCallID: "c1", Prompt: "internal",
	})
	_ = sub.AppendMessage(ctx, subagentstore.Message{
		RunID: run.ID, Role: role.User, Content: "secret subagent prompt",
	})
	_ = sub.AppendMessage(ctx, subagentstore.Message{
		RunID: run.ID, Role: role.Tool, ToolName: "read_file", Content: "file body",
	})

	ctxSvc := &ctxpkg.Service{
		Cfg:      &config.Config{Context: config.ContextConfig{WindowTokens: 128000}},
		Store:    main,
		Subagent: sub,
		Tools:    tool.NewRegistry(),
	}
	view, err := ctxSvc.BuildAPIContext(ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range view.Messages {
		if strings.Contains(m.Content, "secret subagent") {
			t.Fatalf("leaked subagent content into API context: %q", m.Content)
		}
		if strings.Contains(m.Content, "file body") {
			t.Fatalf("leaked subagent tool into API context: %q", m.Content)
		}
	}
}
