package session

import "strings"

// TruncateTitle shortens a string for session list display.
func TruncateTitle(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
