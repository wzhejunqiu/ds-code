package permission

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/mcp/resultstore"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
)

func TestTokenizeShellCmd(t *testing.T) {
	got := tokenizeShellCmd(`cat .env && echo "hi there"`)
	want := []string{"cat", ".env", "&&", "echo", "hi there"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token[%d] = %q, want %q (full %v)", i, got[i], want[i], got)
		}
	}
}

func TestCheckPathCandidate_ignoresShellRedirection(t *testing.T) {
	root := t.TempDir()
	e := NewEngine("auto", root, false)
	if err := e.checkPathCandidate("2>/dev/null"); err != nil {
		t.Fatalf("redirection token should be ignored: %v", err)
	}
}

func TestCheckPathCandidate_blocksAbsoluteOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	e := NewEngine("auto", root, false)
	if err := e.checkPathCandidate("/etc/passwd"); err == nil {
		t.Fatal("expected deny for absolute path outside workspace")
	}
}

func TestCheckPathCandidate_allowsGitRevisionRange(t *testing.T) {
	root := t.TempDir()
	e := NewEngine("auto", root, false)
	cases := []string{
		"origin/main..v0.1.1",
		"origin/main...v0.1.1",
		"./...",
	}
	for _, tok := range cases {
		if err := e.checkPathCandidate(tok); err != nil {
			t.Fatalf("token %q should be allowed: %v", tok, err)
		}
	}
}

func TestCheckPathCandidate_blocksTraversal(t *testing.T) {
	root := t.TempDir()
	e := NewEngine("auto", root, false)
	if err := e.checkPathCandidate("../outside"); err == nil {
		t.Fatal("expected deny for traversal")
	}
}

func TestEngine_shell_allowsGitDiffTripleDot(t *testing.T) {
	root := t.TempDir()
	e := NewEngine("auto", root, true)
	cmd := "git diff origin/main...v0.1.1 --stat"
	if err := e.Check("bash", map[string]any{"command": cmd}); err != nil {
		t.Fatalf("permission should allow git revision range: %v", err)
	}
}

func TestEngine_shell_allowsGoTestEllipsis(t *testing.T) {
	root := t.TempDir()
	e := NewEngine("auto", root, true)
	if err := e.Check("bash", map[string]any{"command": "go test ./..."}); err != nil {
		t.Fatalf("permission should allow go package ellipsis: %v", err)
	}
}

func TestCheckPathCandidate_allowsGoTestEllipsis(t *testing.T) {
	root := t.TempDir()
	e := NewEngine("auto", root, false)
	if err := e.checkPathCandidate("./..."); err != nil {
		t.Fatalf("./... should be allowed: %v", err)
	}
}

func TestEngine_shell_gitDiffGoTest(t *testing.T) {
	root := t.TempDir()
	e := NewEngine("auto", root, true)
	cmds := []string{
		"git diff origin/main...v0.1.1 --stat",
		"git log origin/main..v0.1.1",
		"go test ./...",
	}
	for _, cmd := range cmds {
		if err := e.Check("bash", map[string]any{"command": cmd}); err != nil {
			t.Fatalf("command %q should be allowed: %v", cmd, err)
		}
	}
}

func TestEngine_shell_dotDotInside(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "readme.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := NewEngine("auto", root, true)
	if err := e.Check("bash", map[string]any{"command": "cat pkg/../pkg/readme.txt"}); err != nil {
		t.Fatalf("legal .. segment should be allowed: %v", err)
	}
}

func TestCheckShellDenylistPaths_embeddedLiteral(t *testing.T) {
	root := t.TempDir()
	e := NewEngine("auto", root, true)
	err := e.checkShellDenylistPaths(`python3 -c "open('.env').read()"`)
	if err == nil {
		t.Fatal("expected deny for embedded .env")
	}
}

func TestEngine_shell_deniesSpillAbsPath(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	store := &resultstore.Store{ProjectRoot: root}
	spillPath, err := store.Save("sess-1", "call_abc", "secret mcp output")
	if err != nil {
		t.Fatal(err)
	}
	e := NewEngine("auto", root, false)
	e.ProjectRoot = root
	e.SpillSessionID = "sess-1"

	cmd := "cat " + spillPath
	if err := e.Check("bash", map[string]any{"command": cmd}); err == nil {
		t.Fatal("shell should deny reading spill absolute path outside workspace")
	}
	if _, err := e.CheckReadablePath(spillPath); err != nil {
		t.Fatalf("read_file should allow same spill path: %v", err)
	}
}
