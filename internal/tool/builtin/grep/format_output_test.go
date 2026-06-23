package grep

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/tool/builtin"
)

func TestFormatGrepOutput_filesWithMatches(t *testing.T) {
	t.Run("A1_single_file", func(t *testing.T) {
		got := formatGrepOutput(builtin.GrepOutputFilesWithMatches, []string{"internal/foo.go"}, grepPageMeta{
			TotalFiles: 1, TotalEntries: 1,
		})
		want := "Found 1 files\ninternal/foo.go"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("A2_multiple_files", func(t *testing.T) {
		got := formatGrepOutput(builtin.GrepOutputFilesWithMatches, []string{"b.go", "a.go"}, grepPageMeta{
			TotalFiles: 2, TotalEntries: 2,
		})
		want := "Found 2 files\nb.go\na.go"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("A3_no_match", func(t *testing.T) {
		got := formatGrepOutput(builtin.GrepOutputFilesWithMatches, nil, grepPageMeta{})
		if got != "Found 0 files" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("A4_truncated", func(t *testing.T) {
		got := formatGrepOutput(builtin.GrepOutputFilesWithMatches, []string{"a.go", "b.go"}, grepPageMeta{
			Limit: 2, Offset: 0, TotalFiles: 5, TotalEntries: 5,
		})
		want := "Found 5 files\na.go\nb.go\n[Showing results with pagination = limit: 2, offset: 0]"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("A5_offset", func(t *testing.T) {
		got := formatGrepOutput(builtin.GrepOutputFilesWithMatches, []string{"c.go"}, grepPageMeta{
			Limit: 1, Offset: 2, TotalFiles: 3, TotalEntries: 3,
		})
		want := "Found 3 files\nc.go\n[Showing results with pagination = limit: 1, offset: 2]"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("A6_offset_and_limit", func(t *testing.T) {
		got := formatGrepOutput(builtin.GrepOutputFilesWithMatches, []string{"b.go"}, grepPageMeta{
			Limit: 1, Offset: 1, TotalFiles: 3, TotalEntries: 3,
		})
		want := "Found 3 files\nb.go\n[Showing results with pagination = limit: 1, offset: 1]"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("A7_head_limit_zero_no_footer", func(t *testing.T) {
		got := formatGrepOutput(builtin.GrepOutputFilesWithMatches, []string{"a.go", "b.go"}, grepPageMeta{
			Limit: 0, Offset: 0, TotalFiles: 2, TotalEntries: 2,
		})
		want := "Found 2 files\na.go\nb.go"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
}

func TestFormatGrepOutput_content(t *testing.T) {
	t.Run("A8_match_only", func(t *testing.T) {
		body := []string{
			"internal/foo.go:11:func main() {",
			"internal/bar.go:3:func helper() {",
		}
		got := formatGrepOutput(builtin.GrepOutputContent, body, grepPageMeta{TotalEntries: 2})
		want := strings.Join(body, "\n")
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("A9_match_and_context", func(t *testing.T) {
		body := []string{
			"internal/foo.go:10-import \"fmt\"",
			"internal/foo.go:11:func main() {",
			"internal/foo.go:12-    fmt.Println(\"hi\")",
		}
		got := formatGrepOutput(builtin.GrepOutputContent, body, grepPageMeta{TotalEntries: 3})
		want := strings.Join(body, "\n")
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("A10_no_line_numbers", func(t *testing.T) {
		got := formatGrepLine(recordMatch, "foo.go", 11, "text", false)
		if got != "foo.go:text" {
			t.Fatalf("got %q", got)
		}
		got = formatGrepLine(recordContext, "foo.go", 10, "ctx", false)
		if got != "foo.go-ctx" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("A11_no_match", func(t *testing.T) {
		got := formatGrepOutput(builtin.GrepOutputContent, nil, grepPageMeta{})
		if got != "" {
			t.Fatalf("got %q want empty", got)
		}
	})

	t.Run("A12_truncated", func(t *testing.T) {
		body := []string{"a:1:x", "a:2:x"}
		got := formatGrepOutput(builtin.GrepOutputContent, body, grepPageMeta{
			Limit: 2, Offset: 0, TotalEntries: 5,
		})
		want := "a:1:x\na:2:x\n[Showing results with pagination = limit: 2, offset: 0]"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("A13_offset", func(t *testing.T) {
		body := []string{"line3"}
		got := formatGrepOutput(builtin.GrepOutputContent, body, grepPageMeta{
			Limit: 1, Offset: 2, TotalEntries: 5,
		})
		want := "line3\n[Showing results with pagination = limit: 1, offset: 2]"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
}

func TestFormatGrepOutput_count(t *testing.T) {
	t.Run("A14_single_file", func(t *testing.T) {
		got := formatGrepOutput(builtin.GrepOutputCount, []string{"internal/foo.go:3"}, grepPageMeta{
			TotalFiles: 1, TotalMatches: 3, TotalEntries: 1,
		})
		want := "internal/foo.go:3\nFound 3 occurrences across 1 files"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("A15_multiple_files", func(t *testing.T) {
		got := formatGrepOutput(builtin.GrepOutputCount, []string{"internal/foo.go:3", "pkg/bar.go:1"}, grepPageMeta{
			TotalFiles: 2, TotalMatches: 4, TotalEntries: 2,
		})
		want := "internal/foo.go:3\npkg/bar.go:1\nFound 4 occurrences across 2 files"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("A16_no_match", func(t *testing.T) {
		got := formatGrepOutput(builtin.GrepOutputCount, nil, grepPageMeta{})
		if got != "Found 0 occurrences across 0 files" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("A17_truncated", func(t *testing.T) {
		got := formatGrepOutput(builtin.GrepOutputCount, []string{"a.go:3", "b.go:1"}, grepPageMeta{
			Limit: 2, Offset: 0, TotalFiles: 10, TotalMatches: 42, TotalEntries: 10,
		})
		want := "a.go:3\nb.go:1\nFound 42 occurrences across 10 files\n[Showing results with pagination = limit: 2, offset: 0]"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("A18_offset", func(t *testing.T) {
		got := formatGrepOutput(builtin.GrepOutputCount, []string{"b.go:1"}, grepPageMeta{
			Limit: 1, Offset: 1, TotalFiles: 10, TotalMatches: 42, TotalEntries: 10,
		})
		want := "b.go:1\nFound 42 occurrences across 10 files\n[Showing results with pagination = limit: 1, offset: 1]"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
}

func TestFormatPaginationFooter(t *testing.T) {
	t.Run("A19_exact_format", func(t *testing.T) {
		got := formatPaginationFooter(250, 0)
		want := "[Showing results with pagination = limit: 250, offset: 0]"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("A20_no_footer_when_complete", func(t *testing.T) {
		got := formatGrepOutput(builtin.GrepOutputFilesWithMatches, []string{"a.go"}, grepPageMeta{
			Limit: 250, Offset: 0, TotalFiles: 1, TotalEntries: 1,
		})
		want := "Found 1 files\na.go"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
}
