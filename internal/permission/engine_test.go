package permission_test

import (
	"testing"

	"github.com/hejunqiu/ds-code/internal/permission"
)

func TestEngine_askNonInteractive_deniesShell(t *testing.T) {
	e := permission.NewEngine("ask", t.TempDir(), false)
	err := e.Check("shell", map[string]any{"command": "echo hi"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err != permission.ErrNeedTTY {
		t.Fatalf("err = %v", err)
	}
}

func TestEngine_readonly_deniesShell(t *testing.T) {
	e := permission.NewEngine("readonly", t.TempDir(), true)
	err := e.Check("shell", map[string]any{"command": "echo hi"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEngine_resolvePath_blocksTraversal(t *testing.T) {
	root := t.TempDir()
	e := permission.NewEngine("auto", root, true)
	_, err := e.ResolvePath("../outside")
	if err == nil {
		t.Fatal("expected traversal error")
	}
}
