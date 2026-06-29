package datadir

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/version"
)

func TestComputeIdentifier_deterministic(t *testing.T) {
	got := computeIdentifier("550e8400-e29b-41d4-a716-446655440000", "alice")
	sum := sha256.Sum256([]byte("550e8400-e29b-41d4-a716-446655440000" + "alice" + version.Name))
	want := hex.EncodeToString(sum[:])
	if got != want {
		t.Fatalf("computeIdentifier() = %q, want %q", got, want)
	}
	if len(got) != 64 {
		t.Fatalf("length = %d, want 64", len(got))
	}
}

func TestValidIdentifier(t *testing.T) {
	sum := sha256.Sum256([]byte("x"))
	valid := hex.EncodeToString(sum[:])
	if !validIdentifier(valid) {
		t.Fatal("expected valid hex identifier")
	}
	if validIdentifier("not-hex") || validIdentifier("abc") {
		t.Fatal("expected invalid identifier")
	}
}

func isolatedHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	resetIdentifierForTest()
	return dir
}

func TestIdentifier_createsAndReuses(t *testing.T) {
	home := isolatedHome(t)

	id1 := Identifier()
	if id1 == "" {
		t.Fatal("expected non-empty identifier")
	}
	if !validIdentifier(id1) {
		t.Fatalf("invalid identifier %q", id1)
	}
	path := filepath.Join(home, version.UserDataDirName, "identifier")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != id1+"\n" {
		t.Fatalf("file = %q, want %q", string(raw), id1+"\n")
	}

	id2 := Identifier()
	if id1 != id2 {
		t.Fatalf("id1 = %q, id2 = %q", id1, id2)
	}
}

func TestIdentifier_readsExistingFile(t *testing.T) {
	home := isolatedHome(t)

	want := computeIdentifier("fixed-uuid", "bob")
	path := filepath.Join(home, version.UserDataDirName)
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "identifier"), []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := Identifier(); got != want {
		t.Fatalf("Identifier() = %q, want %q", got, want)
	}
}

func TestIdentifier_differentHomesDiffer(t *testing.T) {
	home1 := isolatedHome(t)
	id1 := Identifier()

	dir2 := t.TempDir()
	t.Setenv("HOME", dir2)
	resetIdentifierForTest()
	id2 := Identifier()

	if id1 == id2 {
		t.Fatalf("expected different identifiers for %q vs %q", home1, dir2)
	}
}
