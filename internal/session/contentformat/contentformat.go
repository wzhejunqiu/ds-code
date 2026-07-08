// Package contentformat defines assistant message output format constants.
package contentformat

const (
	Markdown = "markdown"
	HTML     = "html"
)

// Valid reports whether s is a known format.
func Valid(s string) bool {
	return s == Markdown || s == HTML || s == ""
}

// Normalize returns markdown for empty or unknown values.
func Normalize(s string) string {
	if s == HTML {
		return HTML
	}
	return Markdown
}
