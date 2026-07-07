package datadir_test

import (
	"path/filepath"
	"strings"
	"testing"

	desktopdatadir "github.com/wzhejunqiu/ds-code/desktop/datadir"
	"github.com/wzhejunqiu/ds-code/internal/datadir"
)

func TestProjectIDMatchesCLI(t *testing.T) {
	root := "/tmp/my-project"
	if desktopdatadir.ProjectID(root) != datadir.ProjectID(root) {
		t.Fatal("ProjectID must match CLI algorithm")
	}
}

func TestProjectDataDirUnderDesktopPrefix(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	dir, err := desktopdatadir.ProjectDataDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dir, string(filepath.Separator)+"desktop"+string(filepath.Separator)+"projects"+string(filepath.Separator)) {
		t.Fatalf("expected desktop/projects path, got %q", dir)
	}
	cliDir, err := datadir.ProjectDataDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if dir == cliDir {
		t.Fatalf("desktop dir must differ from CLI dir: %q", dir)
	}
}

func TestDefaultDBPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	path := desktopdatadir.DefaultDBPath(root)
	if path == "" {
		t.Fatal("expected non-empty db path")
	}
	if !strings.HasSuffix(path, "sessions.db") {
		t.Fatalf("expected sessions.db suffix, got %q", path)
	}
	if !strings.Contains(path, "desktop") {
		t.Fatalf("expected desktop prefix in path, got %q", path)
	}
}
