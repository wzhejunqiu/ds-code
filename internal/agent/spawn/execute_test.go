package spawn_test

import (
	"context"
	"os"
	"path/filepath"
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

	summary, err := spawn.ExecuteRun(context.Background(), cfg, mockLLM, run, def, perm, reg, sub, nil, nil, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "VERDICT: FAIL") {
		t.Fatalf("expected VERDICT in summary, got %q", summary)
	}
}

func TestExecuteRun_atExpand(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		ProjectRoot:    dir,
		ProjectDataDir: dir,
		LLM:            config.LLMConfig{Model: "m", MaxTokens: 4096},
		Context:        config.ContextConfig{ToolResultMaxChars: 50_000},
		Agent:          config.AgentConfig{MaxTurns: 2},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{{
			Content:      "done",
			FinishReason: "stop",
		}},
	}
	sub := subagentstore.NewMemoryStore()
	perm := permission.NewEngine("readonly", dir, false)
	reg := tool.NewRegistry()
	register.ExploreTools(reg, cfg, perm, nil, false)

	run, err := sub.CreateRun(context.Background(), subagentstore.CreateRunParams{
		ParentSessionID: "parent", ParentToolCallID: "tc-at",
		AgentType: "Explore", SpawnKind: subagentstore.SpawnSync,
		Prompt: "check @hello.txt", Model: "m",
	})
	if err != nil {
		t.Fatal(err)
	}
	def, err := spawn.NewRegistry().Resolve("Explore")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := spawn.ExecuteRun(context.Background(), cfg, mockLLM, run, def, perm, reg, sub, nil, nil, 1, nil); err != nil {
		t.Fatal(err)
	}
	if len(mockLLM.Calls) == 0 {
		t.Fatal("expected LLM call")
	}
	var found bool
	for _, call := range mockLLM.Calls {
		for _, m := range call.Messages {
			if m.Role == "user" && strings.Contains(m.Content, "hello world") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected expanded @hello.txt in LLM messages, got %d calls", len(mockLLM.Calls))
	}
}

func TestExecuteRun_atExpand_worktree(t *testing.T) {
	parentRoot := t.TempDir()
	worktreeRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(parentRoot, "parent-only.txt"), []byte("parent secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktreeRoot, "wt-only.txt"), []byte("worktree content"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProjectRoot:    parentRoot,
		ProjectDataDir: parentRoot,
		LLM:            config.LLMConfig{Model: "m", MaxTokens: 4096},
		Context:        config.ContextConfig{ToolResultMaxChars: 50_000},
		Agent:          config.AgentConfig{MaxTurns: 2},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{{
			Content:      "done",
			FinishReason: "stop",
		}},
	}
	sub := subagentstore.NewMemoryStore()
	parentPerm := permission.NewEngine("readonly", parentRoot, false)
	reg := tool.NewRegistry()
	register.ExploreTools(reg, cfg, parentPerm, nil, false)

	run, err := sub.CreateRun(context.Background(), subagentstore.CreateRunParams{
		ParentSessionID: "parent", ParentToolCallID: "tc-wt",
		AgentType: "Explore", SpawnKind: subagentstore.SpawnSync,
		Prompt: "check @wt-only.txt", Model: "m",
	})
	if err != nil {
		t.Fatal(err)
	}
	run.WorktreePath = worktreeRoot
	def, err := spawn.NewRegistry().Resolve("Explore")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := spawn.ExecuteRun(context.Background(), cfg, mockLLM, run, def, parentPerm, reg, sub, nil, nil, 1, nil); err != nil {
		t.Fatal(err)
	}
	if len(mockLLM.Calls) == 0 {
		t.Fatal("expected LLM call")
	}
	var foundWorktree bool
	var foundParent bool
	for _, call := range mockLLM.Calls {
		for _, m := range call.Messages {
			if m.Role != "user" {
				continue
			}
			if strings.Contains(m.Content, "worktree content") {
				foundWorktree = true
			}
			if strings.Contains(m.Content, "parent secret") {
				foundParent = true
			}
		}
	}
	if !foundWorktree {
		t.Fatal("expected @wt-only.txt expanded from worktree workspace")
	}
	if foundParent {
		t.Fatal("parent-only file must not be expanded from worktree workspace")
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
