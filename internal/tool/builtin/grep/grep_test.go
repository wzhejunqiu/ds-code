package grep_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/grep"
	"github.com/wzhejunqiu/ds-code/internal/tool/searchskip"
)

type grepFixture struct {
	dir string
	g   *grep.GrepTool
}

func newGrepFixture(t *testing.T, opts ...func(*config.Config)) *grepFixture {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{Tools: config.ToolsConfig{
		Grep: config.GrepToolConfig{
			HeadLimit:        250,
			RespectGitignore: false,
			Binary:           "bundled",
			Timeout:          20 * time.Second,
		},
		Search: config.SearchToolConfig{SkipDirs: []string{"node_modules"}},
	}}
	for _, opt := range opts {
		opt(cfg)
	}
	return &grepFixture{
		dir: dir,
		g: &grep.GrepTool{
			Cfg:        cfg,
			Perm:       permission.NewEngine("readonly", dir, false),
			SearchSkip: searchskip.New(cfg.Tools.Search.SkipDirs),
		},
	}
}

func (f *grepFixture) exec(t *testing.T, args map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.g.Execute(context.Background(), raw)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return out
}

func (f *grepFixture) write(t *testing.T, rel, content string) {
	t.Helper()
	p := filepath.Join(f.dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (f *grepFixture) chtimes(t *testing.T, rel string, mod time.Time) {
	t.Helper()
	p := filepath.Join(f.dir, filepath.FromSlash(rel))
	if err := os.Chtimes(p, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func TestGrepTool_B1_files_basic(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "a.txt", "needle one\n")
	f.write(t, "pkg/b.txt", "needle two\n")
	out := f.exec(t, map[string]any{"pattern": "needle"})
	if !strings.HasPrefix(out, "Found 2 files\n") {
		t.Fatalf("got %q", out)
	}
	if strings.Contains(out, f.dir) {
		t.Fatalf("absolute path leaked: %q", out)
	}
	for _, want := range []string{"a.txt", "pkg/b.txt"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestGrepTool_B2_content_basic(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "a.txt", "needle one\nneedle two\n")
	out := f.exec(t, map[string]any{"pattern": "needle", "output_mode": "content"})
	if !strings.Contains(out, "a.txt:1:needle one") || !strings.Contains(out, "a.txt:2:needle two") {
		t.Fatalf("got %q", out)
	}
}

func TestGrepTool_B3_count_basic(t *testing.T) {
	f := newGrepFixture(t)
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	new := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	f.write(t, "a.txt", "needle one\nneedle two\n")
	f.write(t, "b.txt", "needle three\n")
	f.chtimes(t, "a.txt", new)
	f.chtimes(t, "b.txt", old)
	out := f.exec(t, map[string]any{"pattern": "needle", "output_mode": "count"})
	want := "a.txt:2\nb.txt:1\nFound 3 occurrences across 2 files"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestGrepTool_B4_no_match_files(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "a.txt", "hello\n")
	out := f.exec(t, map[string]any{"pattern": "zzz"})
	if out != "Found 0 files" {
		t.Fatalf("got %q", out)
	}
}

func TestGrepTool_B5_no_match_content(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "a.txt", "hello\n")
	out := f.exec(t, map[string]any{"pattern": "zzz", "output_mode": "content"})
	if out != "" {
		t.Fatalf("got %q", out)
	}
}

func TestGrepTool_B6_no_match_count(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "a.txt", "hello\n")
	out := f.exec(t, map[string]any{"pattern": "zzz", "output_mode": "count"})
	if out != "Found 0 occurrences across 0 files" {
		t.Fatalf("got %q", out)
	}
}

func TestGrepTool_B7_rg_exit1_not_error(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "a.txt", "hello\n")

	cases := []struct {
		name       string
		outputMode string
		want       string
	}{
		{"files_with_matches", "", "Found 0 files"},
		{"content", builtin.GrepOutputContent, ""},
		{"count", builtin.GrepOutputCount, "Found 0 occurrences across 0 files"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{"pattern": "zzz_not_found_xyz"}
			if tc.outputMode != "" {
				args["output_mode"] = tc.outputMode
			}
			raw, err := json.Marshal(args)
			if err != nil {
				t.Fatal(err)
			}
			out, err := f.g.Execute(context.Background(), raw)
			if err != nil {
				t.Fatalf("ripgrep exit 1 must not surface as error: %v", err)
			}
			if out != tc.want {
				t.Fatalf("got %q want %q", out, tc.want)
			}
		})
	}
}

