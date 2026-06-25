package model

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/view"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/scroll"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/selection"
)

const chatSyncInterval = 33 * time.Millisecond

func (m *Model) scrollChatToBottom() {
	m.chatScrollY = scroll.ChatBottomSentinel
	m.selDragging = false
	m.selRange = selection.Range{}
}

func (m *Model) chatMaxY() int {
	maxY := m.lineCatalog.TotalLines() - m.chatVP.Height()
	if maxY < 0 {
		return 0
	}
	return maxY
}

func (m *Model) effectiveChatScrollY() int {
	if m.lineCatalog.TotalLines() == 0 {
		if scroll.IsPinnedBottom(m.chatScrollY) {
			return 0
		}
		return m.chatScrollY
	}
	return scroll.EffectiveChatY(m.chatScrollY, m.chatMaxY())
}

func (m *Model) setChatScrollY(y int) {
	maxY := m.chatMaxY()
	y = scroll.EffectiveChatY(y, maxY)
	if y >= maxY {
		m.chatScrollY = scroll.ChatBottomSentinel
	} else {
		m.chatScrollY = y
	}
}

func (m *Model) syncCaches() *view.SyncCaches {
	return &view.SyncCaches{
		Chat:        &m.chatRenderCache,
		MD:          &m.mdSegmentCache,
		Header:      &m.headerCache,
		Catalog:     &m.lineCatalog,
		ChatScrollY: &m.chatScrollY,
	}
}

func (m *Model) resetRenderCaches() {
	m.chatRenderCache.Reset()
	m.mdSegmentCache.Reset()
	m.headerCache.Invalidate()
	m.lineCatalog.Reset()
}

func (m *Model) syncChatView() {
	m.chatSyncScheduled = false
	view.SyncChat(&m.State, &m.chatVP, &m.toolVP, &m.input, m.syncCaches())
	m.updatePlainLines()
}

func (m *Model) refreshLayout() {
	if m.Width == 0 {
		return
	}
	totalLines := m.lineCatalog.TotalLines()
	view.Layout(&m.State, &m.chatVP, &m.toolVP, &m.input, totalLines)
}

func (m *Model) scheduleSyncChatView() tea.Cmd {
	if m.scroll.ScrollActive() || m.scroll.HasPending() {
		m.scrollDeferSync = true
		return nil
	}
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

func (m *Model) syncChatAfterLoad() {
	m.scrollChatToBottom()
	m.syncChatViewResetting()
}

func (m *Model) refreshStatus() {
	view.RefreshStatus(&m.State)
	m.headerCache.Invalidate()
}
