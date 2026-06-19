package permission_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/datadir"
	"github.com/wzhejunqiu/ds-code/internal/mcp/resultstore"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
)

func projectEngine(t *testing.T, projectRoot string) *permission.Engine {
	t.Helper()
	e := permission.NewEngine("auto", projectRoot, false)
	e.ProjectRoot = projectRoot
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
	e := projectEngine(t, root)

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
	e := projectEngine(t, root)

	got, err := e.CheckReadablePath(path)
	if err != nil {
		t.Fatalf("same project other session spill should be readable: %v", err)
	}
	if got != path {
		t.Fatalf("got %q want %q", got, path)
	}
}

func TestCheckReadablePath_mcpSpillRelativePathDenied(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	writeSpill(t, root, "sess-a", "call_abc", "body")
	e := projectEngine(t, root)

	if _, err := e.CheckReadablePath("mcp-result/sess-a/call_abc.txt"); err == nil {
		t.Fatal("relative spill path should be denied")
	}
}

func TestCheckReadablePath_sessionsDB(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	e := projectEngine(t, root)

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

	got, err := e.CheckReadablePath(dbPath)
	if err != nil {
		t.Fatalf("sessions.db should be readable: %v", err)
	}
	if got != dbPath {
		t.Fatalf("got %q want %q", got, dbPath)
	}
}

func TestCheckReadablePath_agentsOutputAllowed(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	e := projectEngine(t, root)

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

	got, err := e.CheckReadablePath(outPath)
	if err != nil {
		t.Fatalf("agents/*.output should be readable: %v", err)
	}
	if got != outPath {
		t.Fatalf("got %q want %q", got, outPath)
	}
}

func TestCheckReadablePath_projectDataReadonlyMode(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	path := writeSpill(t, root, "sess-a", "call_abc", "readonly ok")
	e := permission.NewEngine("readonly", root, false)
	e.ProjectRoot = root

	got, err := e.CheckReadablePath(path)
	if err != nil {
		t.Fatalf("readonly should read project data without ask: %v", err)
	}
	if got != path {
		t.Fatalf("got %q", got)
	}
}

func TestCheckReadablePath_otherProjectDenied(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	testutil.IsolatedHome(t)
	path := writeSpill(t, rootA, "sess-a", "call_abc", "project a only")

	e := projectEngine(t, rootB)
	if _, err := e.CheckReadablePath(path); err == nil {
		t.Fatal("other project data dir should be denied")
	}
}

func TestCheckReadablePath_projectDataDirDenied(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	e := projectEngine(t, root)

	dataDir, err := datadir.ProjectDataDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := e.CheckReadablePath(dataDir); err == nil {
		t.Fatal("directory path should be denied")
	}
}

func TestIsProjectDataPath(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	e := projectEngine(t, root)
	path := writeSpill(t, root, "sess", "call", "body")

	if !e.IsProjectDataPath(path) {
		t.Fatal("spill path should be project data path")
	}
	if e.IsProjectDataPath(filepath.Join(root, "workspace.txt")) {
		t.Fatal("workspace file should not be project data path")
	}
}
