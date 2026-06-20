package permission

import (
	"path/filepath"
	"testing"
)

func TestSkipSensitiveAbs_sensitivePaths(t *testing.T) {
	dir := t.TempDir()
	eng := NewEngine("auto", dir, false)
	paths := []string{
		filepath.Join(dir, ".env"),
		filepath.Join(dir, ".ssh", "id_rsa"),
		filepath.Join(dir, "secrets.json"),
		filepath.Join(dir, "foo", ".aws", "credentials"),
	}
	for _, p := range paths {
		if !eng.SkipSensitiveAbs(p) {
			t.Errorf("SkipSensitiveAbs(%q) = false, want true", p)
		}
	}
}

func TestSkipSensitiveAbs_allowsBenignPaths(t *testing.T) {
	dir := t.TempDir()
	eng := NewEngine("auto", dir, false)
	paths := []string{
		filepath.Join(dir, "README.md"),
		filepath.Join(dir, "docs", "secrets-management.md"),
		filepath.Join(dir, "main.go"),
	}
	for _, p := range paths {
		if eng.SkipSensitiveAbs(p) {
			t.Errorf("SkipSensitiveAbs(%q) = true, want false", p)
		}
	}
}

func TestSkipSensitiveAbs_workspaceRelative(t *testing.T) {
	dir := t.TempDir()
	eng := NewEngine("auto", dir, false)
	env := filepath.Join(dir, ".env")
	ok := filepath.Join(dir, "src", "main.go")
	if !eng.SkipSensitiveAbs(env) {
		t.Fatal(".env should be sensitive")
	}
	if eng.SkipSensitiveAbs(ok) {
		t.Fatal("main.go should not be sensitive")
	}
}
