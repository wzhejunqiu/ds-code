package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuildTag_releaseAndTuitestCompile(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	if _, err := os.Stat(filepath.Join(root, "third_party", "tokenizers", "libtokenizers.a")); err != nil {
		t.Skip("tokenizers static lib missing; run make fetch-tokenizers")
	}

	cases := []struct {
		name string
		args []string
	}{
		{
			name: "release",
			args: []string{"build", "-o", filepath.Join(t.TempDir(), "ds-code"), "./cmd/ds-code"},
		},
		{
			name: "tuitest",
			args: []string{"build", "-tags=tuitest", "-o", filepath.Join(t.TempDir(), "ds-code-tui-test"), "./cmd/ds-code-tui-test"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command("go", tc.args...)
			cmd.Dir = root
			cmd.Env = append(os.Environ(),
				"CGO_ENABLED=1",
				"CGO_LDFLAGS=-L"+filepath.Join(root, "third_party", "tokenizers"),
			)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go %v failed: %v\n%s", tc.args, err, out)
			}
		})
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
