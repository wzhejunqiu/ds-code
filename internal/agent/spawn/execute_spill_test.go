package spawn_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/datadir"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/mcp/resultstore"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/register"
)

type spillMCPTool struct{}

func (spillMCPTool) Name() string        { return "mcp_spill_tool" }
func (spillMCPTool) Description() string { return "stub spill mcp" }
func (spillMCPTool) Schema() map[string]any {
	return map[string]any{"type": "object"}
}
func (spillMCPTool) Execute(context.Context, json.RawMessage) (string, error) {
	return strings.Repeat("mcp-child-", 200), nil
}
func (spillMCPTool) PermissionLevel() permission.Level { return permission.LevelLow }

func runSpawnWithMCP(t *testing.T, projectRoot, worktreePath string, mcpStore *resultstore.Store) {
	t.Helper()
	testutil.IsolatedHome(t)
	cfg := &config.Config{
		ProjectRoot:    projectRoot,
		ProjectDataDir: projectRoot,
		LLM:            config.LLMConfig{Model: "m", MaxTokens: 4096},
		Context:        config.ContextConfig{ToolResultMaxChars: 50_000},
		Agent:          config.AgentConfig{MaxTurns: 5},
	}
	mockLLM := &mock.Client{
		Responses: []*llm.Response{
			{
				ToolCalls: []llm.ToolCall{{
					ID: "call_child", Name: "mcp_spill_tool", Arguments: `{}`,
				}},
				FinishReason: "tool_calls",
			},
			{Content: "done", FinishReason: "stop"},
		},
	}
	sub := subagentstore.NewMemoryStore()
	perm := permission.NewEngine("readonly", projectRoot, false)
	perm.ProjectRoot = projectRoot
	if worktreePath != "" {
		perm = permission.NewEngine("readonly", worktreePath, false)
		perm.ProjectRoot = projectRoot
	}
	reg := tool.NewRegistry()
	reg.RegisterMCPTool(spillMCPTool{}, "test-server")
	register.ExploreTools(reg, cfg, perm, nil, false)

	run, err := sub.CreateRun(context.Background(), subagentstore.CreateRunParams{
		ParentSessionID: "parent", ParentToolCallID: "tc-spill",
		AgentType: "Explore", SpawnKind: subagentstore.SpawnSync,
		Prompt: "use mcp", Model: "m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if worktreePath != "" {
		run.WorktreePath = worktreePath
	}
	def, err := spawn.NewRegistry().Resolve("Explore")
	if err != nil {
		t.Fatal(err)
	}

	_, err = spawn.ExecuteRun(context.Background(), cfg, mockLLM, run, def, perm, reg, sub, nil, nil, 1, mcpStore)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSpawnExecute_childInheritsMCPResults(t *testing.T) {
	projectRoot := t.TempDir()
	store := &resultstore.Store{ProjectRoot: projectRoot}
	runSpawnWithMCP(t, projectRoot, "", store)

	base := datadir.DefaultMCPResultDir(projectRoot)
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("mcp-result dir missing: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected child MCP spill under shared store")
	}
}

func TestSpawnExecute_worktreeSetsProjectRoot(t *testing.T) {
	projectRoot := t.TempDir()
	worktree := t.TempDir()
	store := &resultstore.Store{ProjectRoot: projectRoot}
	runSpawnWithMCP(t, projectRoot, worktree, store)

	wantDir, err := datadir.ProjectDataDir(projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	base := datadir.DefaultMCPResultDir(projectRoot)
	if !strings.HasPrefix(base, wantDir) {
		t.Fatalf("spill base %q not under project data %q", base, wantDir)
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected spill files under cfg.ProjectRoot project_id")
	}
	// Spill must not land under worktree project_id.
	wtData, _ := datadir.ProjectDataDir(worktree)
	if wtData != "" {
		if _, err := os.Stat(filepath.Join(wtData, "mcp-result")); err == nil {
			t.Fatal("spill should not use worktree project_id")
		}
	}
}

func TestSpawnExecute_readonlyWorktreeSetsProjectRoot(t *testing.T) {
	TestSpawnExecute_worktreeSetsProjectRoot(t)
}