func TestGrepTool_B8_head_limit_files(t *testing.T) {
	f := newGrepFixture(t)
	for i := 0; i < 5; i++ {
		f.write(t, fmt.Sprintf("f%02d.txt", i), "needle\n")
	}
	out := f.exec(t, map[string]any{"pattern": "needle", "head_limit": 2})
	if !strings.HasPrefix(out, "Found 5 files\n") {
		t.Fatalf("got %q", out)
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 4 { // summary + 2 paths + footer
		t.Fatalf("expected 4 lines, got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[3], "[Showing results with pagination = limit: 2, offset: 0]") {
		t.Fatalf("missing pagination footer: %q", out)
	}
}

func TestGrepTool_B9_head_limit_content(t *testing.T) {
	f := newGrepFixture(t)
	var b strings.Builder
	for i := 0; i < 5; i++ {
		b.WriteString("needle line\n")
	}
	f.write(t, "many.txt", b.String())
	out := f.exec(t, map[string]any{"pattern": "needle", "output_mode": "content", "head_limit": 2})
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 2 lines + footer, got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[2], "[Showing results with pagination = limit: 2, offset: 0]") {
		t.Fatalf("missing footer: %q", out)
	}
}

func TestGrepTool_B10_head_limit_count(t *testing.T) {
	f := newGrepFixture(t)
	for i := 0; i < 5; i++ {
		f.write(t, fmt.Sprintf("f%02d.txt", i), "needle\nneedle\n")
	}
	out := f.exec(t, map[string]any{"pattern": "needle", "output_mode": "count", "head_limit": 2})
	if !strings.Contains(out, "Found 10 occurrences across 5 files") {
		t.Fatalf("got %q", out)
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 4 { // 2 count lines + summary + footer
		t.Fatalf("expected 4 lines, got %d: %q", len(lines), out)
	}
}

func TestGrepTool_B11_offset_files(t *testing.T) {
	f := newGrepFixture(t)
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	new := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	f.write(t, "old.txt", "needle\n")
	f.write(t, "new.txt", "needle\n")
	f.chtimes(t, "old.txt", old)
	f.chtimes(t, "new.txt", new)
	out := f.exec(t, map[string]any{"pattern": "needle", "offset": 1, "head_limit": 1})
	if !strings.Contains(out, "old.txt") {
		t.Fatalf("expected second file (older): %q", out)
	}
	if !strings.Contains(out, "offset: 1") {
		t.Fatalf("missing offset footer: %q", out)
	}
}

func TestGrepTool_B12_ignore_case(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "a.txt", "NEEDLE\n")
	out := f.exec(t, map[string]any{"pattern": "needle", "-i": true})
	if !strings.Contains(out, "a.txt") {
		t.Fatalf("got %q", out)
	}
}

func TestGrepTool_B13_context_lines(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "a.txt", "before\nneedle\nafter\n")
	out := f.exec(t, map[string]any{"pattern": "needle", "output_mode": "content", "-C": 1})
	if !strings.Contains(out, "a.txt:1-before") {
		t.Fatalf("missing context before: %q", out)
	}
	if !strings.Contains(out, "a.txt:2:needle") {
		t.Fatalf("missing match line: %q", out)
	}
	if !strings.Contains(out, "a.txt:3-after") {
		t.Fatalf("missing context after: %q", out)
	}
}

func TestGrepTool_B14_no_line_numbers(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "a.txt", "needle\n")
	out := f.exec(t, map[string]any{"pattern": "needle", "output_mode": "content", "-n": false})
	if out != "a.txt:needle" {
		t.Fatalf("got %q", out)
	}
}

