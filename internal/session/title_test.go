package session

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestOneLine_collapsesNewlines(t *testing.T) {
	in := "分析\n\n文件的作用\n\n--- @client.go ---"
	got := OneLine(in)
	want := "分析 文件的作用 --- @client.go ---"
	if got != want {
		t.Fatalf("OneLine = %q, want %q", got, want)
	}
}

func TestTruncateTitle_singleLine(t *testing.T) {
	got := TruncateTitle("line1\nline2", 80)
	if strings.Contains(got, "\n") {
		t.Fatalf("TruncateTitle should be single line: %q", got)
	}
	if got != "line1 line2" {
		t.Fatalf("TruncateTitle = %q", got)
	}
}

func TestTruncateTitle_runesNotBytes(t *testing.T) {
	const han = "汉"
	s := strings.Repeat(han, 10)
	got := TruncateTitle(s, 5)
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected suffix ..., got %q", got)
	}
	if strings.Count(got, han) != 2 {
		t.Fatalf("TruncateTitle = %q, want 2 han chars + ...", got)
	}
	if len([]rune(got)) != 5 {
		t.Fatalf("TruncateTitle length = %d runes, want 5", len([]rune(got)))
	}
	if !utf8.ValidString(got) {
		t.Fatalf("invalid UTF-8: %q", got)
	}
}
