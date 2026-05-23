package builtin_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool/builtin"
)

func TestMakeFileCandidate_workspaceRelFromSubdir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "internal", "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(sub, "a.go")
	if err := os.WriteFile(f, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	perm := permission.NewEngine("readonly", dir, false)
	c := builtin.MakeFileCandidate(perm, f, builtin.FileFilter{})
	if c == nil {
		t.Fatal("expected candidate")
	}
	want := "internal/pkg/a.go"
	if c.Rel != want {
		t.Fatalf("rel = %q, want %q", c.Rel, want)
	}
}

func TestMakeFileCandidate_maxFileBytes(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "big.txt")
	content := make([]byte, 100)
	for i := range content {
		content[i] = 'x'
	}
	if err := os.WriteFile(f, content, 0o644); err != nil {
		t.Fatal(err)
	}
	perm := permission.NewEngine("readonly", dir, false)

	if c := builtin.MakeFileCandidate(perm, f, builtin.FileFilter{MaxFileBytes: 50}); c != nil {
		t.Fatal("expected nil for oversized file")
	}
	if c := builtin.MakeFileCandidate(perm, f, builtin.FileFilter{MaxFileBytes: 200}); c == nil {
		t.Fatal("expected candidate under size limit")
	}
}

func TestMakeFileCandidate_skipsBinary(t *testing.T) {
	dir := t.TempDir()
	png := filepath.Join(dir, "x.png")
	data := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	data = append(data, make([]byte, 32)...)
	if err := os.WriteFile(png, data, 0o644); err != nil {
		t.Fatal(err)
	}
	perm := permission.NewEngine("readonly", dir, false)
	if c := builtin.MakeFileCandidate(perm, png, builtin.FileFilter{}); c != nil {
		t.Fatal("expected nil for binary png")
	}
}

func TestValidateGlobMatches_rejectsOutsideWorkspace(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "escape")
	if err := os.Symlink(secret, link); err != nil {
		t.Skip("symlink not supported")
	}

	perm := permission.NewEngine("readonly", dir, false)
	err := builtin.ValidateGlobMatches(perm, []string{link}, "escape")
	if err == nil {
		t.Fatal("expected error for symlink outside workspace")
	}
}

func TestCollectGlobPattern_preservesMatchOrderNotModTime(t *testing.T) {
	dir := t.TempDir()
	aFirst := filepath.Join(dir, "a_first.go")
	zLast := filepath.Join(dir, "z_last.go")
	if err := os.WriteFile(aFirst, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(zLast, []byte("y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	_ = os.Chtimes(aFirst, oldTime, oldTime)
	_ = os.Chtimes(zLast, newTime, newTime)

	perm := permission.NewEngine("readonly", dir, false)
	out, err := builtin.CollectGlobPattern(context.Background(), perm, dir, "*.go", builtin.FileFilter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(out))
	}
	// filepath.Glob order is lexical; mtime desc would put z_last first.
	if out[0].Rel != "a_first.go" || out[1].Rel != "z_last.go" {
		t.Fatalf("CollectGlobPattern should not sort by ModTime: got %q, %q", out[0].Rel, out[1].Rel)
	}
	sorted := append([]builtin.FileCandidate(nil), out...)
	builtin.SortByModTimeDesc(sorted,
		func(c builtin.FileCandidate) time.Time { return c.ModTime },
		func(c builtin.FileCandidate) string { return c.Rel },
	)
	if sorted[0].Rel != "z_last.go" {
		t.Fatalf("caller sort should order by mtime: got %q first", sorted[0].Rel)
	}
}
