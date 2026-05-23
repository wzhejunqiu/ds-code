package context_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/context"
)

func TestCaptureGitSnapshot_inRepo(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "test")
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o644)
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "initial")

	snap, err := context.CaptureGitSnapshot(dir, 16000)
	if err != nil {
		t.Fatal(err)
	}
	if snap == "" {
		t.Fatal("expected non-empty snapshot")
	}
	for _, want := range []string{"### 当前分支", "### 默认分支", "### Git 用户", "### git status", "### 最近提交"} {
		if !strings.Contains(snap, want) {
			t.Fatalf("missing section %q in snapshot:\n%s", want, snap)
		}
	}
	if !strings.Contains(snap, "main") {
		t.Fatalf("expected branch/main in snapshot: %q", snap)
	}
	if !strings.Contains(snap, "test <test@test.com>") {
		t.Fatalf("expected git user in snapshot: %q", snap)
	}
	if !strings.Contains(snap, "initial") {
		t.Fatalf("expected commit message in snapshot: %q", snap)
	}
}

func TestCaptureGitSnapshot_statusTruncated(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "test")
	for i := 0; i < 200; i++ {
		name := fmt.Sprintf("file-%03d.txt", i)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	snap, err := context.CaptureGitSnapshot(dir, 16000)
	if err != nil {
		t.Fatal(err)
	}
	statusIdx := strings.Index(snap, "### git status\n")
	if statusIdx < 0 {
		t.Fatal("missing git status section")
	}
	rest := snap[statusIdx+len("### git status\n"):]
	nextSection := strings.Index(rest, "\n### ")
	if nextSection < 0 {
		t.Fatal("missing next section after git status")
	}
	statusBody := rest[:nextSection]
	if !strings.Contains(statusBody, "[status 已截断]") {
		t.Fatalf("expected status truncation marker, got %d chars", len(statusBody))
	}
	// 截断后约为 2048 + 标记长度
	if len(statusBody) < 2048 {
		t.Fatalf("expected long status body, got %d chars", len(statusBody))
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
