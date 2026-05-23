package session

import "strings"

const (
	titlePromptMaxChars = 2000
	// MaxTitleRunes is the maximum length for persisted/display session titles.
	MaxTitleRunes = 60
)

// OneLine collapses whitespace and replaces newlines for single-line display.
func OneLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			if !prevSpace && b.Len() > 0 {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		if r == ' ' {
			if !prevSpace && b.Len() > 0 {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return strings.TrimSpace(b.String())
}

// TruncateTitle shortens a string for session list display (by rune count).
// When truncated, the result includes "..." and is at most n runes.
func TruncateTitle(s string, n int) string {
	s = OneLine(s)
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-3]) + "..."
}

// TitlePromptSnippet prepares user content for the title subagent prompt.
func TitlePromptSnippet(content string) string {
	return TruncateTitle(content, titlePromptMaxChars)
}
