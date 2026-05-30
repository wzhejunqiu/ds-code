package model

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/view"
)

const chatSyncInterval = 33 * time.Millisecond

func (m *Model) syncCaches() *view.SyncCaches {
	return &view.SyncCaches{
		Chat:   &m.chatRenderCache,
		MD:     &m.mdSegmentCache,
		Header: &m.headerCache,
	}
}

func (m *Model) resetRenderCaches() {
	m.chatRenderCache.Reset()
	m.mdSegmentCache.Reset()
	m.headerCache.Invalidate()
}

func (m *Model) syncChatView() {
	m.chatSyncScheduled = false
	view.SyncChat(&m.State, &m.chatVP, &m.toolVP, &m.input, m.syncCaches())
}

func (m *Model) scheduleSyncChatView() tea.Cmd {
	if m.chatSyncScheduled {
		return nil
	}
	m.chatSyncScheduled = true
	return tea.Tick(chatSyncInterval, func(time.Time) tea.Msg {
		return chatSyncFlushMsg{}
	})
}

func (m *Model) syncChatViewResetting() {
	m.resetRenderCaches()
	m.syncChatView()
}

func (m *Model) refreshStatus() {
	view.RefreshStatus(&m.State)
	m.headerCache.Invalidate()
}
