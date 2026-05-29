package spawn_test

import (
	"context"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/role"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/register"
)

func TestExecuteRun_persistsAgentMessagesNotMainSession(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot: dir,
		LLM:         config.LLMConfig{Model: "m", MaxTokens: 4096},
		Context:     config.ContextConfig{ToolResultMaxChars: 50_000},
		Tools: config.ToolsConfig{
			ReadFile: config.ReadFileToolConfig{MaxLines: 500},
			Agent:    config.AgentToolConfig{SummaryMaxChars: 8000},
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
		AgentType:        "Explore",
		SpawnKind:        subagentstore.SpawnSync,
	})
	if err != nil {
		t.Fatal(err)
	}

	perm := permission.NewEngine("readonly", dir, false)
	reg := tool.NewRegistry()
	register.ExploreTools(reg, cfg, perm, nil, false)

	def, _ := spawn.NewRegistry().Resolve("Explore")
	_, err = spawn.ExecuteRun(context.Background(), cfg, mockLLM, run, def, perm, reg, sub, nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	subMsgs, err := sub.ListMessages(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subMsgs) == 0 {
		t.Fatal("expected agent messages")
	}

	mainMsgs, err := main.ListMessages(context.Background(), parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range mainMsgs {
		if m.Role == role.Tool && m.ToolName == "read_file" {
			t.Fatalf("main session leaked agent tool row: %+v", m)
		}
	}

	sess, _ := main.Get(context.Background(), parent.ID)
	if sess.PromptTokensTotal != 0 {
		t.Fatalf("main session totals should not include agent usage, got %d", sess.PromptTokensTotal)
	}
}

func TestBuildAPIContext_ignoresAgentTable(t *testing.T) {
	ctx := context.Background()
	main := session.NewMemoryStore()
	sub := subagentstore.NewMemoryStore()
	parent, _ := main.CreateSession("m", "max", "enabled", "auto", "agent")

	_ = main.AppendMessage(ctx, session.Message{
		SessionID: parent.ID, Role: role.User, Content: "user q",
	})
	_ = main.AppendMessage(ctx, session.Message{
		SessionID: parent.ID, Role: role.Assistant,
		ToolCallsJSON: `[{"id":"c1","name":"agent","arguments":"{}"}]`,
	})
	_ = main.AppendMessage(ctx, session.Message{
		SessionID: parent.ID, Role: role.Tool, ToolCallID: "c1", ToolName: "agent", Content: "agent summary",
	})

	run, _ := sub.CreateRun(ctx, subagentstore.CreateRunParams{
		ParentSessionID: parent.ID, ParentToolCallID: "c1", Prompt: "internal",
	})
	_ = sub.AppendMessage(ctx, subagentstore.Message{
		RunID: run.ID, Role: role.User, Content: "secret agent prompt",
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
		if strings.Contains(m.Content, "secret agent") {
			t.Fatalf("leaked agent content into API context: %q", m.Content)
		}
		if strings.Contains(m.Content, "file body") {
			t.Fatalf("leaked agent tool into API context: %q", m.Content)
		}
	}
}
