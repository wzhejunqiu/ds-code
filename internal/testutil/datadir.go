package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/datadir"
)

// SetIsolatedHome redirects HOME (and USERPROFILE on Windows) to dir.
func SetIsolatedHome(dir string) error {
	if err := os.Setenv("HOME", dir); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		if err := os.Setenv("USERPROFILE", dir); err != nil {
			return err
		}
	}
	return nil
}

// NewIsolatedHome creates a temp directory and redirects HOME to it.
// The caller must remove dir when finished (e.g. harness Close).
func NewIsolatedHome() (string, error) {
	dir, err := os.MkdirTemp("", "ds-code-isolated-home-*")
	if err != nil {
		return "", err
	}
	if err := SetIsolatedHome(dir); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

// IsolatedHome redirects HOME (and USERPROFILE on Windows) to a fresh t.TempDir()
// and returns that directory. All ~/.ds-code paths resolve under it for the test.
//
// Call at the start of any test that triggers EnsureProjectDataDir, OpenDefault,
// logging.Setup, checkpoint.OpenStore, or shelljobs manager.Open.
func IsolatedHome(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}
	return dir
}

// IsolatedProjectRoot returns a workspace directory under an isolated HOME.
// Both the workspace and ~/.ds-code under that HOME are removed when the test finishes.
func IsolatedProjectRoot(t testing.TB) string {
	t.Helper()
	home := IsolatedHome(t)
	root := filepath.Join(home, "workspace")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

// CleanupProjectData removes ~/.ds-code/projects/<id>/ for projectRoot.
func CleanupProjectData(projectRoot string) error {
	dir, err := datadir.ProjectDataDir(projectRoot)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}
