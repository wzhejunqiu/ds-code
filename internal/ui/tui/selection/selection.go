package selection

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	ansiRE         = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07`)
	highlightStyle = lipgloss.NewStyle().Reverse(true)
)

// StripANSI removes terminal escape sequences from rendered text.
func StripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

// Point is a line/column position in plain viewport text.
type Point struct {
	Line int
	Col  int
}

// Range is an inclusive selection span in plain text coordinates.
type Range struct {
	Start Point
	End   Point
}

func (r Range) Normalized() Range {
	a, b := r.Start, r.End
	if a.Line > b.Line || (a.Line == b.Line && a.Col > b.Col) {
		a, b = b, a
	}
	return Range{Start: a, End: b}
}

func (r Range) Active() bool {
	return r.Start != r.End
}

// Extract returns selected plain text from logical lines.
func Extract(lines []string, r Range) string {
	r = r.Normalized()
	if len(lines) == 0 || !r.Active() {
		return ""
	}
	if r.Start.Line == r.End.Line {
		line := safeLine(lines, r.Start.Line)
		return sliceCols(line, r.Start.Col, r.End.Col)
	}
	var b strings.Builder
	b.WriteString(sliceCols(safeLine(lines, r.Start.Line), r.Start.Col, len(safeLine(lines, r.Start.Line))))
	for i := r.Start.Line + 1; i < r.End.Line; i++ {
		b.WriteByte('\n')
		b.WriteString(lines[i])
	}
	b.WriteByte('\n')
	b.WriteString(sliceCols(safeLine(lines, r.End.Line), 0, r.End.Col))
	return b.String()
}

func safeLine(lines []string, idx int) string {
	if idx < 0 || idx >= len(lines) {
		return ""
	}
	return lines[idx]
}

func sliceCols(line string, start, end int) string {
	runes := []rune(line)
	if start < 0 {
		start = 0
	}
	if end > len(runes) {
		end = len(runes)
	}
	if start >= end {
		return ""
	}
	return string(runes[start:end])
}

// LinesFromContent splits stripped viewport content into logical lines.
func LinesFromContent(content string) []string {
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

// HighlightLines returns plain text with reverse-video styling on the selected span.
func HighlightLines(lines []string, r Range) string {
	r = r.Normalized()
	if len(lines) == 0 || !r.Active() {
		return strings.Join(lines, "\n")
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = highlightLine(line, i, r)
	}
	return strings.Join(out, "\n")
}

func highlightLine(line string, lineIdx int, r Range) string {
	if lineIdx < r.Start.Line || lineIdx > r.End.Line {
		return line
	}
	runes := []rune(line)
	if len(runes) == 0 {
		if lineIdx >= r.Start.Line && lineIdx <= r.End.Line {
			return highlightStyle.Render("")
		}
		return line
	}
	lo, hi := 0, len(runes)
	if lineIdx == r.Start.Line {
		lo = clampCol(r.Start.Col, len(runes))
	}
	if lineIdx == r.End.Line {
		hi = clampCol(r.End.Col, len(runes))
	}
	if lo >= hi {
		return line
	}
	var b strings.Builder
	if lo > 0 {
		b.WriteString(string(runes[:lo]))
	}
	b.WriteString(highlightStyle.Render(string(runes[lo:hi])))
	if hi < len(runes) {
		b.WriteString(string(runes[hi:]))
	}
	return b.String()
}

func clampCol(col, max int) int {
	if col < 0 {
		return 0
	}
	if col > max {
		return max
	}
	return col
}
