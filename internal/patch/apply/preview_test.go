package apply_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/patch/apply"
)

func TestPreview_addFile(t *testing.T) {
	dir := t.TempDir()
	resolve := func(rel string) (string, error) {
		return apply.ResolveWorkspacePath(dir, rel)
	}
	patch := `*** Begin Patch
*** Add File: new.txt
+hello
+world
*** End Patch`
	previews, err := apply.Preview(dir, patch, resolve)
	if err != nil {
		t.Fatal(err)
	}
	if len(previews) != 1 {
		t.Fatalf("previews = %d", len(previews))
	}
	if previews[0].Original != "" {
		t.Fatalf("original = %q", previews[0].Original)
	}
	if previews[0].Modified != "hello\nworld\n" {
		t.Fatalf("modified = %q", previews[0].Modified)
	}
}

func TestPreview_updateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolve := func(rel string) (string, error) {
		return apply.ResolveWorkspacePath(dir, rel)
	}
	patch := `*** Begin Patch
*** Update File: a.txt
@@ line1
-line2
+line2 changed
*** End Patch`
	previews, err := apply.Preview(dir, patch, resolve)
	if err != nil {
		t.Fatal(err)
	}
	if len(previews) != 1 {
		t.Fatalf("previews = %d", len(previews))
	}
	if previews[0].Modified != "line1\nline2 changed\n" {
		t.Fatalf("modified = %q", previews[0].Modified)
	}
}
