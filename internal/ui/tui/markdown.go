package tui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	glamourStyles "github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
)

var (
	mdMu       sync.Mutex
	mdRenderer *glamour.TermRenderer
	mdWidth    int
)

func renderMarkdown(content string, width int) (out string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("markdown render panic: %v", r)
		}
	}()
	if width < 1 {
		width = 1
	}
	r, err := markdownRenderer(width)
	if err != nil {
		return "", err
	}
	rendered, err := r.Render(content)
	if err != nil {
		return "", err
	}
	return normalizeMarkdownOutput(rendered), nil
}

func markdownRenderer(width int) (*glamour.TermRenderer, error) {
	mdMu.Lock()
	defer mdMu.Unlock()
	if mdRenderer != nil && mdWidth == width {
		return mdRenderer, nil
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(chatMarkdownStyles()),
		glamour.WithColorProfile(lipgloss.ColorProfile()),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}
	mdRenderer = r
	mdWidth = width
	return r, nil
}

// chatMarkdownStyles tweaks the built-in light theme for chat (headings use glamour defaults).
func chatMarkdownStyles() ansi.StyleConfig {
	s := glamourStyles.LightStyleConfig
	zero := uint(0)
	s.Document.Margin = &zero
	return s
}

// normalizeMarkdownOutput trims glamour's word-wrap padding and leading newlines.
func normalizeMarkdownOutput(s string) string {
	s = strings.TrimLeft(s, "\n")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
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

func renderMarkdownPrefixedBlock(prefix string, prefixStyle lipgloss.Style, content string, width, indent int) []string {
	bodyWidth := width - indent
	if bodyWidth < 1 {
		bodyWidth = 1
	}
	trimmed := strings.TrimRight(content, "\n")
	if trimmed == "" {
		return nil
	}

	rendered, err := renderMarkdown(trimmed, bodyWidth)
	if err != nil || strings.TrimSpace(rendered) == "" {
		return renderPlainPrefixedBlock(prefix, prefixStyle, content, width, indent)
	}

	rendered = strings.TrimRight(rendered, "\n")
	lines := strings.Split(rendered, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(stripANSI(line)) == "" {
			continue
		}
		p, ps := prefix, prefixStyle
		if len(out) > 0 {
			p = strings.Repeat(" ", indent)
			ps = lipgloss.NewStyle()
		}
		out = append(out, renderPlainPrefixedLine(p, ps, line))
	}
	return out
}
