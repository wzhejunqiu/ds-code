package rgbin_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/grep/rgbin"
)

func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "rgbin-home-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "rgbin TestMain: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(home)
	if err := os.Setenv("HOME", home); err != nil {
		fmt.Fprintf(os.Stderr, "rgbin TestMain: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestRGBin_Path(t *testing.T) {
	t.Run("C6_first_extract_executable", func(t *testing.T) {
		path, err := rgbin.Path()
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.IsDir() {
			t.Fatal("path is directory")
		}
		if info.Mode()&0o111 == 0 {
			t.Fatal("not executable")
		}
	})

	t.Run("C6b_cached_path_skips_rehash", func(t *testing.T) {
		path1, err := rgbin.Path()
		if err != nil {
			t.Fatal(err)
		}
		path2, err := rgbin.Path()
		if err != nil {
			t.Fatal(err)
		}
		if path1 != path2 {
			t.Fatalf("path changed: %q -> %q", path1, path2)
		}
	})

	t.Run("C7_tamper_self_heal", func(t *testing.T) {
		path, err := rgbin.Path()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("tampered"), 0o700); err != nil {
			t.Fatal(err)
		}
		path2, err := rgbin.Path()
		if err != nil {
			t.Fatal(err)
		}
		if path2 != path {
			t.Fatalf("path changed: %q -> %q", path, path2)
		}
		info, err := os.Stat(path2)
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() < 1000 {
			t.Fatalf("expected restored binary, size=%d", info.Size())
		}
	})
}
