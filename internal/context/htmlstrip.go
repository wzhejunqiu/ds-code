package context

import (
	"regexp"
	"strings"
)

var htmlTagRE = regexp.MustCompile(`(?i)<[^>]*>`)

// StripHTMLTags removes HTML tags, leaving plain text for compact summarization.
func StripHTMLTags(s string) string {
	if s == "" || !strings.Contains(s, "<") {
		return s
	}
	out := htmlTagRE.ReplaceAllString(s, "")
	return strings.TrimSpace(out)
}
