package permission_test

import (
	"os"
	"path/filepath"
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

func TestEngine_resolvePath_allowsAbsoluteInsideWorkspace(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "internal", "ui")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(inner, "resume_picker.go")
	if err := os.WriteFile(file, []byte("package ui\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := permission.NewEngine("auto", root, true)
	got, err := e.ResolvePath(file)
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	want, err := filepath.EvalSymlinks(file)
	if err != nil {
		want = file
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEngine_resolvePath_rejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape-link")
	if err := os.Symlink(secret, link); err != nil {
		t.Skip("symlink not supported:", err)
	}

	e := permission.NewEngine("auto", root, true)
	_, err := e.ResolvePath("escape-link")
	if err == nil {
		t.Fatal("expected symlink escape to be denied")
	}
}

func TestEngine_resolvePath_rejectsAbsoluteOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	e := permission.NewEngine("auto", root, true)
	outside := filepath.Join(t.TempDir(), "secret.txt")
	_, err := e.ResolvePath(outside)
	if err == nil {
		t.Fatal("expected outside workspace error")
	}
}
