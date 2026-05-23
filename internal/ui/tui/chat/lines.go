package chat

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hejunqiu/ds-code/internal/ui/tui/markdown"
)

func renderUserBlock(content string, width int) []string {
	return renderHighlightedBlock(userPrompt, content, width, lipgloss.Width(userPrompt))
}

func renderAssistantBlock(content string, width int) []string {
	return markdown.RenderPrefixedBlock(assistantBullet, &styleBullet, content, width, lipgloss.Width(assistantBullet))
}

func renderHighlightedBlock(prefix, content string, width, indent int) []string {
	wrapped := markdown.WrapText(strings.TrimRight(content, "\n"), width-indent)
	if wrapped == "" {
		return nil
	}
	lines := strings.Split(wrapped, "\n")
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		p := prefix
		if i > 0 {
			p = strings.Repeat(" ", indent)
		}
		out = append(out, renderHighlightedLine(p, line, width))
	}
	return out
}

func renderAssistantLine(body string) string {
	return renderPlainPrefixedLine(assistantBullet, &styleBullet, body)
}

func renderHighlightedLine(prefix, text string, width int) string {
	return styleUserBg.Width(width).Align(lipgloss.Left).Render(prefix + text)
}

func renderPlainPrefixedLine(prefix string, prefixStyle *lipgloss.Style, body string) string {
	if prefix == "" {
		return body
	}
	if prefixStyle == nil {
		return prefix + body
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, prefixStyle.Render(prefix), body)
}

// Truncate shortens s to at most n bytes with an ellipsis suffix.
func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
