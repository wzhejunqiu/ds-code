package header

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// wrapCells breaks s into lines of at most maxCells terminal columns (UTF-8 safe).
func wrapCells(s string, maxCells int) []string {
	if maxCells <= 0 {
		return []string{s}
	}
	if lipgloss.Width(s) <= maxCells {
		return []string{s}
	}
	var lines []string
	var b strings.Builder
	width := 0
	flush := func() {
		if b.Len() == 0 {
			return
		}
		lines = append(lines, b.String())
		b.Reset()
		width = 0
	}
	for _, r := range s {
		seg := string(r)
		w := lipgloss.Width(seg)
		if w > maxCells {
			flush()
			lines = append(lines, seg)
			continue
		}
		if width+w > maxCells && b.Len() > 0 {
			flush()
		}
		b.WriteString(seg)
		width += w
	}
	flush()
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}
