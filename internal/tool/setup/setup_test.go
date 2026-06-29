package setup_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/runmode"
	"github.com/wzhejunqiu/ds-code/internal/tool/setup"
)

func TestBuildRegistry_planOmitsWriteTools(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{ProjectRoot: dir}
	perm := permission.NewEngine("ask", dir, false)
	deps := setup.Deps{Cfg: cfg, Perm: perm, Strict: false, LLM: &mock.Client{}}

	plan := setup.BuildRegistry(runmode.Plan, deps)
	agent := setup.BuildRegistry(runmode.Agent, deps)

	for _, name := range []string{"read_file", "grep", "glob"} {
		if _, ok := plan.Get(name); !ok {
			t.Fatalf("plan missing %s", name)
		}
	}
	for _, name := range []string{"bash", "apply_patch", "write_file", "agent"} {
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
	reg := setup.BuildRegistry(runmode.Plan, setup.Deps{Cfg: cfg, Perm: perm, Strict: false, LLM: &mock.Client{}})
	if _, ok := reg.Get("web_fetch"); !ok {
		t.Fatal("plan with fetch_enabled and LLM should register web_fetch")
	}
}

func TestBuildRegistry_planWithoutLLMOmitsWebFetch(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot: dir,
		Web:         config.WebConfig{FetchEnabled: true, Allowlist: []string{"example.com"}},
		LSP:         config.LSPConfig{Enabled: false},
	}
	perm := permission.NewEngine("readonly", dir, false)
	reg := setup.BuildRegistry(runmode.Plan, setup.Deps{Cfg: cfg, Perm: perm, Strict: false})
	if _, ok := reg.Get("web_fetch"); ok {
		t.Fatal("web_fetch should not register without LLM")
	}
}
