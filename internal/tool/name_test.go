package tool_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/llm/mock"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/setup"
)

func TestName_builtinsMatchRegistry(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{ProjectRoot: dir}
	perm := permission.NewEngine("ask", dir, false)
	reg := setup.BuildRegistry("agent", setup.Deps{Cfg: cfg, Perm: perm, Strict: false, LLM: &mock.Client{}})

	for _, name := range []tool.Name{
		tool.NameReadFile,
		tool.NameWriteFile,
		tool.NameApplyPatch,
		tool.NameShell,
		tool.NameGlob,
		tool.NameGrep,
		tool.NameAgent,
		tool.NameToolSearch,
	} {
		if _, ok := reg.Get(name.String()); !ok {
			t.Errorf("registry missing builtin tool %q", name)
		}
	}
}
