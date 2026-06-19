package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/datadir"
	"github.com/wzhejunqiu/ds-code/internal/logging"
)

// pathWithin reports whether child is parent or a descendant of parent.
func pathWithin(parent, child string) bool {
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	if err != nil {
		return false
	}
	if rel == ".." {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func TestNewIsolatedHome_redirectsProjectDataDir(t *testing.T) {
	realHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	dir, err := NewIsolatedHome()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	dataDir, err := datadir.EnsureProjectDataDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !pathWithin(dir, dataDir) {
		t.Fatalf("ProjectDataDir = %q, want under isolated home %q", dataDir, dir)
	}
	if pathWithin(filepath.Join(realHome, ".ds-code"), dataDir) && dir != realHome {
		t.Fatalf("ProjectDataDir leaked to real home: %q", dataDir)
	}
}

func TestIsolatedHome_beforeLoggingSetup(t *testing.T) {
	realHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	home := IsolatedHome(t)
	root := t.TempDir()

	cleanup, err := logging.Setup(logging.Options{ProjectRoot: root, Verbosity: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	logDir := datadir.DefaultLogsDir(root)
	if logDir == "" {
		t.Fatal("expected log dir")
	}
	if !pathWithin(home, logDir) {
		t.Fatalf("logs dir = %q, want under isolated home %q", logDir, home)
	}
	leaked, err := datadir.ProjectDataDir("")
	if err != nil {
		t.Fatal(err)
	}
	if pathWithin(filepath.Join(realHome, ".ds-code"), leaked) && home != realHome {
		// Empty project root still resolves under current HOME; must not be real home.
		if _, err := os.Stat(leaked); err == nil && !pathWithin(home, leaked) {
			t.Fatalf("empty project root data dir leaked to real home: %q", leaked)
		}
	}
}
