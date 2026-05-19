package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	wspkg "github.com/hejunqiu/ds-code/internal/workspace"
)

func TestValidateRel_rejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if err := wspkg.ValidateRel(root, "../outside.txt"); err == nil {
		t.Fatal("expected traversal error")
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
