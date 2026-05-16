package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// RequireGopls skips the test when gopls is not installed.
func RequireGopls(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not in PATH (install: go install golang.org/x/tools/gopls@latest)")
	}
}

// WriteGoModuleWithTypeError creates a minimal Go module with a type error in main.go.
func WriteGoModuleWithTypeError(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write(t, filepath.Join(dir, "go.mod"), "module testmod\n\ngo 1.22\n")
	write(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {\n\tvar x int = \"bad\"\n}\n")
	return dir
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
