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
