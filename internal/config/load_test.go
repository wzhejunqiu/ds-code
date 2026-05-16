package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/spf13/cobra"
)

func TestResolveProjectRoot_gitRepo(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "pkg")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := config.ResolveProjectRoot(sub)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(root)
	if got != want {
		t.Fatalf("project root = %q, want %q", got, want)
	}
}

func TestProjectID_stable(t *testing.T) {
	a := config.ProjectID("/tmp/foo")
	b := config.ProjectID("/tmp/foo")
	if a != b || len(a) != 64 {
		t.Fatalf("project id = %q", a)
	}
}

func TestLoad_defaults(t *testing.T) {
	dir := t.TempDir()
	cmd := &cobra.Command{}
	config.BindFlags(cmd)

	cfg, err := config.Load(cmd, config.Options{StartDir: dir, SkipProjectDataDir: true})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLM.Model != "deepseek-v4-pro" {
		t.Fatalf("model = %q", cfg.LLM.Model)
	}
	if cfg.Permission.Mode != "ask" {
		t.Fatalf("permission = %q", cfg.Permission.Mode)
	}
	if cfg.Context.WindowTokens != 1_048_576 {
		t.Fatalf("window = %d", cfg.Context.WindowTokens)
	}
	if cfg.ProjectDataDir == "" {
		t.Fatal("expected project data dir")
	}
}

func TestLoad_rejectsAPIKeyInYAML(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dsCode := filepath.Join(dir, ".ds-code")
	if err := os.Mkdir(dsCode, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `llm:
  api_key: sk-secret
`
	if err := os.WriteFile(filepath.Join(dsCode, "config.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	config.BindFlags(cmd)
	_, err := config.Load(cmd, config.Options{StartDir: dir, SkipProjectDataDir: true})
	if err == nil {
		t.Fatal("expected error for llm.api_key in yaml")
	}
}

func TestLoadAPIKey(t *testing.T) {
	t.Setenv("DS_CODE_DEEPSEEK_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
	_, err := config.LoadAPIKey()
	if err == nil {
		t.Fatal("expected missing key error")
	}

	t.Setenv("DS_CODE_DEEPSEEK_API_KEY", "sk-test")
	k, err := config.LoadAPIKey()
	if err != nil || k != "sk-test" {
		t.Fatalf("key = %q err = %v", k, err)
	}
}
