package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

func (m *model) handleResumeKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "up":
		m.resumeMoveSelection(-1)
		return true
	case "down", "tab":
		m.resumeMoveSelection(1)
		return true
	case "pgup":
		m.resumePageSelection(-1)
		return true
	case "pgdown":
		m.resumePageSelection(1)
		return true
	case "enter":
		// Handled before handleResumeKey in the KeyMsg block.
		return true
	case "esc":
		m.clearResumePicker()
		return true
	}
	return false
}

func (m *model) clearResumePicker() {
	m.overlay = overlayNone
	m.resumeSessions = nil
	m.resumeFilter = ""
	m.resumeIdx = 0
	m.resumeScroll = 0
	m.overlayText = ""
}

func (m *model) updateResumePicker(filter string) {
	// textinput emits updates on cursor blink; do not reset selection unless filter changed.
	if m.overlay == overlayResume && filter == m.resumeFilter && len(m.resumeSessions) > 0 {
		return
	}
	filterChanged := m.resumeFilter != filter
	m.resumeFilter = filter

	list, err := m.listResumeSessions(filter)
	if err != nil {
		m.errLine = err.Error()
		m.clearResumePicker()
		return
	}
	m.resumeSessions = list
	if filterChanged || len(list) == 0 {
		m.resumeIdx = 0
		m.resumeScroll = 0
	} else if m.resumeIdx >= len(list) {
		m.resumeIdx = len(list) - 1
	}
	if len(list) == 0 {
		m.overlayText = "No matching sessions."
		m.overlay = overlayResume
		return
	}
	m.overlay = overlayResume
	m.renderResumeOverlay()
}

func (m *model) fetchResumeList() tea.Cmd {
	d := m.deps
	return func() tea.Msg {
		list, err := d.Store.ListSessions(context.Background(), resumeListMax)
		return resumeListMsg{sessions: list, err: err}
	}
}

func (m *model) loadInitialHistory() tea.Cmd {
	if m.deps == nil || m.deps.Store == nil || m.sessionID == "" {
		return nil
	}
	d := m.deps
	sid := m.sessionID
	reasoningOpen := m.reasoningAll
	return func() tea.Msg {
		chat, err := loadSessionChat(d.Store, sid, reasoningOpen)
		return historyLoadedMsg{chat: chat, err: err}
	}
}

func (m *model) resumeSession(id string) tea.Cmd {
	d := m.deps
	reasoningOpen := m.reasoningAll
	return func() tea.Msg {
		ctx := context.Background()
		if _, err := d.Store.Get(ctx, id); err != nil {
			return sessionResumedMsg{err: err}
		}
		chat, err := loadSessionChat(d.Store, id, reasoningOpen)
		if err != nil {
			return sessionResumedMsg{err: err}
		}
		return sessionResumedMsg{sessionID: id, chat: chat}
	}
}
