//go:build tuitest

package tuitest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStackClose_removesProjectDataDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	s, err := newStackCore()
	if err != nil {
		t.Fatal(err)
	}
	dataDir := s.Cfg.ProjectDataDir
	if dataDir == "" {
		t.Fatal("expected ProjectDataDir")
	}
	if _, err := os.Stat(dataDir); err != nil {
		t.Fatalf("project data dir missing before close: %v", err)
	}
	if _, err := os.Stat(s.Project); err != nil {
		t.Fatalf("project workspace missing before close: %v", err)
	}

	s.Close()

	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Fatalf("project data dir still exists after close: %s", dataDir)
	}
	if _, err := os.Stat(s.Project); !os.IsNotExist(err) {
		t.Fatal("project workspace still exists after close")
	}
}

func TestNewHarness_usesIsolatedHome(t *testing.T) {
	s, err := NewHarness()
	if err != nil {
		t.Fatal(err)
	}
	if s.homeDir == "" {
		t.Fatal("expected isolated homeDir")
	}
	dataDir := s.Cfg.ProjectDataDir
	wantPrefix := filepath.Join(s.homeDir, ".ds-code", "projects")
	if dataDir == "" || !filepath.HasPrefix(dataDir, wantPrefix) {
		t.Fatalf("ProjectDataDir = %q, want under %q", dataDir, wantPrefix)
	}

	s.Close()

	if _, err := os.Stat(s.homeDir); !os.IsNotExist(err) {
		t.Fatalf("harness home still exists: %s", s.homeDir)
	}
}
