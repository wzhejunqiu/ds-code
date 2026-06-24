package glob

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/permission"
)

func TestParseGlobInput_explicitPath(t *testing.T) {
	t.Run("H1_no_path_not_explicit", func(t *testing.T) {
		in, err := parseGlobInput(json.RawMessage(`{"pattern":"*.go"}`))
		if err != nil {
			t.Fatal(err)
		}
		if in.explicitPath {
			t.Fatal("expected implicit path")
		}
	})

	t.Run("H2_empty_path_is_explicit", func(t *testing.T) {
		in, err := parseGlobInput(json.RawMessage(`{"pattern":"*.go","path":""}`))
		if err != nil {
			t.Fatal(err)
		}
		if !in.explicitPath {
			t.Fatal("expected explicit path when key present")
		}
	})

	t.Run("H3_path_value", func(t *testing.T) {
		in, err := parseGlobInput(json.RawMessage(`{"pattern":"*.go","path":"internal/pkg"}`))
		if err != nil {
			t.Fatal(err)
		}
		if !in.explicitPath || in.Path != "internal/pkg" {
			t.Fatalf("got %+v", in)
		}
	})
}

func TestValidateExplicitPath(t *testing.T) {
	dir := t.TempDir()
	perm := permission.NewEngine("readonly", dir, false)

	t.Run("H4_missing_dir", func(t *testing.T) {
		err := validateExplicitPath("no/such/dir", perm)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "目录不存在") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("H5_file_not_dir", func(t *testing.T) {
		p := filepath.Join(dir, "file.go")
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		err := validateExplicitPath("file.go", perm)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "必须是目录") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("H6_valid_dir", func(t *testing.T) {
		sub := filepath.Join(dir, "pkg")
		if err := os.Mkdir(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := validateExplicitPath("pkg", perm); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})

	t.Run("H7_unc_skips_stat", func(t *testing.T) {
		if err := validateExplicitPath(`\\server\share\foo`, perm); err != nil {
			t.Fatalf("UNC should pass through: %v", err)
		}
		if err := validateExplicitPath("//server/share/foo", perm); err != nil {
			t.Fatalf("UNC should pass through: %v", err)
		}
	})
}

func TestIsHiddenRelPath(t *testing.T) {
	cases := []struct {
		rel  string
		want bool
	}{
		{".hidden", true},
		{"foo/.bar/baz.go", true},
		{"internal/pkg/a.go", false},
	}
	for _, tc := range cases {
		if got := isHiddenRelPath(tc.rel); got != tc.want {
			t.Fatalf("isHiddenRelPath(%q) = %v want %v", tc.rel, got, tc.want)
		}
	}
}
