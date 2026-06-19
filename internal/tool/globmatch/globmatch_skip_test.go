package globmatch_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool/globmatch"
	"github.com/wzhejunqiu/ds-code/internal/tool/searchskip"
)

func TestGlobmatch_skipDirSkipsNodeModules(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules", "pkg")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "hit.js"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ok.go"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}

	perm := permission.NewEngine("readonly", root, false)
	skip := searchskip.New([]string{"node_modules"})
	skipDir := func(rel string) bool { return skip.ShouldSkipWalkDir(rel, ".") }
	skipSensitive := func(abs string) bool { return perm.SkipSensitiveAbs(abs) }

	matches, err := globmatch.MatchFiles(root, "**/*", 0, skipDir, skipSensitive)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range matches {
		rel, _ := filepath.Rel(root, m)
		if strings.HasPrefix(filepath.ToSlash(rel), "node_modules/") {
			t.Fatalf("walk should skip node_modules, got %q", rel)
		}
	}
	found := false
	for _, m := range matches {
		if filepath.Base(m) == "ok.go" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected ok.go in matches")
	}
}
