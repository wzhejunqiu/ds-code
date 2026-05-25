package spawn_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/agent/spawn"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/read_file"
)

func TestExecuteRun_rebindsWorktreePerm(t *testing.T) {
	parentPerm := permission.NewEngine("auto", "/parent", false)
	childPerm := permission.NewEngine("auto", "/worktree", false)

	reg := tool.NewRegistry()
	reg.Register(&read_file.ReadFileTool{
		Cfg:  &config.Config{ProjectRoot: "/parent"},
		Perm: parentPerm,
	})

	childReg := spawn.FilterToolRegistry(reg, spawn.AgentTypeDefinition{Type: "general-purpose"}, false)
	childReg = tool.RebindRegistryPerm(childReg, childPerm)

	tl, ok := childReg.Get("read_file")
	if !ok {
		t.Fatal("missing read_file")
	}
	rf, ok := tl.(*read_file.ReadFileTool)
	if !ok {
		t.Fatal("expected ReadFileTool")
	}
	if rf.Perm.Workspace != "/worktree" {
		t.Fatalf("workspace = %q, want /worktree", rf.Perm.Workspace)
	}
}
