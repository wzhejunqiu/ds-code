package worktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWorktreeCreateRemove(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitInDir(t, dir, "init")
	gitInDir(t, dir, "config", "user.email", "t@example.com")
	gitInDir(t, dir, "config", "user.name", "test")
	_ = os.WriteFile(filepath.Join(dir, "README"), []byte("hi"), 0o644)
	gitInDir(t, dir, "add", "README")
	gitInDir(t, dir, "commit", "-m", "init")

	base := filepath.Join(dir, "worktrees")
	mgr := NewManager(base)
	ctx := context.Background()
	path, branch, err := mgr.Create(ctx, dir, "run1")
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.ValidatePath(path); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Remove(ctx, dir, path, branch); err != nil {
		t.Fatal(err)
	}
}

func gitInDir(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
