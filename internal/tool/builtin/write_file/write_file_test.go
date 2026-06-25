package write_file_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/write_file"
	"github.com/wzhejunqiu/ds-code/internal/tool/readgate"
)

func TestWriteFile_newFile_noReadRequired(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{ProjectRoot: root}
	perm := permission.NewEngine("auto", root, false)
	tool := &write_file.WriteFileTool{Cfg: cfg, Perm: perm, Strict: false}
	args, _ := json.Marshal(map[string]any{
		"path":    "new.txt",
		"content": "hello",
	})
	out, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "已写入") {
		t.Fatalf("out=%q", out)
	}
}

func TestWriteFile_readGate_missingRead(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "foo.go")
	if err := os.WriteFile(target, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{ProjectRoot: root}
	perm := permission.NewEngine("auto", root, false)
	tool := &write_file.WriteFileTool{Cfg: cfg, Perm: perm, Strict: false}
	args, _ := json.Marshal(map[string]any{
		"path":    "foo.go",
		"content": "package pkg\n// added\n",
	})
	ctx := withReadGate(t, root, map[string]struct{}{}, map[string]struct{}{})
	_, err := tool.Execute(ctx, args)
	if err == nil || !strings.Contains(err.Error(), "覆盖已有文件前须先 read_file") {
		t.Fatalf("expected must-read error, got %v", err)
	}
}

func TestWriteFile_readGate_sameBatch(t *testing.T) {
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
	tool := &write_file.WriteFileTool{Cfg: cfg, Perm: perm, Strict: false}
	args, _ := json.Marshal(map[string]any{
		"path":    "foo.go",
		"content": "package pkg\n// added\n",
	})
	ctx := withReadGate(t, root, map[string]struct{}{canon: {}}, map[string]struct{}{canon: {}})
	_, err = tool.Execute(ctx, args)
	if err == nil || !strings.Contains(err.Error(), "同一文件") {
		t.Fatalf("expected same-batch error, got %v", err)
	}
}

func TestWriteFile_readGate_ok(t *testing.T) {
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
	tool := &write_file.WriteFileTool{Cfg: cfg, Perm: perm, Strict: false}
	args, _ := json.Marshal(map[string]any{
		"path":    "foo.go",
		"content": "package pkg\n// added\n",
	})
	ctx := withReadGate(t, root, map[string]struct{}{canon: {}}, map[string]struct{}{})
	out, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "已写入") {
		t.Fatalf("out=%q", out)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "// added") {
		t.Fatalf("content=%q", got)
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
