package patch_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/patch"
	patchapply "github.com/wzhejunqiu/ds-code/internal/patch/apply"
)

func TestParse_addAndUpdate(t *testing.T) {
	text := `*** Begin Patch
*** Add File: new.txt
+hello
*** Update File: main.go
@@ func main
-print("hi")
+print("bye")
*** End Patch`
	changes, err := patch.Parse(text, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("changes = %d", len(changes))
	}
	if changes[0].Kind != patch.ChangeAdd || changes[0].Path != "new.txt" {
		t.Fatalf("add: %+v", changes[0])
	}
	if changes[1].Kind != patch.ChangeUpdate || len(changes[1].Chunks) != 1 {
		t.Fatalf("update: %+v", changes[1])
	}
}

func TestApply_roundTrip(t *testing.T) {
	dir := t.TempDir()
	orig := "package main\n\nfunc main() {\n\tprint(\"hi\")\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	patchText := `*** Begin Patch
*** Update File: main.go
@@ func main
-	print("hi")
+	print("bye")
*** End Patch`
	resolve := func(rel string) (string, error) {
		return filepath.Join(dir, rel), nil
	}
	_, err := patchapply.Apply(dir, patchText, resolve, patchapply.Options{MaxChangedLines: 100})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `print("bye")`) {
		t.Fatalf("content = %q", b)
	}
}

func TestParse_heredoc(t *testing.T) {
	text := `<<EOF
*** Begin Patch
*** Add File: heredoc.txt
+line
*** End Patch
EOF`
	changes, err := patch.Parse(text, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Path != "heredoc.txt" {
		t.Fatalf("changes = %+v", changes)
	}
}

func TestParse_delete(t *testing.T) {
	text := `*** Begin Patch
*** Delete File: old.go
*** End Patch`
	changes, err := patch.Parse(text, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Kind != patch.ChangeDelete || changes[0].Path != "old.go" {
		t.Fatalf("changes = %+v", changes)
	}
}

func TestParse_move(t *testing.T) {
	text := `*** Begin Patch
*** Update File: src.go
*** Move to: dst.go
@@ func
-old
+new
*** End Patch`
	changes, err := patch.Parse(text, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 {
		t.Fatalf("changes = %d", len(changes))
	}
	if changes[0].MoveTo != "dst.go" || len(changes[0].Chunks) != 1 {
		t.Fatalf("update = %+v", changes[0])
	}
}

func TestParse_emptyPatch(t *testing.T) {
	_, err := patch.Parse("", "")
	if err == nil {
		t.Fatal("expected error for empty patch")
	}
}

func TestParse_invalidChangeLine(t *testing.T) {
	text := `*** Begin Patch
*** Update File: f.go
@@
?bad
*** End Patch`
	_, err := patch.Parse(text, "")
	if err == nil {
		t.Fatal("expected error for invalid change line")
	}
}

func TestParse_addLineWithoutPlus(t *testing.T) {
	text := `*** Begin Patch
*** Add File: x.txt
not prefixed
*** End Patch`
	_, err := patch.Parse(text, "")
	if err == nil {
		t.Fatal("expected error for line without '+'")
	}
}

func TestParse_updateUnexpectedLine(t *testing.T) {
	text := `*** Begin Patch
*** Update File: f.go
garbage
*** End Patch`
	_, err := patch.Parse(text, "")
	if err == nil {
		t.Fatal("expected error for unexpected update line")
	}
}

func TestParse_duplicatePath(t *testing.T) {
	text := `*** Begin Patch
*** Add File: dup.txt
+a
*** Update File: dup.txt
@@
-x
+y
*** End Patch`
	_, err := patch.Parse(text, "")
	if err == nil {
		t.Fatal("expected duplicate path error")
	}
}

func TestParse_heredocCustomDelimiter(t *testing.T) {
	text := `<<'PATCH'
*** Begin Patch
*** Add File: x.txt
+line
*** End Patch
PATCH`
	changes, err := patch.Parse(text, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Path != "x.txt" {
		t.Fatalf("changes = %+v", changes)
	}
}

func TestCountChangedLines_contextNotDoubleCounted(t *testing.T) {
	text := `*** Begin Patch
*** Update File: f.txt
@@ ctx
 context
-old
+new
*** End Patch`
	n, err := patch.CountChangedLines(text, "")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("changed lines = %d, want 2", n)
	}
}

func TestApply_delete(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "remove.me")
	if err := os.WriteFile(target, []byte("gone"), 0o644); err != nil {
		t.Fatal(err)
	}
	patchText := `*** Begin Patch
*** Delete File: remove.me
*** End Patch`
	resolve := func(rel string) (string, error) {
		return filepath.Join(dir, rel), nil
	}
	summary, err := patchapply.Apply(dir, patchText, resolve, patchapply.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "delete remove.me") {
		t.Fatalf("summary = %q", summary)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("file should be deleted")
	}
}

func TestApply_move(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.go")
	dst := filepath.Join(dir, "dst.go")
	if err := os.WriteFile(src, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patchText := `*** Begin Patch
*** Update File: src.go
*** Move to: dst.go
@@ func main
-func main() {}
+func main() { println("moved") }
*** End Patch`
	resolve := func(rel string) (string, error) {
		return filepath.Join(dir, rel), nil
	}
	_, err := patchapply.Apply(dir, patchText, resolve, patchapply.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatal("source should be removed after move")
	}
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `println("moved")`) {
		t.Fatalf("dst content = %q", b)
	}
}

func TestApply_maxChangedLines(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patchText := `*** Begin Patch
*** Update File: f.txt
@@
-a
+x
-b
+y
-c
+z
*** End Patch`
	resolve := func(rel string) (string, error) {
		return filepath.Join(dir, rel), nil
	}
	_, err := patchapply.Apply(dir, patchText, resolve, patchapply.Options{MaxChangedLines: 2})
	if err == nil {
		t.Fatal("expected max changed lines error")
	}
}

func TestApply_contextNotFound(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patchText := `*** Begin Patch
*** Update File: f.txt
@@ missingContext
+inserted
*** End Patch`
	resolve := func(rel string) (string, error) {
		return filepath.Join(dir, rel), nil
	}
	_, err := patchapply.Apply(dir, patchText, resolve, patchapply.Options{})
	if err == nil {
		t.Fatal("expected context not found error")
	}
}

func TestApply_addExistingFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "exists.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	patchText := `*** Begin Patch
*** Add File: exists.txt
+new
*** End Patch`
	resolve := func(rel string) (string, error) {
		return filepath.Join(dir, rel), nil
	}
	_, err := patchapply.Apply(dir, patchText, resolve, patchapply.Options{})
	if err == nil {
		t.Fatal("expected add file exists error")
	}
}

func TestApply_rollbackOnFailure(t *testing.T) {
	dir := t.TempDir()
	orig := "line1\nline2\n"
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	bad := `*** Begin Patch
*** Update File: f.txt
@@
-nope
+yes
*** End Patch`
	resolve := func(rel string) (string, error) {
		return filepath.Join(dir, rel), nil
	}
	_, err := patchapply.Apply(dir, bad, resolve, patchapply.Options{})
	if err == nil {
		t.Fatal("expected apply error")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "f.txt"))
	if string(b) != orig {
		t.Fatalf("file not rolled back: %q", b)
	}
}
