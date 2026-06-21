package permission_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/permission"
)

func TestEngine_askNonInteractive_deniesShell(t *testing.T) {
	e := permission.NewEngine("ask", t.TempDir(), false)
	err := e.Check("bash", map[string]any{"command": "echo hi"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err != permission.ErrNeedTTY {
		t.Fatalf("err = %v", err)
	}
}

func TestEngine_readonly_allowsGitStatus(t *testing.T) {
	e := permission.NewEngine("readonly", t.TempDir(), true)
	if err := e.Check("bash", map[string]any{"command": "git status"}); err != nil {
		t.Fatalf("git status should be allowed in readonly: %v", err)
	}
}

func TestEngine_readonly_deniesShellHighRisk(t *testing.T) {
	e := permission.NewEngine("readonly", t.TempDir(), true)
	err := e.Check("bash", map[string]any{"command": "rm -rf /tmp/foo"})
	if err == nil {
		t.Fatal("expected error for rm -rf")
	}
}

func TestEngine_readonly_deniesPrivilegedShell(t *testing.T) {
	e := permission.NewEngine("readonly", t.TempDir(), true)
	err := e.Check("bash", map[string]any{"command": "git push origin main"})
	if err == nil {
		t.Fatal("expected error for git push in readonly")
	}
}

func TestEngine_askInteractive_shellAskSinglePrompt(t *testing.T) {
	var prompts int
	e := permission.NewEngine("ask", t.TempDir(), true)
	e.Prompter = func(tool, summary string) (bool, error) {
		prompts++
		return true, nil
	}
	if err := e.Check("bash", map[string]any{"command": "git push origin main"}); err != nil {
		t.Fatalf("git push: %v", err)
	}
	if prompts != 1 {
		t.Fatalf("expected 1 prompt, got %d", prompts)
	}
}

func TestEngine_resolvePath_blocksTraversal(t *testing.T) {
	root := t.TempDir()
	e := permission.NewEngine("auto", root, true)
	_, err := e.ResolvePath("../outside")
	if err == nil {
		t.Fatal("expected traversal error")
	}
}

func TestEngine_resolvePath_allowsAbsoluteInsideWorkspace(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "internal", "ui")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(inner, "resume_picker.go")
	if err := os.WriteFile(file, []byte("package ui\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := permission.NewEngine("auto", root, true)
	got, err := e.ResolvePath(file)
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	want, err := filepath.EvalSymlinks(file)
	if err != nil {
		want = file
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEngine_resolvePath_rejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape-link")
	if err := os.Symlink(secret, link); err != nil {
		t.Skip("symlink not supported:", err)
	}

	e := permission.NewEngine("auto", root, true)
	_, err := e.ResolvePath("escape-link")
	if err == nil {
		t.Fatal("expected symlink escape to be denied")
	}
}

func TestEngine_resolvePath_rejectsAbsoluteOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	e := permission.NewEngine("auto", root, true)
	outside := filepath.Join(t.TempDir(), "secret.txt")
	_, err := e.ResolvePath(outside)
	if err == nil {
		t.Fatal("expected outside workspace error")
	}
}

func TestEngine_check_deniesSensitivePathOnWriteFile(t *testing.T) {
	root := t.TempDir()
	e := permission.NewEngine("auto", root, true)
	err := e.Check("write_file", map[string]any{"path": ".env"})
	if err == nil {
		t.Fatal("expected sensitive path denial")
	}
}

func TestEngine_check_deniesSensitivePathInPathsArray(t *testing.T) {
	root := t.TempDir()
	e := permission.NewEngine("auto", root, true)
	err := e.Check("read_file", map[string]any{
		"paths": []any{".env"},
	})
	if err == nil {
		t.Fatal("expected sensitive path denial")
	}
}

func TestEngine_check_deniesHighRiskShell(t *testing.T) {
	e := permission.NewEngine("auto", t.TempDir(), true)
	err := e.Check("bash", map[string]any{"command": "rm -rf /"})
	if err == nil {
		t.Fatal("expected high-risk shell denial")
	}
}

func TestEngine_auto_deniesShellReadingSensitiveFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	e := permission.NewEngine("auto", root, true)
	cases := []string{
		"cat .env",
		"head .envrc",
		"cat .aws/credentials",
		`python3 -c 'open(".env").read()'`,
	}
	for _, cmd := range cases {
		err := e.Check("bash", map[string]any{"command": cmd})
		if err == nil {
			t.Fatalf("auto mode should deny shell reading sensitive paths: %q", cmd)
		}
		if !errors.Is(err, permission.ErrDenied) {
			t.Fatalf("cmd %q: err = %v, want ErrDenied", cmd, err)
		}
	}
}

func TestEngine_auto_allowsShellBenignRead(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "readme.txt"), []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := permission.NewEngine("auto", root, true)
	if err := e.Check("bash", map[string]any{"command": "cat readme.txt"}); err != nil {
		t.Fatalf("benign shell read should be allowed in auto: %v", err)
	}
}

func TestEngine_checkReadablePath_dotDotInside(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "util.go")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := permission.NewEngine("auto", root, true)
	got, err := e.CheckReadablePath("pkg/../pkg/util.go")
	if err != nil {
		t.Fatalf("CheckReadablePath: %v", err)
	}
	want, _ := filepath.EvalSymlinks(file)
	if want == "" {
		want = file
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestEngine_resolvePath_allowsDotDotInside(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "util.go")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := permission.NewEngine("auto", root, true)
	got, err := e.ResolvePath("pkg/../pkg/util.go")
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	want, _ := filepath.EvalSymlinks(file)
	if want == "" {
		want = file
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestEngine_checkReadablePath_deniesNormalizedEnv(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	e := permission.NewEngine("auto", root, true)
	_, err := e.CheckReadablePath("pkg/../.env")
	if err == nil {
		t.Fatal("expected sensitive denial after normalization")
	}
}

func TestEngine_checkReadablePath_deniesSensitive(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	e := permission.NewEngine("auto", root, true)
	_, err := e.CheckReadablePath(".env")
	if err == nil {
		t.Fatal("expected sensitive denial")
	}
}

func TestEngine_check_applyPatchSensitiveFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("KEY=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := permission.NewEngine("auto", root, true)
	patch := `*** Begin Patch
*** Update File: .env
@@
-KEY=1
+KEY=2
*** End Patch`
	err := e.Check("apply_patch", map[string]any{"patch": patch})
	if err == nil {
		t.Fatal("expected sensitive path denial for apply_patch")
	}
}

func TestEngine_check_applyPatchInvalidPatch(t *testing.T) {
	e := permission.NewEngine("auto", t.TempDir(), true)
	err := e.Check("apply_patch", map[string]any{"patch": "not a patch"})
	if err == nil {
		t.Fatal("expected invalid patch error")
	}
}

func TestEngine_askInteractive_prompterApproves(t *testing.T) {
	root := t.TempDir()
	e := permission.NewEngine("ask", root, true)
	e.Prompter = func(tool, summary string) (bool, error) {
		if tool != "write_file" || summary == "" {
			t.Fatalf("prompter tool=%q summary=%q", tool, summary)
		}
		return true, nil
	}
	if err := e.Check("write_file", map[string]any{"path": "notes.txt"}); err != nil {
		t.Fatalf("expected approval: %v", err)
	}
}

func TestEngine_askInteractive_prompterRejects(t *testing.T) {
	e := permission.NewEngine("ask", t.TempDir(), true)
	e.Prompter = func(tool, summary string) (bool, error) {
		return false, nil
	}
	err := e.Check("write_file", map[string]any{"path": "notes.txt"})
	if !errors.Is(err, permission.ErrRejected) {
		t.Fatalf("err = %v, want ErrRejected", err)
	}
}

func TestEngine_askInteractive_noPrompter(t *testing.T) {
	e := permission.NewEngine("ask", t.TempDir(), true)
	err := e.Check("bash", map[string]any{"command": "git push origin main"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, permission.ErrDenied) {
		t.Fatalf("err = %v", err)
	}
}

func TestEngine_setWriteToolDetector_readonlyDeniesMCPWrite(t *testing.T) {
	e := permission.NewEngine("readonly", t.TempDir(), true)
	e.SetWriteToolDetector(func(name string) bool {
		return name == "mcp__fs__write_file"
	})
	err := e.Check("mcp__fs__write_file", map[string]any{"path": "x"})
	if err == nil {
		t.Fatal("expected readonly denial")
	}
}

func TestEngine_askInteractive_prompterSummarizesMCPArgs(t *testing.T) {
	var gotSummary string
	e := permission.NewEngine("ask", t.TempDir(), true)
	e.SetWriteToolDetector(func(name string) bool {
		return name == "mcp__srv__execute"
	})
	e.Prompter = func(tool, summary string) (bool, error) {
		gotSummary = summary
		return true, nil
	}
	if err := e.Check("mcp__srv__execute", map[string]any{"cmd": "ls"}); err != nil {
		t.Fatal(err)
	}
	if gotSummary == "" {
		t.Fatal("expected non-empty MCP summary for prompter")
	}
}

func TestPermission_FormatArgsSummary_MCPBareName(t *testing.T) {
	var gotSummary string
	e := permission.NewEngine("ask", t.TempDir(), true)
	e.SetWriteToolDetector(func(name string) bool {
		return name == "write_nodes"
	})
	e.SetMCPToolDetector(func(name string) bool {
		return name == "write_nodes"
	})
	e.Prompter = func(tool, summary string) (bool, error) {
		gotSummary = summary
		return true, nil
	}
	if err := e.Check("write_nodes", map[string]any{"query": "x", "limit": 3}); err != nil {
		t.Fatal(err)
	}
	if gotSummary == "" || !strings.Contains(gotSummary, "query") {
		t.Fatalf("summary = %q", gotSummary)
	}
}

func TestEngine_auto_deniesShellOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	e := permission.NewEngine("auto", root, true)
	outside := filepath.Join(t.TempDir(), "readme.txt")
	if err := os.WriteFile(outside, []byte("secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := e.Check("bash", map[string]any{"command": "cat " + outside})
	if err == nil {
		t.Fatal("expected denial for absolute path outside workspace")
	}
	if !errors.Is(err, permission.ErrDenied) {
		t.Fatalf("err = %v", err)
	}
}

func TestEngine_auto_allowsShellRedirectToken(t *testing.T) {
	e := permission.NewEngine("auto", t.TempDir(), true)
	if err := e.Check("bash", map[string]any{"command": "echo hi 2>/dev/null"}); err != nil {
		t.Fatalf("redirect token should not be blocked: %v", err)
	}
}
