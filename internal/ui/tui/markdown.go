package tui

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	glamourStyles "github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/hejunqiu/ds-code/internal/ui/theme"
	"github.com/muesli/termenv"
)

var (
	mdMu       sync.Mutex
	mdRenderer *glamour.TermRenderer
	mdWidth    int
	mdProfile  termenv.Profile

	// codeBlockBoxStyle frames fenced code blocks in the chat pane.
	codeBlockBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.Divider).
				Padding(0, 1)
)

type markdownPart struct {
	text   string
	fenced bool
	lang   string
	code   string
}

func renderMarkdown(content string, width int) (out string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("markdown render panic: %v", r)
		}
	}()
	if width < 1 {
		width = 1
	}
	parts := splitMarkdownByFences(content)
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
		rendered, err := renderMarkdownSegment(segment, innerWidth)
		if err != nil {
			return "", err
		}
		if part.fenced {
			rendered = boxRenderedCodeBlock(rendered)
		}
		b.WriteString(rendered)
	}
	return normalizeMarkdownOutput(b.String()), nil
}

func renderMarkdownSegment(content string, width int) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", nil
	}
	r, err := markdownRenderer(width)
	if err != nil {
		return "", err
	}
	return r.Render(content)
}

func markdownRenderer(width int) (*glamour.TermRenderer, error) {
	mdMu.Lock()
	defer mdMu.Unlock()
	profile := lipgloss.ColorProfile()
	if mdRenderer != nil && mdWidth == width && mdProfile == profile {
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
	mdProfile = profile
	return r, nil
}

// chatMarkdownStyles tweaks the built-in light theme for chat.
// Glamour's default light theme decorates H2–H6 with literal "## " prefixes; we replace
// those with typography-only styles (bold / underline / color) suited to the chat pane.
func chatMarkdownStyles() ansi.StyleConfig {
	s := glamourStyles.LightStyleConfig
	zero := uint(0)
	s.Document.Margin = &zero

	deepSeek := string(theme.DeepSeek)
	text := string(theme.Text)
	muted := string(theme.Muted)

	s.Heading = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix: "\n",
			Color:       &deepSeek,
			Bold:        mdBool(true),
		},
	}
	s.H1 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:     &deepSeek,
			Bold:      mdBool(true),
			Underline: mdBool(true),
		},
	}
	s.H2 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:     &deepSeek,
			Bold:      mdBool(true),
			Underline: mdBool(true),
		},
	}
	s.H3 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &deepSeek,
			Bold:  mdBool(true),
		},
	}
	s.H4 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &text,
			Bold:  mdBool(true),
		},
	}
	s.H5 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &text,
			Bold:  mdBool(false),
		},
	}
	s.H6 = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &muted,
			Bold:  mdBool(false),
		},
	}
	s.CodeBlock = ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: &text,
			},
			Margin: &zero,
		},
		Chroma: codeBlockChromaStyles(),
	}
	return s
}

// codeBlockChromaStyles returns syntax colors without per-token backgrounds.
func codeBlockChromaStyles() *ansi.Chroma {
	c := glamourStyles.LightStyleConfig.CodeBlock.Chroma
	if c == nil {
		text := string(theme.Text)
		return &ansi.Chroma{
			Text: ansi.StylePrimitive{Color: &text},
		}
	}
	ch := *c
	clearChromaBackgrounds(&ch)
	errColor := string(theme.Error)
	ch.Error = ansi.StylePrimitive{Color: &errColor}
	return &ch
}

func clearChromaBackgrounds(c *ansi.Chroma) {
	v := reflect.ValueOf(c).Elem()
	primType := reflect.TypeOf(ansi.StylePrimitive{})
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Type() != primType {
			continue
		}
		sp := f.Addr().Interface().(*ansi.StylePrimitive)
		sp.BackgroundColor = nil
	}
}

func mdBool(v bool) *bool { return &v }

func splitMarkdownByFences(content string) []markdownPart {
	if content == "" {
		return nil
	}
	var parts []markdownPart
	rest := content
	for {
		start := strings.Index(rest, "```")
		if start < 0 {
			if rest != "" {
				parts = append(parts, markdownPart{text: rest})
			}
			break
		}
		if start > 0 {
			parts = append(parts, markdownPart{text: rest[:start]})
		}
		afterOpen := rest[start+3:]
		langLine, codeBody, ok := strings.Cut(afterOpen, "\n")
		if !ok {
			parts = append(parts, markdownPart{text: rest[start:]})
			break
		}
		lang := strings.TrimSpace(langLine)
		closeAt := findClosingFenceLine(codeBody)
		if closeAt < 0 {
			parts = append(parts, markdownPart{text: rest[start:]})
			break
		}
		code := codeBody[:closeAt]
		if strings.HasSuffix(code, "\n") {
			code = strings.TrimSuffix(code, "\n")
		}
		parts = append(parts, markdownPart{fenced: true, lang: lang, code: code})
		rest = codeBody[closeAt:]
		if afterClose, ok := strings.CutPrefix(rest, "```"); ok {
			rest = afterClose
			if nl := strings.Index(rest, "\n"); nl >= 0 {
				rest = rest[nl+1:]
			} else {
				rest = ""
			}
		}
	}
	return parts
}

// findClosingFenceLine returns the byte index of a line that is only ``` (optional spaces).
func findClosingFenceLine(s string) int {
	offset := 0
	for {
		nl := strings.Index(s[offset:], "\n")
		var line string
		lineStart := offset
		if nl < 0 {
			line = s[offset:]
			offset = len(s)
		} else {
			line = s[offset : offset+nl]
			offset += nl + 1
		}
		if strings.TrimSpace(line) == "```" {
			return lineStart
		}
		if nl < 0 {
			break
		}
	}
	return -1
}

func fencedMarkdown(lang, code string) string {
	if lang == "" {
		return "```\n" + code + "\n```"
	}
	return "```" + lang + "\n" + code + "\n```"
}

// codeBlockInnerWidth is the glamour wrap width inside the rounded border (border + padding).
func codeBlockInnerWidth(outer int) int {
	const frame = 4
	if outer <= frame {
		return 1
	}
	return outer - frame
}

func boxRenderedCodeBlock(rendered string) string {
	lines := trimEmptyMarkdownLines(strings.Split(strings.TrimRight(rendered, "\n"), "\n"))
	if len(lines) == 0 {
		return ""
	}
	return codeBlockBoxStyle.Render(strings.Join(lines, "\n")) + "\n"
}

func trimEmptyMarkdownLines(lines []string) []string {
	out := lines[:0]
	for _, line := range lines {
		if strings.TrimSpace(stripANSI(line)) == "" {
			continue
		}
		out = append(out, strings.TrimRight(line, " "))
	}
	return out
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
	return xansi.Strip(s)
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
