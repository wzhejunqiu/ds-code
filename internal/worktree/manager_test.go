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
	path, branch, err := mgr.Create(ctx, dir, "run1", CreateOptions{SparsePaths: []string{"/*"}})
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

func TestWorktreeCreate_symlinksLargeDirs(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	gitInDir(t, dir, "init")
	gitInDir(t, dir, "config", "user.email", "t@example.com")
	gitInDir(t, dir, "config", "user.name", "test")
	_ = os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "index.js"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "README"), []byte("hi"), 0o644)
	gitInDir(t, dir, "add", "README")
	gitInDir(t, dir, "commit", "-m", "init")

	base := filepath.Join(dir, "worktrees")
	mgr := NewManager(base)
	ctx := context.Background()
	path, branch, err := mgr.Create(ctx, dir, "run2", CreateOptions{
		SparsePaths: []string{"/*"},
		SymlinkDirs: []string{"node_modules"},
	})
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(path, "node_modules")
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("symlink missing: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("node_modules should be symlink, mode=%v", info.Mode())
	}
	_ = mgr.Remove(ctx, dir, path, branch)
}

func TestValidateSlug(t *testing.T) {
	if err := ValidateSlug("ok-slug_1"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSlug("../bad"); err == nil {
		t.Fatal("expected error for ..")
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
