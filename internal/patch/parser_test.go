package patch_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/patch"
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
	changes, err := patch.Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("changes = %d", len(changes))
	}
	if changes[0].Kind != "add" || changes[0].Path != "new.txt" {
		t.Fatalf("add: %+v", changes[0])
	}
	if changes[1].Kind != "update" || len(changes[1].Chunks) != 1 {
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
	_, err := patch.Apply(dir, patchText, resolve, patch.ApplyOptions{MaxChangedLines: 100})
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
	_, err := patch.Apply(dir, bad, resolve, patch.ApplyOptions{})
	if err == nil {
		t.Fatal("expected apply error")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "f.txt"))
	if string(b) != orig {
		t.Fatalf("file not rolled back: %q", b)
	}
}
