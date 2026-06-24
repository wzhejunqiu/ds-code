package glob

import (
	"testing"
)

// Output format tests (G1–G7) for glob-specific truncation (not grep pagination).

func TestFormatGlobOutput(t *testing.T) {
	t.Run("G1_single_file", func(t *testing.T) {
		got := formatGlobOutput([]string{"internal/foo.go"}, 1, 100)
		want := "Found 1 files\ninternal/foo.go"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("G2_multiple_files", func(t *testing.T) {
		got := formatGlobOutput([]string{"b.go", "a.go"}, 2, 100)
		want := "Found 2 files\nb.go\na.go"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("G3_no_match", func(t *testing.T) {
		got := formatGlobOutput(nil, 0, 100)
		if got != "Found 0 files" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("G4_truncated", func(t *testing.T) {
		got := formatGlobOutput([]string{"a.go", "b.go"}, 5, 2)
		want := "Found 2 files\na.go\nb.go\n（结果已截断，请使用更具体的 path 或 pattern）"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("G5_limit_zero_no_footer", func(t *testing.T) {
		got := formatGlobOutput([]string{"a.go", "b.go"}, 2, 0)
		want := "Found 2 files\na.go\nb.go"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("G6_workspace_relative_paths", func(t *testing.T) {
		got := formatGlobOutput([]string{"internal/pkg/a.go"}, 1, 100)
		if got != "Found 1 files\ninternal/pkg/a.go" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("G7_no_leading_dot_slash", func(t *testing.T) {
		paths := []string{"foo.go"}
		got := formatGlobOutput(paths, 1, 100)
		if paths[0] != "foo.go" {
			t.Fatalf("path mutated")
		}
		if got != "Found 1 files\nfoo.go" {
			t.Fatalf("got %q", got)
		}
	})
}
