package context

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/tool"
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
}
