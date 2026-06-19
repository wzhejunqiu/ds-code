package resultstore_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/mcp/resultstore"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
)

func TestSpillCallFilename_sanitizes(t *testing.T) {
	// spillCallFilename is not exported; test via Save with predictable call id.
	testutil.IsolatedHome(t)
	root := t.TempDir()
	store := &resultstore.Store{ProjectRoot: root}
	path, err := store.Save("sess-1", "call/foo", "body")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "call_foo.txt") {
		t.Fatalf("path = %q want call_foo.txt suffix", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "body" {
		t.Fatalf("content = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o want 0600", info.Mode().Perm())
	}
}

func TestSave_emptyCallIDUsesULID(t *testing.T) {
	testutil.IsolatedHome(t)
	root := t.TempDir()
	store := &resultstore.Store{ProjectRoot: root}
	p1, err := store.Save("sess-1", "", "a")
	if err != nil {
		t.Fatal(err)
	}
	p2, err := store.Save("sess-1", "", "b")
	if err != nil {
		t.Fatal(err)
	}
	if p1 == p2 {
		t.Fatal("empty call ids should produce distinct spill files")
	}
}

func TestSave_overwritesSameCallID(t *testing.T) {
	testutil.IsolatedHome(t)
	root := t.TempDir()
	store := &resultstore.Store{ProjectRoot: root}
	p1, _ := store.Save("sess-1", "call_abc", "first")
	p2, _ := store.Save("sess-1", "call_abc", "second")
	if p1 != p2 {
		t.Fatalf("expected same path %q vs %q", p1, p2)
	}
	data, _ := os.ReadFile(p2)
	if string(data) != "second" {
		t.Fatalf("got %q", data)
	}
	dir := filepath.Dir(p1)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode = %o want 0700", info.Mode().Perm())
	}
}
