package inspect_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wzhejunqiu/ds-code/desktop/inspect"
)

func TestLanguageForPath(t *testing.T) {
	if inspect.LanguageForPath("main.go") != "go" {
		t.Fatal("expected go")
	}
	if inspect.LanguageForPath("app.ts") != "typescript" {
		t.Fatal("expected typescript")
	}
}

func TestReadFilePreview(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := inspect.ReadFilePreview(dir, "f.txt", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.Content != "b\n" {
		t.Fatalf("content = %q", res.Content)
	}
}
