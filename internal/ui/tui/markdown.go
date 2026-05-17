package tui

import (
	"regexp"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	glamourStyles "github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	mdMu       sync.Mutex
	mdRenderer *glamour.TermRenderer
	mdWidth    int

	atxHeadingLine = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)
)

type markdownPart struct {
	kind  markdownPartKind
	level int
	text  string
}

func renderMarkdown(content string, width int) (string, error) {
	if width < 1 {
		width = 1
	}
	parts := splitMarkdownParts(content)
	var b strings.Builder
	for i, p := range parts {
		switch p.kind {
		case markdownPartHeading:
			h := renderChatHeading(p.level, p.text, width)
			if h == "" {
				continue
			}
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
				b.WriteByte('\n')
			}
			if i > 0 && p.level == 1 {
				b.WriteByte('\n')
			}
			b.WriteString(h)
			b.WriteByte('\n')
		case markdownPartBody:
			text := strings.TrimSpace(p.text)
			if text == "" {
				continue
			}
			body, err := renderGlamourBody(text, width)
			if err != nil {
				return "", err
			}
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
				b.WriteByte('\n')
			}
			b.WriteString(body)
		}
	}
	return normalizeMarkdownOutput(b.String()), nil
}

func splitMarkdownParts(content string) []markdownPart {
	var parts []markdownPart
	var body strings.Builder
	flushBody := func() {
		if body.Len() == 0 {
			return
		}
		parts = append(parts, markdownPart{kind: markdownPartBody, text: body.String()})
		body.Reset()
	}

	for _, line := range strings.Split(content, "\n") {
		if m := atxHeadingLine.FindStringSubmatch(line); m != nil {
			flushBody()
			parts = append(parts, markdownPart{
				kind:  markdownPartHeading,
				level: len(m[1]),
				text:  m[2],
			})
			continue
		}
		if body.Len() > 0 {
			body.WriteByte('\n')
		}
		body.WriteString(line)
	}
	flushBody()
	return parts
}

// renderChatHeading renders ATX headings. Monospace terminals cannot change font
// size; we simulate H1 (largest) → H6 (normal width) via letter-spacing and bold.
func renderChatHeading(level int, title string, width int) string {
	title = strings.TrimSpace(title)
	if title == "" || level < 1 || level > 6 {
		return ""
	}
	spacing := headingLetterSpacing(level)
	var lines []string
	for _, line := range strings.Split(wrapText(title, width), "\n") {
		if line == "" {
			continue
		}
		if spacing > 0 {
			line = widenRunes(line, spacing)
		}
		lines = append(lines, styleHeadingLine(line))
	}
	return strings.Join(lines, "\n")
}

func styleHeadingLine(line string) string {
	return termenv.String(line).Bold().String()
}

// headingLetterSpacing returns extra spaces inserted between runes (H1 widest → H6 none).
func headingLetterSpacing(level int) int {
	switch level {
	case 1:
		return 2
	case 2:
		return 1
	default:
		return 0
	}
}

func widenRunes(s string, gap int) string {
	if gap <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= 1 {
		return s
	}
	sep := strings.Repeat(" ", gap)
	var b strings.Builder
	for i, r := range runes {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteRune(r)
	}
	return b.String()
}

func renderGlamourBody(content string, width int) (string, error) {
	r, err := markdownRenderer(width)
	if err != nil {
		return "", err
	}
	rendered, err := r.Render(content)
	if err != nil {
		return "", err
	}
	return rendered, nil
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

// chatMarkdownStyles configures glamour for non-heading markdown (lists, code, etc.).
func chatMarkdownStyles() ansi.StyleConfig {
	s := glamourStyles.LightStyleConfig

	zero := uint(0)
	s.Document.Margin = &zero

	s.Heading.Bold = boolPtr(true)
	s.Heading.Color = nil
	s.Heading.BackgroundColor = nil

	for _, level := range []*ansi.StyleBlock{&s.H1, &s.H2, &s.H3, &s.H4, &s.H5, &s.H6} {
		level.Prefix = ""
		level.Suffix = ""
		level.Color = nil
		level.BackgroundColor = nil
		level.Margin = &zero
		level.Bold = boolPtr(true)
	}

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

func boolPtr(b bool) *bool { return &b }

func stringPtr(s string) *string { return &s }

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
		if strings.TrimSpace(line) == "" {
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