func TestGrepTool_B15_glob(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "pkg/a.go", "needle go\n")
	f.write(t, "pkg/b.txt", "needle txt\n")
	out := f.exec(t, map[string]any{"pattern": "needle", "path": "pkg", "glob": "*.go"})
	if !strings.Contains(out, "a.go") {
		t.Fatalf("got %q", out)
	}
	if strings.Contains(out, "b.txt") {
		t.Fatalf("glob should exclude txt: %q", out)
	}
}

func TestGrepTool_B16_type_go(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "a.go", "package main\nneedle\n")
	f.write(t, "a.txt", "needle\n")
	out := f.exec(t, map[string]any{"pattern": "needle", "type": "go"})
	if !strings.Contains(out, "a.go") {
		t.Fatalf("got %q", out)
	}
	if strings.Contains(out, "a.txt") {
		t.Fatalf("type filter failed: %q", out)
	}
}

func TestGrepTool_B17_multiline(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "a.txt", "start\nneedle end\n")
	out := f.exec(t, map[string]any{"pattern": "start\\nneedle", "output_mode": "content", "multiline": true})
	if !strings.Contains(out, "a.txt") {
		t.Fatalf("got %q", out)
	}
}

func TestGrepTool_B18_mtime_sort(t *testing.T) {
	f := newGrepFixture(t)
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	new := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	f.write(t, "old.txt", "needle old\n")
	f.write(t, "new.txt", "needle new\n")
	f.chtimes(t, "old.txt", old)
	f.chtimes(t, "new.txt", new)
	out := f.exec(t, map[string]any{"pattern": "needle", "output_mode": "content"})
	newIdx := strings.Index(out, "new.txt")
	oldIdx := strings.Index(out, "old.txt")
	if newIdx < 0 || oldIdx < 0 || newIdx > oldIdx {
		t.Fatalf("newer file should appear first: %q", out)
	}
}

func TestGrepTool_B19_sensitive_paths(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, ".env", "SECRET=needle\n")
	f.write(t, "ok.txt", "needle ok\n")
	out := f.exec(t, map[string]any{"pattern": "needle"})
	if strings.Contains(out, ".env") || strings.Contains(out, "SECRET") {
		t.Fatalf("leaked sensitive: %q", out)
	}
	if !strings.Contains(out, "ok.txt") {
		t.Fatalf("got %q", out)
	}
}

func TestGrepTool_B20_skip_dirs(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "node_modules/pkg/hit.txt", "needle hit\n")
	f.write(t, "ok.txt", "needle ok\n")
	out := f.exec(t, map[string]any{"pattern": "needle", "path": "."})
	if strings.Contains(out, "node_modules") {
		t.Fatalf("should skip node_modules: %q", out)
	}
	if !strings.Contains(out, "ok.txt") {
		t.Fatalf("got %q", out)
	}
}

func TestGrepTool_B21_explicit_skip_dir_path(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "node_modules/pkg/hit.txt", "needle hit\n")
	f.write(t, "ok.txt", "needle ok\n")
	out := f.exec(t, map[string]any{"pattern": "needle", "path": "node_modules"})
	if !strings.Contains(out, "hit.txt") {
		t.Fatalf("explicit path should search skip_dir: %q", out)
	}
}

func TestGrepTool_B22_git_path_empty(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, ".git/HEAD", "needle in git\n")
	modes := []struct {
		mode string
		want string
	}{
		{"", "Found 0 files"},
		{"content", ""},
		{"count", "Found 0 occurrences across 0 files"},
	}
	for _, tc := range modes {
		args := map[string]any{"pattern": "needle", "path": ".git"}
		if tc.mode != "" {
			args["output_mode"] = tc.mode
		}
		out := f.exec(t, args)
		if out != tc.want {
			t.Fatalf("mode %q: got %q want %q", tc.mode, out, tc.want)
		}
	}
}

func TestGrepTool_B22b_broad_path_skips_git(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "a.txt", "needle in source\n")
	f.write(t, ".git/HEAD", "needle in git\n")
	out := f.exec(t, map[string]any{"pattern": "needle"})
	if strings.Contains(out, ".git") {
		t.Fatalf("broad search must not include .git, got %q", out)
	}
	if !strings.Contains(out, "a.txt") {
		t.Fatalf("expected a.txt in results, got %q", out)
	}
}

