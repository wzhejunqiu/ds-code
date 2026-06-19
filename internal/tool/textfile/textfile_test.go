package textfile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool/textfile"
)

func TestIsSearchable_textFiles(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "hello.go")
	if err := os.WriteFile(goFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mdFile := filepath.Join(dir, "readme.md")
	if err := os.WriteFile(mdFile, []byte("# hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !textfile.IsSearchable(goFile) {
		t.Fatal("hello.go should be searchable")
	}
	if !textfile.IsSearchable(mdFile) {
		t.Fatal("readme.md should be searchable")
	}
}

func TestIsSearchable_pngMagic(t *testing.T) {
	dir := t.TempDir()
	png := filepath.Join(dir, "fake.png")
	data := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	data = append(data, make([]byte, 32)...)
	if err := os.WriteFile(png, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if textfile.IsSearchable(png) {
		t.Fatal("png should not be searchable")
	}
}

func TestIsSearchable_nulByte(t *testing.T) {
	dir := t.TempDir()
	weird := filepath.Join(dir, "weird.txt")
	if err := os.WriteFile(weird, []byte("hello\x00world"), 0o644); err != nil {
		t.Fatal(err)
	}
	if textfile.IsSearchable(weird) {
		t.Fatal("file with NUL should not be searchable")
	}
}

func TestIsSearchable_blockedExt(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "data.pdf")
	if err := os.WriteFile(f, []byte("not really pdf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if textfile.IsSearchable(f) {
		t.Fatal(".pdf extension should be blocked without reading")
	}
}

func TestIsTextFile_matchesIsSearchable(t *testing.T) {
	dir := t.TempDir()
	textPath := filepath.Join(dir, "hello.go")
	if err := os.WriteFile(textPath, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pngPath := filepath.Join(dir, "fake.png")
	pngData := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	pngData = append(pngData, make([]byte, 32)...)
	if err := os.WriteFile(pngPath, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{textPath, pngPath} {
		if textfile.IsTextFile(path) != textfile.IsSearchable(path) {
			t.Fatalf("IsTextFile(%q) != IsSearchable for same path", path)
		}
	}
}
