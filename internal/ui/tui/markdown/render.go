package markdown

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Render converts markdown to styled terminal text at the given width.
func Render(content string, width int) (out string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("markdown render panic: %v", r)
		}
	}()
	if width < 1 {
		width = 1
	}
	parts := splitByFences(content)
	if len(parts) == 0 {
		return "", nil
	}
	var b strings.Builder
	for _, part := range parts {
		segment := part.text
		innerWidth := width
		if part.fenced {
			segment = fencedMarkdown(part.lang, part.code)
			innerWidth = codeBlockInnerWidth(width)
		}
		rendered, err := renderSegment(segment, innerWidth)
		if err != nil {
			return "", err
		}
		if part.fenced {
			rendered = boxRenderedCodeBlock(rendered)
		}
		b.WriteString(rendered)
	}
	return normalizeOutput(b.String()), nil
}

func renderSegment(content string, width int) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", nil
	}
	r, err := markdownRenderer(width)
	if err != nil {
		return "", err
	}
	return r.Render(content)
}

// RenderPrefixedBlock renders markdown with a leading prefix on the first line.
func RenderPrefixedBlock(prefix string, prefixStyle lipgloss.Style, content string, width, indent int) []string {
	bodyWidth := width - indent
	if bodyWidth < 1 {
		bodyWidth = 1
	}
	trimmed := strings.TrimRight(content, "\n")
	if trimmed == "" {
		return nil
	}

	rendered, err := Render(trimmed, bodyWidth)
	if err != nil || strings.TrimSpace(rendered) == "" {
		return RenderPlainPrefixedBlock(prefix, prefixStyle, content, width, indent)
	}

	rendered = strings.TrimRight(rendered, "\n")
	lines := strings.Split(rendered, "\n")
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		if strings.TrimSpace(StripANSI(line)) == "" {
			continue
		}
		p, ps := prefix, prefixStyle
		if i > 0 {
			p = strings.Repeat(" ", indent)
			ps = lipgloss.NewStyle()
		}
		out = append(out, joinPrefixedLine(p, ps, line))
	}
	return out
}

// RenderPlainPrefixedBlock wraps plain text with a prefix column.
func RenderPlainPrefixedBlock(prefix string, prefixStyle lipgloss.Style, content string, width, indent int) []string {
	wrapped := WrapText(strings.TrimRight(content, "\n"), width-indent)
	if wrapped == "" {
		return nil
	}
	lines := strings.Split(wrapped, "\n")
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		p, ps := prefix, prefixStyle
		if i > 0 {
			p = strings.Repeat(" ", indent)
			ps = lipgloss.NewStyle()
		}
		out = append(out, joinPrefixedLine(p, ps, line))
	}
	return out
}

func joinPrefixedLine(prefix string, prefixStyle lipgloss.Style, body string) string {
	if prefix == "" {
		return body
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, prefixStyle.Render(prefix), body)
}

// WrapText wraps s to width (simple rune-agnostic split).
func WrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		for len(line) > width {
			out = append(out, line[:width])
			line = line[width:]
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
