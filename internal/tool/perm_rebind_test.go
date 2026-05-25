package tool

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/permission"
)

type permToolStub struct {
	name string
	perm *permission.Engine
}

func (p *permToolStub) Name() string        { return p.name }
func (p *permToolStub) Description() string { return "stub" }
func (p *permToolStub) Schema() map[string]any {
	return ObjectSchema(map[string]any{}, nil, false)
}
func (p *permToolStub) Execute(context.Context, json.RawMessage) (string, error) { return "", nil }
func (p *permToolStub) PermissionLevel() permission.Level                        { return permission.LevelLow }

func (p *permToolStub) WithPerm(perm *permission.Engine) Tool {
	cp := *p
	cp.perm = perm
	return &cp
}

func TestRebindRegistryPerm_rebindsPermBound(t *testing.T) {
	parent := permission.NewEngine("auto", "/parent", false)
	child := permission.NewEngine("auto", "/worktree", false)

	reg := NewRegistry()
	reg.Register(&permToolStub{name: "read_file", perm: parent})

	out := RebindRegistryPerm(reg, child)
	tl, ok := out.Get("read_file")
	if !ok {
		t.Fatal("missing tool")
	}
	pt, ok := tl.(*permToolStub)
	if !ok {
		t.Fatal("expected permToolStub")
	}
	if pt.perm.Workspace != "/worktree" {
		t.Fatalf("workspace = %q, want /worktree", pt.perm.Workspace)
	}
}

func TestRebindRegistryPerm_deferred(t *testing.T) {
	parent := permission.NewEngine("auto", "/parent", false)
	child := permission.NewEngine("auto", "/worktree", false)

	reg := NewRegistry()
	reg.Register(WrapDeferred(&permToolStub{name: "shell", perm: parent}))

	out := RebindRegistryPerm(reg, child)
	tl, ok := out.Get("shell")
	if !ok {
		t.Fatal("missing tool")
	}
	dw, ok := tl.(*deferredWrapper)
	if !ok {
		t.Fatal("expected deferredWrapper")
	}
	inner, ok := dw.inner.(*permToolStub)
	if !ok {
		t.Fatal("expected inner permToolStub")
	}
	if inner.perm.Workspace != "/worktree" {
		t.Fatalf("workspace = %q, want /worktree", inner.perm.Workspace)
	}
}
