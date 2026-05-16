package context_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/context"
)

func TestCaptureGitSnapshot_inRepo(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "test")
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o644)
	runGit(t, dir, "add", "README.md")

	snap, err := context.CaptureGitSnapshot(dir, 16000)
	if err != nil {
		t.Fatal(err)
	}
	if snap == "" {
		t.Fatal("expected non-empty snapshot")
	}
	if !strings.Contains(snap, "git status") {
		t.Fatalf("missing status header: %q", snap)
	}
}

func TestCaptureGitSnapshot_notRepo(t *testing.T) {
	dir := t.TempDir()
	snap, err := context.CaptureGitSnapshot(dir, 16000)
	if err != nil {
		t.Fatal(err)
	}
	if snap != "" {
		t.Fatalf("expected empty snapshot, got %q", snap)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
