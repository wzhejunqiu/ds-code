package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}

func TestRenderMarkdownHeadingsStyled(t *testing.T) {
	content := "# Title\n\n## Subtitle\n\n### Section\n\nbody"
	out, err := renderMarkdown(content, 60)
	if err != nil {
		t.Fatal(err)
	}
	plain := stripANSI(out)
	for _, marker := range []string{"# ", "## ", "### ", "#### "} {
		if strings.Contains(plain, marker) {
			t.Fatalf("raw heading marker %q in output:\n%s", marker, plain)
		}
	}
	for _, text := range []string{"Title", "Subtitle", "Section", "body"} {
		if !headingTextPresent(out, text) {
			t.Fatalf("missing %q in output:\n%s", text, out)
		}
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Fatalf("trailing padding on line: %q", line)
		}
	}
}

func TestRenderMarkdownHeadingEmojiTitle(t *testing.T) {
	content := "# 📋 ds-code 代码变更审查报告\n\nbody"
	out, err := renderMarkdown(content, 80)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stripANSI(out), "# ") {
		t.Fatalf("raw heading marker in output:\n%s", stripANSI(out))
	}
	if !headingTextPresent(out, "ds-code") || !headingTextPresent(out, "代码变更审查报告") {
		t.Fatalf("missing heading text:\n%s", out)
	}
}

func TestRenderMarkdownCodeBlockNoBackgroundANSI(t *testing.T) {
	content := "```go\nfunc foo() { x := \"hi\" }\n```"
	out, err := renderMarkdown(content, 70)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "48;") {
		t.Fatalf("unexpected background SGR in code block:\n%s", out)
	}
}

func TestRenderMarkdownCodeBlockBorder(t *testing.T) {
	content := "text\n\n```go\nfmt.Println(\"hi\")\n```\n\nmore"
	out, err := renderMarkdown(content, 60)
	if err != nil {
		t.Fatal(err)
	}
	plain := stripANSI(out)
	if !strings.Contains(plain, "Println") || !strings.Contains(plain, "hi") {
		t.Fatalf("missing code content:\n%s", plain)
	}
	for _, border := range []string{"╭", "╮", "╰", "╯", "│"} {
		if !strings.Contains(plain, border) {
			t.Fatalf("expected code block border %q in output:\n%s", border, plain)
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
