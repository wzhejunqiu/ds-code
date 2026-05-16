package setup_test

import (
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/llm/mock"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool/setup"
)

func TestBuildRegistry_planOmitsWriteTools(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{ProjectRoot: dir}
	perm := permission.NewEngine("ask", dir, false)
	deps := setup.Deps{Cfg: cfg, Perm: perm, Strict: false, LLM: &mock.Client{}}

	plan := setup.BuildRegistry("plan", deps)
	agent := setup.BuildRegistry("agent", deps)

	for _, name := range []string{"read_file", "grep", "glob", "list_dir"} {
		if _, ok := plan.Get(name); !ok {
			t.Fatalf("plan missing %s", name)
		}
	}
	for _, name := range []string{"shell", "apply_patch", "write_file", "task"} {
		if _, ok := plan.Get(name); ok {
			t.Fatalf("plan should not have %s", name)
		}
		if _, ok := agent.Get(name); !ok {
			t.Fatalf("agent missing %s", name)
		}
	}
}

func TestBuildRegistry_planWithWebFetch(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot: dir,
		Web:         config.WebConfig{FetchEnabled: true, Allowlist: []string{"example.com"}},
		LSP:         config.LSPConfig{Enabled: false},
	}
	perm := permission.NewEngine("readonly", dir, false)
	reg := setup.BuildRegistry("plan", setup.Deps{Cfg: cfg, Perm: perm, Strict: false})
	if _, ok := reg.Get("web_fetch"); !ok {
		t.Fatal("plan with fetch_enabled should register web_fetch")
	}
}
