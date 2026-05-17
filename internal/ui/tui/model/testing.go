package model

import (
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/session"
)

// TestSyncChatView syncs chat/tool viewports (for tests in package tui).
func (m *Model) TestSyncChatView() {
	m.syncChatView()
}

// TestSyncResumePicker rebuilds the resume overlay picker (for tests in package tui).
func (m *Model) TestSyncResumePicker() {
	session.SyncResumePicker(&m.State, &m.resumePicker)
}

// TestInputSetValue sets the prompt input value (for tests in package tui).
func (m *Model) TestInputSetValue(s string) {
	m.input.SetValue(s)
}
