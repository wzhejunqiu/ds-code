package selection_test

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/ui/tui/selection"
)

func TestStripANSI(t *testing.T) {
	in := "\x1b[31mhello\x1b[0m world"
	got := selection.StripANSI(in)
	if got != "hello world" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractSingleLine(t *testing.T) {
	lines := []string{"abcdef"}
	r := selection.Range{
		Start: selection.Point{Line: 0, Col: 1},
		End:   selection.Point{Line: 0, Col: 4},
	}
	got := selection.Extract(lines, r)
	if got != "bcd" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractMultiLine(t *testing.T) {
	lines := []string{"ab", "cd", "ef"}
	r := selection.Range{
		Start: selection.Point{Line: 0, Col: 1},
		End:   selection.Point{Line: 2, Col: 1},
	}
	got := selection.Extract(lines, r)
	want := "b\ncd\ne"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestHighlightLines_singleAndMultiLine(t *testing.T) {
	lines := []string{"abcdef", "ghij", "klmno"}
	r := selection.Range{
		Start: selection.Point{Line: 0, Col: 1},
		End:   selection.Point{Line: 2, Col: 2},
	}
	got := selection.HighlightLines(lines, r)
	plain := selection.LinesFromContent(selection.StripANSI(got))
	extracted := selection.Extract(plain, r)
	want := selection.Extract(lines, r)
	if extracted != want {
		t.Fatalf("highlighted extract = %q want %q (full %q)", extracted, want, got)
	}
	if !strings.Contains(got, "bcd") {
		t.Fatalf("missing selected segment in %q", got)
	}
}
