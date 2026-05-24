package context

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

func TestBuildAPIContext_includesRulesAndSkills(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".ds-code", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "test.md"), []byte("Always run tests."), 0o644); err != nil {
		t.Fatal(err)
	}

	store := session.NewMemoryStore()
	sess, err := store.NewSession("m", "max", "enabled", "readonly", "agent")
	if err != nil {
		t.Fatal(err)
	}

	rules, err := LoadRules(dir)
	if err != nil || rules == "" {
		t.Fatalf("LoadRules: %v rules=%q", err, rules)
	}

	svc := &Service{
		Cfg:        &config.Config{ProjectRoot: dir, Context: config.ContextConfig{WindowTokens: 1_000_000}},
		Store:      store,
		Tools:      tool.NewRegistry(),
		Rules:      rules,
		SkillsText: "## Skill\nUse TDD.",
	}

	view, err := svc.BuildAPIContext(context.Background(), sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	merged := view.MergedSystem()
	if !contains(merged, "Always run tests") {
		t.Fatalf("rules not in system: %s", merged)
	}
	if !contains(merged, "Use TDD") {
		t.Fatalf("skills not in system: %s", merged)
	}
	if !contains(merged, dir) {
		t.Fatalf("project_root not in runtime env: %s", merged)
	}
	if !contains(merged, "## 运行环境") {
		t.Fatalf("runtime env section missing: %s", merged)
	}
	if !contains(merged, runtime.GOOS) {
		t.Fatalf("OS info not in system (GOOS=%s): %s", runtime.GOOS, merged)
	}
}
