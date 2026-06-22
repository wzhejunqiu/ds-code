package readgate_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool/readgate"
)

func TestGate_CheckApplyPatch_sameBatch(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.go", "package a\n")

	canon := mustCanon(t, root, "a.go")
	gate := readgate.NewGate(root, map[string]struct{}{canon: {}}, map[string]struct{}{canon: {}}, nil)

	err := gate.CheckApplyPatch([]string{"a.go"}, readgateSameBatchFmt, readgateMustReadFmt)
	if err == nil || !strings.Contains(err.Error(), "same-batch") {
		t.Fatalf("expected same-batch error, got %v", err)
	}
}

func TestGate_CheckApplyPatch_missing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.go", "package a\n")

	gate := readgate.NewGate(root, map[string]struct{}{}, map[string]struct{}{}, nil)
	err := gate.CheckApplyPatch([]string{"a.go"}, readgateSameBatchFmt, readgateMustReadFmt)
	if err == nil || !strings.Contains(err.Error(), "must-read") {
		t.Fatalf("expected must-read error, got %v", err)
	}
}

func TestGate_CheckApplyPatch_ok(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.go", "package a\n")

	canon := mustCanon(t, root, "a.go")
	gate := readgate.NewGate(root, map[string]struct{}{canon: {}}, map[string]struct{}{}, nil)
	if err := gate.CheckApplyPatch([]string{"a.go"}, readgateSameBatchFmt, readgateMustReadFmt); err != nil {
		t.Fatal(err)
	}
}

func TestGate_MarkPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.go", "package a\n")

	var marked []string
	gate := readgate.NewGate(root, map[string]struct{}{}, map[string]struct{}{}, func(c string) {
		marked = append(marked, c)
	})
	if err := gate.MarkPath("a.go"); err != nil {
		t.Fatal(err)
	}
	if len(marked) != 1 {
		t.Fatalf("marked = %v", marked)
	}
}

func TestFromContext_nil(t *testing.T) {
	if _, ok := readgate.FromContext(context.Background()); ok {
		t.Fatal("expected no gate")
	}
}

const (
	readgateSameBatchFmt = "same-batch: %s"
	readgateMustReadFmt  = "must-read: %s"
)

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustCanon(t *testing.T, workspace, rel string) string {
	t.Helper()
	c, err := readgate.CanonicalPath(workspace, rel)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
