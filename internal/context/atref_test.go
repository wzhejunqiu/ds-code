package context_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/permission"
)

func TestAtExpander_file(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("explain @foo.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "@foo.go") {
		t.Fatalf("output should preserve @ ref in user text: %q", out)
	}
	if !strings.Contains(out, "package main") {
		t.Fatalf("output missing file content: %q", out)
	}
}

func TestAtExpander_preservesAtInSentence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("按照 @foo.go 的要求")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "按照 @foo.go 的要求") {
		t.Fatalf("output should preserve @ ref in sentence: %q", out)
	}
	if !strings.Contains(out, "--- @foo.go") {
		t.Fatalf("output missing expanded block: %q", out)
	}
}

func TestAtExpander_dirIncludesSensitiveAtRoot(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "code.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("see @./")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, ".env") {
		t.Fatalf("directory @ ref should list .env path: %q", out)
	}
	if !strings.Contains(out, "pkg/code.go") {
		t.Fatalf("expected pkg/code.go in listing: %q", out)
	}
	if strings.Contains(out, "SECRET=1") {
		t.Fatalf("directory @ ref should not include file contents: %q", out)
	}
}

func TestAtExpander_sensitiveFileAllowed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("check @.env")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SECRET=1") {
		t.Fatalf("expected @.env to expand sensitive content: %q", out)
	}
}

func TestAtExpander_sshDirAllowed(t *testing.T) {
	dir := t.TempDir()
	sshDir := filepath.Join(dir, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host *\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("review @.ssh/config")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Host *") {
		t.Fatalf("expected @.ssh/config content: %q", out)
	}
}

func TestAtExpander_dirIgnoresSkipDirs(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules", "pkg")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "lib.go"), []byte("package lib\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Context: config.ContextConfig{AtReferenceMaxChars: 10000, AtDirMaxFiles: 20},
		Tools:   config.ToolsConfig{Search: config.SearchToolConfig{SkipDirs: []string{"node_modules"}}},
	}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("load @node_modules/pkg/")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "lib.go") {
		t.Fatalf("@dir should list files under skip_dirs: %q", out)
	}
	if strings.Contains(out, "package lib") {
		t.Fatalf("@dir should not include file contents: %q", out)
	}
}

func TestAtExpander_dirIgnoresGitignore(t *testing.T) {
	dir := t.TempDir()
	ignored := filepath.Join(dir, "ignored")
	if err := os.MkdirAll(ignored, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("ignored/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ignored, "code.go"), []byte("package ignored\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000, AtDirMaxFiles: 20}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("see @ignored/")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "code.go") {
		t.Fatalf("@dir should list gitignored files: %q", out)
	}
	if strings.Contains(out, "package ignored") {
		t.Fatalf("@dir should not include file contents: %q", out)
	}
}

func TestAtExpander_dirAllowsNodeModules(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules", "left-pad")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "index.js"), []byte("module.exports = {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000, AtDirMaxFiles: 20}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("inspect @node_modules/left-pad/")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "index.js") {
		t.Fatalf("expected index.js in listing: %q", out)
	}
	if strings.Contains(out, "module.exports") {
		t.Fatalf("directory @ ref should not include file contents: %q", out)
	}
}

func TestAtExpander_dirListingOnly(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "docs")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.md"), []byte("# A\nsecret-a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.md"), []byte("# B\nsecret-b\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("review @docs/")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "a.md") || !strings.Contains(out, "b.md") {
		t.Fatalf("expected both files listed: %q", out)
	}
	if strings.Contains(out, "secret-a") || strings.Contains(out, "secret-b") {
		t.Fatalf("directory listing must not include file bodies: %q", out)
	}
	if !strings.Contains(out, "read_file") {
		t.Fatalf("expected footer hint to use read_file: %q", out)
	}
}

func TestStripAtRefExpansionBlocks_dirListing(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "docs")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	original := "请严格检查 @docs/ 的要求"
	out, err := exp.Expand(original)
	if err != nil {
		t.Fatal(err)
	}
	if got := context.StripAtRefExpansionBlocks(out); got != original {
		t.Fatalf("got %q, want %q", got, original)
	}
}

func TestStripAtRefExpansionBlocks_file(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	original := "explain @foo.go"
	out, err := exp.Expand(original)
	if err != nil {
		t.Fatal(err)
	}
	if got := context.StripAtRefExpansionBlocks(out); got != original {
		t.Fatalf("got %q, want %q", got, original)
	}
}

func TestStripAtRefExpansionBlocks_multipleRefs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.go"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	original := "compare @a.go and @b.go"
	out, err := exp.Expand(original)
	if err != nil {
		t.Fatal(err)
	}
	if got := context.StripAtRefExpansionBlocks(out); got != original {
		t.Fatalf("got %q, want %q", got, original)
	}
}

func TestStripAtRefExpansionBlocks_atRefOnly(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "docs")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	original := "@docs/"
	out, err := exp.Expand(original)
	if err != nil {
		t.Fatal(err)
	}
	if got := context.StripAtRefExpansionBlocks(out); got != original {
		t.Fatalf("got %q, want %q", got, original)
	}
}
