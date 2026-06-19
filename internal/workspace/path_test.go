package workspace_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	wspkg "github.com/wzhejunqiu/ds-code/internal/workspace"
)

func TestValidateRel_rejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if err := wspkg.ValidateRel(root, "../outside.txt"); err == nil {
		t.Fatal("expected traversal error")
	}
}

func TestValidateRel_allowsDotDotInside(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "util.go")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := wspkg.ValidateRel(root, "pkg/../pkg/util.go"); err != nil {
		t.Fatalf("expected legal .. segment to resolve inside workspace: %v", err)
	}
}

func TestValidateRel_allowsDoubleDotInFilename(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "a..b.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := wspkg.ValidateRel(root, "a..b.txt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRel_allowsDotRoot(t *testing.T) {
	root := t.TempDir()
	if err := wspkg.ValidateRel(root, "."); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveRel_outsideWorkspaceUsesSentinel(t *testing.T) {
	root := t.TempDir()
	_, err := wspkg.ResolveRel(root, "../outside")
	if !errors.Is(err, wspkg.ErrOutsideWorkspace) {
		t.Fatalf("err = %v, want ErrOutsideWorkspace", err)
	}
}

func TestValidateRel_allowsInside(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "ok.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := wspkg.ValidateRel(root, "ok.txt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRel_rejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape")
	if err := os.Symlink(secret, link); err != nil {
		t.Skip("symlink not supported")
	}
	if err := wspkg.ValidateRel(root, "escape"); err == nil {
		t.Fatal("expected symlink escape error")
	}
}
