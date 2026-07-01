package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
)

func TestAppendWebAllowlist_createsFile(t *testing.T) {
	root := t.TempDir()
	if err := config.AppendWebAllowlist(root, "example.com"); err != nil {
		t.Fatal(err)
	}
	path := config.ProjectConfigPath(root)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "example.com") {
		t.Fatalf("content = %s", b)
	}
}

func TestAppendWebAllowlist_preservesCommentsAndFields(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Dir(config.ProjectConfigPath(root))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	initial := "# project config\npermission:\n  mode: ask\nweb:\n  fetch_enabled: true\n  allowlist:\n    - pkg.go.dev\n"
	path := config.ProjectConfigPath(root)
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.AppendWebAllowlist(root, "example.com"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := string(b)
	if !strings.Contains(out, "# project config") {
		t.Fatal("comment lost")
	}
	if !strings.Contains(out, "mode: ask") {
		t.Fatal("permission.mode changed")
	}
	if !strings.Contains(out, "pkg.go.dev") {
		t.Fatal("existing allowlist entry lost")
	}
	if !strings.Contains(out, "example.com") {
		t.Fatal("new host missing")
	}
}

func TestAppendWebAllowlist_dedupConcurrent(t *testing.T) {
	root := t.TempDir()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = config.AppendWebAllowlist(root, "dup.test")
		}()
	}
	wg.Wait()
	b, err := os.ReadFile(config.ProjectConfigPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(b), "dup.test") != 1 {
		t.Fatalf("expected one dup.test entry, got:\n%s", b)
	}
}
