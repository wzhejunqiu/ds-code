package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hejunqiu/ds-code/internal/session"
)

const resumeListMax = 50

func (m *model) resumePageSize() int {
	if m.height <= 0 {
		return 8
	}
	// Space below the input frame for the session list overlay.
	n := m.height/5 - 1
	if n < 4 {
		n = 4
	}
	if n > 14 {
		n = 14
	}
	return n
}

func (m *model) ensureResumeScrollVisible() {
	total := len(m.resumeSessions)
	if total == 0 {
		m.resumeScroll = 0
		return
	}
	page := m.resumePageSize()
	if m.resumeIdx < 0 {
		m.resumeIdx = 0
	}
	if m.resumeIdx >= total {
		m.resumeIdx = total - 1
	}
	if m.resumeIdx < m.resumeScroll {
		m.resumeScroll = m.resumeIdx
	}
	if m.resumeIdx >= m.resumeScroll+page {
		m.resumeScroll = m.resumeIdx - page + 1
	}
	maxScroll := total - page
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.resumeScroll > maxScroll {
		m.resumeScroll = maxScroll
	}
	if m.resumeScroll < 0 {
		m.resumeScroll = 0
	}
}

func (m *model) resumeMoveSelection(delta int) {
	if len(m.resumeSessions) == 0 {
		return
	}
	m.resumeIdx += delta
	m.ensureResumeScrollVisible()
	m.renderResumeOverlay()
}

func (m *model) resumePageSelection(pages int) {
	if len(m.resumeSessions) == 0 {
		return
	}
	page := m.resumePageSize()
	m.resumeIdx += pages * page
	m.ensureResumeScrollVisible()
	m.renderResumeOverlay()
}

func (m *model) listResumeSessions(filter string) ([]session.Summary, error) {
	list, err := m.deps.Store.ListSessions(context.Background(), resumeListMax)
	if err != nil {
		return nil, err
	}
	if filter == "" {
		return list, nil
	}
	var filtered []session.Summary
	lower := strings.ToLower(filter)
	for _, s := range list {
		if strings.HasPrefix(s.ID, filter) ||
			strings.Contains(strings.ToLower(s.ID), lower) ||
			strings.Contains(strings.ToLower(s.Title), lower) {
			filtered = append(filtered, s)
		}
	}
	return filtered, nil
}

func (m *model) renderResumeOverlay() {
	total := len(m.resumeSessions)
	if total == 0 {
		m.overlayText = "No matching sessions."
		return
	}
	m.ensureResumeScrollVisible()

	page := m.resumePageSize()
	end := m.resumeScroll + page
	if end > total {
		end = total
	}

	var b strings.Builder
	b.WriteString("Recent sessions (↑↓ scroll, PgUp/PgDn, Enter to resume):\n")
	for i := m.resumeScroll; i < end; i++ {
		s := m.resumeSessions[i]
		prefix := "  "
		if i == m.resumeIdx {
			prefix = "▸ "
		}
		title := s.Title
		if title == "" {
			title = "(untitled)"
		}
		fmt.Fprintf(&b, "%s%s  %s  %s\n", prefix, s.ID, title, formatResumeUpdated(s.UpdatedAt))
	}
	if total > page {
		fmt.Fprintf(&b, "  — %d–%d of %d —", m.resumeScroll+1, end, total)
	}
	m.overlayText = strings.TrimRight(b.String(), "\n")
}

func formatResumeUpdated(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04")
}
