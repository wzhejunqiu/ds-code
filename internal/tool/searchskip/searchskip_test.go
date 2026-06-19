package searchskip_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool/searchskip"
)

func TestMatcherSkipGit(t *testing.T) {
	m := searchskip.New(nil)
	if !m.SkipDir(".git") {
		t.Fatal("expected .git skipped")
	}
	if m.SkipDir("src") {
		t.Fatal("src should not be skipped by default")
	}
}

func TestMatcherSkipDirs(t *testing.T) {
	m := searchskip.New([]string{"node_modules", "target"})
	if !m.SkipDir("node_modules") || !m.Ignored("node_modules/foo") {
		t.Fatal("node_modules should be skipped")
	}
	if !m.SkipDir("target") {
		t.Fatal("target should be skipped")
	}
}

func TestIgnoredInScope_explicitSkipDirPath(t *testing.T) {
	m := searchskip.New([]string{"node_modules"})
	if m.IgnoredInScope("node_modules/foo/bar.js", "node_modules") {
		t.Fatal("explicit path=node_modules should not filter node_modules/*")
	}
	if m.IgnoredInScope("node_modules/pkg/index.js", "node_modules/pkg") {
		t.Fatal("explicit path=node_modules/pkg should not filter under it")
	}
	if !m.IgnoredInScope("node_modules/foo.js", ".") {
		t.Fatal("wide scope should still filter skip_dirs matches")
	}
}

func TestIgnoredInScope_gitAlwaysFiltered(t *testing.T) {
	m := searchskip.New([]string{"node_modules"})
	if !m.IgnoredInScope(".git/HEAD", "node_modules") {
		t.Fatal(".git must always be filtered even with explicit skip_dir scope")
	}
	if !m.IgnoredInScope(".git/config", ".git") {
		t.Fatal(".git must always be filtered even when path=.git")
	}
}

func TestSearchSkip_invalidConfigEntries(t *testing.T) {
	m := searchskip.New([]string{"../outside", "/etc", "", "node_modules"})
	if !m.SkipDir("node_modules") {
		t.Fatal("valid skip_dirs entry should still apply")
	}
	if !m.IgnoredInScope("node_modules/pkg/index.js", ".") {
		t.Fatal("valid skip_dirs should filter under wide scope")
	}
}
