package llm

import "strings"

// IsContextTooLong reports whether err indicates context length exceeded.
func IsContextTooLong(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	phrases := []string{
		"context length exceeded",
		"maximum context length",
		"context window",
		"context is too long",
		"exceeds the context",
		"too many tokens",
	}
	for _, p := range phrases {
		if strings.Contains(s, p) {
			return true
		}
	}
	if strings.Contains(s, "context") && strings.Contains(s, "token") &&
		(strings.Contains(s, "limit") || strings.Contains(s, "exceed")) {
		return true
	}
	return false
}
