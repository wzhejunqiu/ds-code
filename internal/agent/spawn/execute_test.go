package spawn_test

import (
	"context"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/register"
)

func TestExecuteRun_verificationVerdict(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot:    dir,
		ProjectDataDir: dir,
		LLM:            config.LLMConfig{Model: "m", MaxTokens: 4096},
		Context:        config.ContextConfig{ToolResultMaxChars: 50_000},
		Agent:          config.AgentConfig{MaxTurns: 5},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{{
			Content:      "Checked files.\nVERDICT: FAIL",
			FinishReason: "stop",
		}},
	}
	sub := subagentstore.NewMemoryStore()
	perm := permission.NewEngine("readonly", dir, false)
	reg := tool.NewRegistry()
	register.ExploreTools(reg, cfg, perm, nil, false)

	run, err := sub.CreateRun(context.Background(), subagentstore.CreateRunParams{
		ParentSessionID: "parent", ParentToolCallID: "tc-v",
		AgentType: "verification", SpawnKind: subagentstore.SpawnSync,
		Prompt: "verify", Model: "m",
	})
	if err != nil {
		t.Fatal(err)
	}
	def, err := spawn.NewRegistry().Resolve("verification")
	if err != nil {
		t.Fatal(err)
	}

	summary, err := spawn.ExecuteRun(context.Background(), cfg, mockLLM, run, def, perm, reg, sub, nil, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "VERDICT: FAIL") {
		t.Fatalf("expected VERDICT in summary, got %q", summary)
	}
}

func TestExploreShell_gitStatusAllowed_rmDenied(t *testing.T) {
	dir := t.TempDir()
	perm := permission.NewEngine("readonly", dir, false)
	if err := perm.Check("shell", map[string]any{"command": "git status"}); err != nil {
		t.Fatalf("git status should be allowed: %v", err)
	}
	if err := perm.Check("shell", map[string]any{"command": "rm -rf /"}); err == nil {
		t.Fatal("rm -rf should be denied in readonly")
	}
}
