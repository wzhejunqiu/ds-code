package testutil

import (
	"runtime"
	"testing"
)

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
