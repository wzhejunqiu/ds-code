package grep

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/searchskip"
)

func TestRelPathFromWorkspace(t *testing.T) {
	dir := t.TempDir()
	perm := permission.NewEngine("readonly", dir, false)
	sub := filepath.Join(dir, "pkg", "foo.go")
	if err := os.MkdirAll(filepath.Dir(sub), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sub, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("C1_abs_to_rel", func(t *testing.T) {
		rel, ok := relPathFromWorkspace(perm, sub)
		if !ok || rel != "pkg/foo.go" {
			t.Fatalf("got %q ok=%v", rel, ok)
		}
	})

	t.Run("C2_already_relative", func(t *testing.T) {
		rel, ok := relPathFromWorkspace(perm, "pkg/foo.go")
		if !ok || rel != "pkg/foo.go" {
			t.Fatalf("got %q ok=%v", rel, ok)
		}
	})

	t.Run("C3_escape_discarded", func(t *testing.T) {
		outside := filepath.Join(dir, "..", "outside.txt")
		_, ok := relPathFromWorkspace(perm, outside)
		if ok {
			t.Fatal("expected outside path to be discarded")
		}
	})
}

func TestBuildRipgrepArgs(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{Tools: config.ToolsConfig{
		Grep:   config.GrepToolConfig{RespectGitignore: false},
		Search: config.SearchToolConfig{SkipDirs: []string{"node_modules"}},
	}}
	tool := &GrepTool{Cfg: cfg, Perm: permission.NewEngine("readonly", dir, false), SearchSkip: searchskip.New(nil)}

	args, err := buildRipgrepArgs(tool, grepInput{Pattern: "needle", Path: "pkg", Glob: "*.go"}, builtin.GrepOutputFilesWithMatches)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"--json", "--no-ignore", "--glob", "*.go", "--glob", "!.git/**", "needle", "pkg"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
}

func TestResolveRipgrepBinary(t *testing.T) {
	t.Run("C5_bundled", func(t *testing.T) {
		path, err := resolveRipgrepBinary(config.GrepToolConfig{Binary: "bundled"})
		if err != nil {
			t.Fatal(err)
		}
		if path == "" {
			t.Fatal("empty path")
		}
	})

	t.Run("C5_system", func(t *testing.T) {
		if _, err := exec.LookPath("rg"); err != nil {
			t.Skip("rg not in PATH")
		}
		path, err := resolveRipgrepBinary(config.GrepToolConfig{Binary: "system"})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(path, "rg") && !strings.HasSuffix(path, "rg.exe") {
			t.Fatalf("unexpected path %q", path)
		}
	})

	t.Run("C5_path", func(t *testing.T) {
		path, err := resolveRipgrepBinary(config.GrepToolConfig{Binary: "path"})
		if err == nil {
			t.Fatal("expected error for empty binary_path")
		}
		_ = path
	})
}
