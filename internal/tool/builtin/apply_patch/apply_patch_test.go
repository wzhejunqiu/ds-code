package apply_patch_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/apply_patch"
	"github.com/wzhejunqiu/ds-code/internal/tool/readgate"
)

func TestApplyPatch_allowsDotDotInside(t *testing.T) {
	root := t.TempDir()
	pkg := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(pkg, "foo.go")
	if err := os.WriteFile(target, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{ProjectRoot: root}
	perm := permission.NewEngine("auto", root, false)
	tool := &apply_patch.ApplyPatchTool{Cfg: cfg, Perm: perm, Strict: false}

	patch := `*** Begin Patch
*** Update File: pkg/../pkg/foo.go
@@
 package pkg
+// added
*** End Patch`
	args, _ := json.Marshal(map[string]any{"patch": patch})
	out, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("apply_patch with dot-dot segment: %v", err)
	}
	if !strings.Contains(out, "已应用") {
		t.Fatalf("unexpected output: %q", out)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "// added") {
		t.Fatalf("patch not applied: %q", data)
	}
}

func TestApplyPatch_readGate_missingRead(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "foo.go")
	if err := os.WriteFile(target, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{ProjectRoot: root}
	perm := permission.NewEngine("auto", root, false)
	tool := &apply_patch.ApplyPatchTool{Cfg: cfg, Perm: perm, Strict: false}
	patch := `*** Begin Patch
*** Update File: foo.go
@@
 package pkg
+// added
*** End Patch`
	args, _ := json.Marshal(map[string]any{"patch": patch})
	ctx := withReadGate(t, root, map[string]struct{}{}, map[string]struct{}{})
	_, err := tool.Execute(ctx, args)
	if err == nil || !strings.Contains(err.Error(), "编辑前须先 read_file") {
		t.Fatalf("expected must-read error, got %v", err)
	}
}

func TestApplyPatch_readGate_sameBatch(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "foo.go")
	if err := os.WriteFile(target, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	canon, err := readgateCanonical(root, "foo.go")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{ProjectRoot: root}
	perm := permission.NewEngine("auto", root, false)
	tool := &apply_patch.ApplyPatchTool{Cfg: cfg, Perm: perm, Strict: false}
	patch := `*** Begin Patch
*** Update File: foo.go
@@
 package pkg
+// added
*** End Patch`
	args, _ := json.Marshal(map[string]any{"patch": patch})
	ctx := withReadGate(t, root, map[string]struct{}{canon: {}}, map[string]struct{}{canon: {}})
	_, err = tool.Execute(ctx, args)
	if err == nil || !strings.Contains(err.Error(), "同一文件") {
		t.Fatalf("expected same-batch error, got %v", err)
	}
}

func TestApplyPatch_readGate_ok(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "foo.go")
	if err := os.WriteFile(target, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	canon, err := readgateCanonical(root, "foo.go")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{ProjectRoot: root}
	perm := permission.NewEngine("auto", root, false)
	tool := &apply_patch.ApplyPatchTool{Cfg: cfg, Perm: perm, Strict: false}
	patch := `*** Begin Patch
*** Update File: foo.go
@@
 package pkg
+// added
*** End Patch`
	args, _ := json.Marshal(map[string]any{"patch": patch})
	ctx := withReadGate(t, root, map[string]struct{}{canon: {}}, map[string]struct{}{})
	out, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "已应用") {
		t.Fatalf("out=%q", out)
	}
}

func TestApplyPatch_readGate_addOnly(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{ProjectRoot: root}
	perm := permission.NewEngine("auto", root, false)
	tool := &apply_patch.ApplyPatchTool{Cfg: cfg, Perm: perm, Strict: false}
	patch := `*** Begin Patch
*** Add File: new.go
+package new
*** End Patch`
	args, _ := json.Marshal(map[string]any{"patch": patch})
	ctx := withReadGate(t, root, map[string]struct{}{}, map[string]struct{}{})
	out, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "已应用") {
		t.Fatalf("out=%q", out)
	}
}

func withReadGate(t *testing.T, workspace string, snapshot, sameBatch map[string]struct{}) context.Context {
	t.Helper()
	gate := readgate.NewGate(workspace, snapshot, sameBatch, func(string) {})
	return readgate.WithGate(context.Background(), gate)
}

func readgateCanonical(workspace, rel string) (string, error) {
	return readgate.CanonicalPath(workspace, rel)
}
