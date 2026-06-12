//go:build tuitest

package tuitest

import (
	"os"
	"os/exec"
	"path/filepath"
)

// PrepareProjectRoot creates a minimal project tree for harness tests.
func PrepareProjectRoot(dir string) error {
	if err := writeMinimalFixture(dir); err != nil {
		return err
	}
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	_ = cmd.Run()
	return nil
}

func writeMinimalFixture(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	files := map[string]string{
		"AGENTS.md":           "# harness\n",
		"sample.go":           "package main\n\nfunc Hello() string { return \"hello\" }\n",
		"sample_multiline.go": "package main\n\nfunc Hello() string {\n\treturn \"hello\"\n}\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}
