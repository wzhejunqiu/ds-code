package permission_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/datadir"
	"github.com/wzhejunqiu/ds-code/internal/mcp/resultstore"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
)

func spillEngine(t *testing.T, projectRoot, sessionID string) *permission.Engine {
	t.Helper()
	e := permission.NewEngine("auto", projectRoot, false)
	e.ProjectRoot = projectRoot
	e.SpillSessionID = sessionID
	return e
}

func writeSpill(t *testing.T, projectRoot, sessionID, callID, body string) string {
	t.Helper()
	store := &resultstore.Store{ProjectRoot: projectRoot}
	path, err := store.Save(sessionID, callID, body)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckReadablePath_mcpSpillFile(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	path := writeSpill(t, root, "sess-a", "call_abc", "full mcp body")
	e := spillEngine(t, root, "sess-a")

	got, err := e.CheckReadablePath(path)
	if err != nil {
		t.Fatalf("read spill: %v", err)
	}
	if got != path {
		t.Fatalf("got %q want %q", got, path)
	}
}

func TestCheckReadablePath_mcpSpillOtherSession(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	path := writeSpill(t, root, "sess-child", "call_abc", "child spill")
	e := spillEngine(t, root, "sess-parent")

	if _, err := e.CheckReadablePath(path); err == nil {
		t.Fatal("parent session should not read child spill")
	}
}

func TestCheckReadablePath_mcpSpillRelativePathDenied(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	writeSpill(t, root, "sess-a", "call_abc", "body")
	e := spillEngine(t, root, "sess-a")

	if _, err := e.CheckReadablePath("mcp-result/sess-a/call_abc.txt"); err == nil {
		t.Fatal("relative spill path should be denied")
	}
}

func TestCheckReadablePath_mcpSpillNonTxtDenied(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	e := spillEngine(t, root, "sess-a")

	dataDir, err := datadir.ProjectDataDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dataDir, "sessions.db")
	if err := os.WriteFile(dbPath, []byte("sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := e.CheckReadablePath(dbPath); err == nil {
		t.Fatal("sessions.db should not be readable via spill exception")
	}
}

func TestCheckReadablePath_agentsOutputDenied(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	e := spillEngine(t, root, "sess-a")

	dataDir, err := datadir.ProjectDataDir(root)
	if err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dataDir, "agents", "parent-sess", "tc-1.output")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outPath, []byte("summary spill"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := e.CheckReadablePath(outPath); err == nil {
		t.Fatal("agents/*.output should not be readable via mcp-result exception")
	}
}

func TestCheckReadablePath_mcpSpillReadonlyMode(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	path := writeSpill(t, root, "sess-a", "call_abc", "readonly ok")
	e := permission.NewEngine("readonly", root, false)
	e.ProjectRoot = root
	e.SpillSessionID = "sess-a"

	got, err := e.CheckReadablePath(path)
	if err != nil {
		t.Fatalf("readonly should read spill without ask: %v", err)
	}
	if got != path {
		t.Fatalf("got %q", got)
	}
}

func TestCheckReadablePath_subagentSpillDeniedFromParent(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	childPath := writeSpill(t, root, "child-session", "call_1", "child only")

	parent := spillEngine(t, root, "parent-session")
	if _, err := parent.CheckReadablePath(childPath); err == nil {
		t.Fatal("parent SpillSessionID must not read child spill")
	}
	if !strings.Contains(childPath, "child-session") {
		t.Fatalf("unexpected spill path %q", childPath)
	}
}
