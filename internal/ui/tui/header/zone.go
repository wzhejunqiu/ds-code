package header

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/style"
)

const (
	warnPrefix            = "⚠ "
	maxNoticeVisibleLines = 8
	noticeBlockGap        = 1 // blank line between notice blocks
)

type noticeLine struct {
	text  string
	level Level
}

// ZoneWidth returns the notification column width for the given terminal width.
func ZoneWidth(termWidth int, narrow bool) int {
	if termWidth < 20 {
		termWidth = 20
	}
	if narrow {
		w := termWidth - 4
		if w < 10 {
			return 10
		}
		return w
	}
	w := termWidth / 2
	if w < 20 {
		w = termWidth - 4
	}
	if w < 10 {
		w = 10
	}
	return w
}

// BuildNoticeLines expands notices into wrapped plain-text lines (no ANSI).
func BuildNoticeLines(notices []Notice, zoneWidth int) []noticeLine {
	if zoneWidth < 10 {
		zoneWidth = 10
	}
	var out []noticeLine
	for i, n := range notices {
		if strings.TrimSpace(n.Text) == "" {
			continue
		}
		if i > 0 && len(out) > 0 {
			for g := 0; g < noticeBlockGap; g++ {
				out = append(out, noticeLine{level: n.Level})
			}
		}
		paragraphs := strings.Split(n.Text, "\n")
		firstInBlock := true
		for _, para := range paragraphs {
			para = strings.TrimSpace(para)
			if para == "" {
				continue
			}
			prefix := ""
			lineWidth := zoneWidth
			if firstInBlock && n.Level == NoticeWarn {
				prefix = warnPrefix
				lineWidth = zoneWidth - lipgloss.Width(warnPrefix)
				if lineWidth < 8 {
					lineWidth = 8
				}
			}
			wrapped := wrapCells(para, lineWidth)
			prefixWidth := lipgloss.Width(prefix)
			for j, w := range wrapped {
				line := w
				if j == 0 {
					line = prefix + w
				} else if prefixWidth > 0 {
					line = strings.Repeat(" ", prefixWidth) + w
				}
				out = append(out, noticeLine{text: line, level: n.Level})
			}
			firstInBlock = false
		}
	}
	return out
}

// visibleContentLines returns how many notice content lines fit in the zone.
// When scrolling is needed, one line is reserved for the scroll counter hint.
func visibleContentLines(total int) int {
	if total <= maxNoticeVisibleLines {
		return total
	}
	return maxNoticeVisibleLines - 1
}

func needsScroll(total int) bool {
	return total > maxNoticeVisibleLines
}

// MaxScrollOffset returns the maximum scroll offset for wrapped notice lines.
func MaxScrollOffset(notices []Notice, zoneWidth int) int {
	total := len(BuildNoticeLines(notices, zoneWidth))
	visible := visibleContentLines(total)
	if !needsScroll(total) {
		return 0
	}
	return total - visible
}

// ScrollHint returns a footer hint when scrolling is available, or "".
func ScrollHint(offset, total int) string {
	if !needsScroll(total) {
		return ""
	}
	visible := visibleContentLines(total)
	end := offset + visible
	if end > total {
		end = total
	}
	return fmt.Sprintf("通知 %d–%d / %d", offset+1, end, total)
}

func renderNotificationZone(notices []Notice, zoneWidth int, scrollOffset int) string {
	lines := BuildNoticeLines(notices, zoneWidth)
	if len(lines) == 0 {
		return ""
	}
	total := len(lines)
	maxOff := MaxScrollOffset(notices, zoneWidth)
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	if scrollOffset > maxOff {
		scrollOffset = maxOff
	}
	visible := visibleContentLines(total)
	window := lines
	if needsScroll(total) {
		window = lines[scrollOffset : scrollOffset+visible]
	}
	var rendered []string
	for _, ln := range window {
		if ln.text == "" {
			rendered = append(rendered, "")
			continue
		}
		st := style.HeaderMeta
		if ln.level == NoticeWarn {
			st = st.Foreground(lipgloss.Color("#DC2626"))
		}
		rendered = append(rendered, st.Render(ln.text))
	}
	if hint := ScrollHint(scrollOffset, total); hint != "" {
		rendered = append(rendered, style.HeaderMeta.Render(hint))
	}
	return strings.Join(rendered, "\n")
}
