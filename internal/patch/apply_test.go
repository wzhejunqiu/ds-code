package patch_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/patch"
)

func TestPaths(t *testing.T) {
	text := `*** Begin Patch
*** Add File: a.go
+x
*** Delete File: old.go
*** End Patch`
	paths, err := patch.Paths(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("paths = %v", paths)
	}
}

func TestApply_updateAndMoveRollback(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolve := func(rel string) (string, error) {
		return filepath.Join(dir, rel), nil
	}

	patchText := `*** Begin Patch
*** Update File: src.txt
*** Move to: dst.txt
@@ line1
-line1
+line1-moved
*** End Patch`

	summary, err := patch.Apply(dir, patchText, resolve, patch.ApplyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "src.txt -> dst.txt") {
		t.Fatalf("summary = %q", summary)
	}
	if _, err := os.Stat(src); err == nil {
		t.Fatal("source should be removed after move")
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "line1-moved\nline2\n" {
		t.Fatalf("dst content = %q", got)
	}
}

func TestApply_findContextPrefersExactLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.go")
	content := "func foo() {}\nfunc food() {}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	patchText := `*** Begin Patch
*** Update File: f.go
@@ func foo() {}
-func foo() {}
+func foo() { /* edited */ }
*** End Patch`

	resolve := func(rel string) (string, error) {
		return filepath.Join(dir, rel), nil
	}
	_, err := patch.Apply(dir, patchText, resolve, patch.ApplyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "/* edited */") {
		t.Fatalf("patch did not apply at exact context: %q", got)
	}
	if strings.Contains(string(got), "food() { /* edited */") {
		t.Fatalf("patch matched wrong line: %q", got)
	}
}
