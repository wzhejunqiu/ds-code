package spawn_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

func TestFromSkill_requiresParentContext(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".ds-code", "skills", "test-fork")
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := "---\ncontext: fork\n---\nDo the fork task.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := testConfig()
	cfg.ProjectRoot = dir
	cfg.Tools.Agent.ForkEnabled = true

	var store subagentstore.Store = subagentstore.NewMemoryStore()
	svc := spawn.NewService(cfg, testPerm(), testRegistry(), nil, store)
	svc.ParentContext = nil

	inv := agent.ToolInvocation{SessionID: "sess-1", ToolCallID: "skill:test-fork"}
	_, err := svc.FromSkill(context.Background(), inv, "test-fork", true)
	if err == nil {
		t.Fatal("expected error without parent context")
	}
}

func TestFromSkill_rejectsNonForkSkill(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".ds-code", "skills", "plain")
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Plain skill\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{ProjectRoot: dir, Tools: config.ToolsConfig{Agent: config.AgentToolConfig{ForkEnabled: true}}}
	svc := spawn.NewService(cfg, testPerm(), testRegistry(), nil, subagentstore.NewMemoryStore())

	inv := agent.ToolInvocation{SessionID: "sess-1", ToolCallID: "skill:plain"}
	_, err := svc.FromSkill(context.Background(), inv, "plain", true)
	if err == nil {
		t.Fatal("expected ErrSkillNotFork")
	}
}
