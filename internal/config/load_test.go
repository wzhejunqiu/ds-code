package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/wzhejunqiu/ds-code/internal/config"
)

func TestResolveProjectRoot_gitWorktreeFile(t *testing.T) {
	root := t.TempDir()
	gitFile := filepath.Join(root, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: /path/to/actual/.git/worktrees/foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := config.ResolveProjectRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(root)
	if got != want {
		t.Fatalf("project root = %q, want %q", got, want)
	}
}

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
	if cfg.LLM.Subagent.Model != "deepseek-v4-flash" {
		t.Fatalf("subagent model = %q", cfg.LLM.Subagent.Model)
	}
	if cfg.LLM.Subagent.Thinking.Type != "disabled" {
		t.Fatalf("subagent thinking = %q", cfg.LLM.Subagent.Thinking.Type)
	}
	if cfg.LLM.Subagent.ReasoningEffort != "high" {
		t.Fatalf("subagent effort = %q", cfg.LLM.Subagent.ReasoningEffort)
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

func TestApplyCLIDerived_verboseCount(t *testing.T) {
	cmd := &cobra.Command{}
	config.BindFlags(cmd)
	if err := cmd.ParseFlags([]string{"-vv"}); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	if err := config.ApplyCLIDerived(cfg, cmd); err != nil {
		t.Fatal(err)
	}
	if cfg.LogVerbosity != 2 {
		t.Fatalf("LogVerbosity = %d, want 2", cfg.LogVerbosity)
	}
}

func TestApplyCLIDerived_allowLogSensitiveData(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"vv_and_flag", []string{"-vv", "--allow-log-sensitive-data"}, true},
		{"flag_without_vv", []string{"--allow-log-sensitive-data"}, false},
		{"vv_only", []string{"-vv"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			config.BindFlags(cmd)
			if err := cmd.ParseFlags(tc.args); err != nil {
				t.Fatal(err)
			}
			cfg := &config.Config{}
			if err := config.ApplyCLIDerived(cfg, cmd); err != nil {
				t.Fatal(err)
			}
			if cfg.AllowLogSensitiveData != tc.want {
				t.Fatalf("AllowLogSensitiveData = %v, want %v", cfg.AllowLogSensitiveData, tc.want)
			}
		})
	}
}

func TestLoad_rejectsProjectAutoPermission(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dsCode := filepath.Join(dir, ".ds-code")
	if err := os.Mkdir(dsCode, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `permission:
  mode: auto
`
	if err := os.WriteFile(filepath.Join(dsCode, "config.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	config.BindFlags(cmd)
	_, err := config.Load(cmd, config.Options{StartDir: dir, SkipProjectDataDir: true})
	if err == nil || !strings.Contains(err.Error(), "cannot set permission.mode to auto") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoad_rejectsEnvAutoWithoutCLI(t *testing.T) {
	dir := t.TempDir()
	cmd := &cobra.Command{}
	config.BindFlags(cmd)
	t.Setenv("DS_CODE_PERMISSION_MODE", "auto")
	t.Cleanup(func() { t.Setenv("DS_CODE_PERMISSION_MODE", "") })

	_, err := config.Load(cmd, config.Options{StartDir: dir, SkipProjectDataDir: true})
	if err == nil || !strings.Contains(err.Error(), "requires --dangerously-auto") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoad_allowsEnvAutoWithCLI(t *testing.T) {
	dir := t.TempDir()
	cmd := &cobra.Command{}
	config.BindFlags(cmd)
	if err := cmd.ParseFlags([]string{"--dangerously-auto"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DS_CODE_PERMISSION_MODE", "auto")
	t.Cleanup(func() { t.Setenv("DS_CODE_PERMISSION_MODE", "") })

	cfg, err := config.Load(cmd, config.Options{StartDir: dir, SkipProjectDataDir: true})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Permission.Mode != "auto" {
		t.Fatalf("mode = %q", cfg.Permission.Mode)
	}
}

func TestLoad_rejectsInvalidEnvBlacklistPattern(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dsCode := filepath.Join(dir, ".ds-code")
	if err := os.Mkdir(dsCode, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `tools:
  shell:
    env_blacklist:
      - "[invalid"
`
	if err := os.WriteFile(filepath.Join(dsCode, "config.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	config.BindFlags(cmd)
	_, err := config.Load(cmd, config.Options{StartDir: dir, SkipProjectDataDir: true})
	if err == nil || !strings.Contains(err.Error(), "env_blacklist") {
		t.Fatalf("err = %v", err)
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

	t.Setenv("DS_CODE_DEEPSEEK_API_KEY", "  sk-trimmed  \n")
	k, err = config.LoadAPIKey()
	if err != nil || k != "sk-trimmed" {
		t.Fatalf("trimmed key = %q err = %v", k, err)
	}
}
