package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/ui/tui/component"
)

const resumeListMax = 50

var resumePickerKeys = component.PickerKeyOpts{Tab: component.PickerTabMoveDown}

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

func (m *model) syncResumePicker() {
	items := make([]string, len(m.resumeSessions))
	for i, s := range m.resumeSessions {
		title := s.Title
		if title == "" {
			title = "(untitled)"
		}
		items[i] = fmt.Sprintf("%s  %s  %s", s.ID, title, formatResumeUpdated(s.UpdatedAt))
	}
	m.resumePicker.Header = "Recent sessions (↑↓ scroll, PgUp/PgDn, Enter to resume):"
	m.resumePicker.Empty = "No matching sessions."
	m.resumePicker.PageSize = m.resumePageSize()
	m.resumePicker.SetItems(items)
	m.overlayText = m.resumePicker.View()
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

func formatResumeUpdated(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04")
}

func (m *model) handleResumeKey(msg tea.KeyMsg) bool {
	action, handled := m.resumePicker.HandleKey(msg, resumePickerKeys)
	if !handled {
		return false
	}
	if action == component.PickerKeyCancel {
		m.dismissOverlay()
		return true
	}
	m.syncResumePicker()
	return true
}

func (m *model) clearResumePicker() {
	m.overlay = overlayNone
	m.resumeSessions = nil
	m.resumeFilter = ""
	m.resumePicker.Clear()
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
		m.resumePicker.ResetSelection()
	} else {
		m.resumePicker.ClampSelection()
	}
	if len(list) == 0 {
		m.overlay = overlayResume
		m.syncResumePicker()
		return
	}
	m.overlay = overlayResume
	m.syncResumePicker()
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