func TestGrepTool_B23_respect_gitignore_true(t *testing.T) {
	f := newGrepFixture(t, func(c *config.Config) {
		c.Tools.Grep.RespectGitignore = true
	})
	f.write(t, ".gitignore", "ignored/\n")
	f.write(t, "ignored/hit.txt", "needle hit\n")
	if err := os.MkdirAll(filepath.Join(f.dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.dir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := f.exec(t, map[string]any{"pattern": "needle"})
	if strings.Contains(out, "ignored") {
		t.Fatalf("respect_gitignore=true should skip ignored: %q", out)
	}
}

func TestGrepTool_B24_respect_gitignore_false(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, ".gitignore", "ignored/\n")
	f.write(t, "ignored/hit.txt", "needle hit\n")
	out := f.exec(t, map[string]any{"pattern": "needle", "path": "ignored"})
	if !strings.Contains(out, "hit.txt") {
		t.Fatalf("default should search ignored when path explicit: %q", out)
	}
}

func TestGrepTool_B25_timeout(t *testing.T) {
	f := newGrepFixture(t, func(c *config.Config) {
		c.Tools.Grep.Timeout = time.Millisecond
	})
	for i := 0; i < 50; i++ {
		f.write(t, fmt.Sprintf("f%03d.txt", i), strings.Repeat("needle line\n", 100))
	}
	raw, _ := json.Marshal(map[string]any{"pattern": "needle"})
	_, err := f.g.Execute(context.Background(), raw)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "超时") {
		t.Fatalf("expected timeout message, got %v", err)
	}
}

func TestGrepTool_B26_context_canceled(t *testing.T) {
	f := newGrepFixture(t)
	for i := 0; i < 100; i++ {
		f.write(t, fmt.Sprintf("f%03d.txt", i), "needle line\n")
	}
	raw, _ := json.Marshal(map[string]any{"pattern": "needle", "output_mode": "content"})
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := f.g.Execute(ctx, raw)
		errCh <- err
	}()
	cancel()
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestGrepTool_B27_invalid_regex(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "a.txt", "x\n")
	raw, _ := json.Marshal(map[string]any{"pattern": "["})
	_, err := f.g.Execute(context.Background(), raw)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), builtin.ErrInvalidRegex) {
		t.Fatalf("got %v", err)
	}
}

func TestGrepTool_invalidOutputMode(t *testing.T) {
	f := newGrepFixture(t)
	raw, _ := json.Marshal(map[string]any{"pattern": "x", "output_mode": "invalid"})
	_, err := f.g.Execute(context.Background(), raw)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGrepTool_files_dedupes(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, "a.txt", "needle one\nneedle two\n")
	out := f.exec(t, map[string]any{"pattern": "needle", "output_mode": "files_with_matches"})
	want := "Found 1 files\na.txt"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestGrepTool_findsGitignoredPaths_default(t *testing.T) {
	f := newGrepFixture(t)
	f.write(t, ".gitignore", "pkg/build/\n")
	f.write(t, "pkg/src/hit.txt", "needle hit\n")
	f.write(t, "pkg/build/miss.txt", "needle miss\n")
	out := f.exec(t, map[string]any{"pattern": "needle", "path": "pkg"})
	if !strings.Contains(out, "miss.txt") || !strings.Contains(out, "hit.txt") {
		t.Fatalf("default should not follow gitignore: %q", out)
	}
}

func TestGrepTool_skipsBinaryFiles(t *testing.T) {
	f := newGrepFixture(t)
	pngData := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	pngData = append(pngData, []byte("needle in png")...)
	p := filepath.Join(f.dir, "img.png")
	if err := os.WriteFile(p, pngData, 0o644); err != nil {
		t.Fatal(err)
	}
	f.write(t, "ok.txt", "needle text\n")
	out := f.exec(t, map[string]any{"pattern": "needle"})
	if strings.Contains(out, "img.png") {
		t.Fatalf("should skip binary: %q", out)
	}
	if !strings.Contains(out, "ok.txt") {
		t.Fatalf("got %q", out)
	}
}
