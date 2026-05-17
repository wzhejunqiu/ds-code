package tui

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestRenderMarkdownHeadingsStyled(t *testing.T) {
	content := "# Title\n\n## Subtitle\n\n### Section\n\nbody"
	out, err := renderMarkdown(content, 60)
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"# Title", "## Subtitle", "### Section"} {
		if strings.Contains(out, marker) {
			t.Fatalf("raw heading marker %q in output:\n%s", marker, out)
		}
	}
	for _, text := range []string{"Title", "Subtitle", "Section", "body"} {
		if !headingTextPresent(out, text) {
			t.Fatalf("missing %q in output:\n%s", text, out)
		}
	}
	if strings.Contains(out, "\x1b[;;") {
		t.Fatalf("invalid SGR from H1 background banner:\n%s", out)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI styling in output:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Fatalf("trailing padding on line: %q", line)
		}
	}
}

func TestRenderMarkdownAllHeadingLevelsBold(t *testing.T) {
	for level := 1; level <= 6; level++ {
		md := strings.Repeat("#", level) + " H\n"
		out, err := renderMarkdown(md, 60)
		if err != nil {
			t.Fatalf("level %d: %v", level, err)
		}
		line := headingLine(out, "H", false)
		if line == "" {
			t.Fatalf("level %d: missing heading line in:\n%s", level, out)
		}
		if !hasANSIBold(line) {
			t.Fatalf("level %d: want bold:\n%s", level, line)
		}
	}
}

func TestRenderMarkdownHeadingSizeTiers(t *testing.T) {
	const title = "标题"
	var widths []int
	for level := 1; level <= 6; level++ {
		md := strings.Repeat("#", level) + " " + title + "\n"
		out, err := renderMarkdown(md, 80)
		if err != nil {
			t.Fatal(err)
		}
		line := headingLine(out, title, true)
		widths = append(widths, runewidth.StringWidth(stripANSI(line)))
	}
	if widths[0] <= widths[1] || widths[1] <= widths[2] {
		t.Fatalf("H1–H3 width should decrease: %v", widths)
	}
	if widths[0] <= widths[5] {
		t.Fatalf("H1 should be wider than H6: %v", widths)
	}
	// H4–H6 share normal cell width (no extra letter-spacing).
	for i := 3; i < 6; i++ {
		if widths[i] != widths[3] {
			t.Fatalf("H4–H6 should match normal body width: %v", widths)
		}
	}
}

func TestRenderAssistantMarkdownHeadingsNoPaddingLines(t *testing.T) {
	content := "# Title\n\n## Subtitle\n\n### Section\n\nbody"
	lines := renderAssistantBlock(content, 60)
	for _, line := range lines {
		if strings.TrimSpace(stripANSI(line)) == "" {
			t.Fatalf("empty padded line in output:\n%s", strings.Join(lines, "\n"))
		}
	}
}

func headingTextPresent(out, text string) bool {
	plain := strings.ReplaceAll(stripANSI(out), " ", "")
	compact := strings.ReplaceAll(text, " ", "")
	return strings.Contains(plain, compact)
}

func headingLine(out, substr string, ignoreSpaces bool) string {
	for _, l := range strings.Split(out, "\n") {
		plain := stripANSI(l)
		if ignoreSpaces {
			plain = strings.ReplaceAll(plain, " ", "")
			substr = strings.ReplaceAll(substr, " ", "")
		}
		if strings.Contains(plain, substr) {
			return l
		}
	}
	return ""
}

func hasANSIBold(s string) bool {
	return strings.Contains(s, "\x1b[1m") || strings.Contains(s, ";1m")
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
