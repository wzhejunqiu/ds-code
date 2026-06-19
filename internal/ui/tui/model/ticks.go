package model

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	uipkg "github.com/wzhejunqiu/ds-code/internal/ui"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/header"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
)

const (
	thinkingFineTick     = 100 * time.Millisecond
	thinkingCoarseTick   = time.Second
	noticeScrollInterval = 4 * time.Second
)

func thinkingTickAfter(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg { return msg.ThinkingTickMsg{} })
}

func exitConfirmTimeoutTick() tea.Cmd {
	return tea.Tick(uipkg.ExitConfirmTimeout, func(time.Time) tea.Msg { return msg.ExitConfirmTimeoutMsg{} })
}

func noticeScrollTickAfter(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg { return msg.NoticeScrollTickMsg{} })
}

func (m *Model) scheduleNoticeScroll() tea.Cmd {
	if m.Width <= 0 {
		return nil
	}
	narrow := m.Width < 72
	if !header.NeedsAutoScroll(m.StartupNotices, m.Width, narrow) {
		return nil
	}
	return noticeScrollTickAfter(noticeScrollInterval)
}

func (m *Model) handleNoticeScrollTick() tea.Cmd {
	narrow := m.Width > 0 && m.Width < 72
	if !header.AdvanceScrollOffset(m.StartupNotices, m.Width, narrow, &m.NoticeScrollOffset) {
		return nil
	}
	m.headerCache.Invalidate()
	m.syncChatView()
	return m.withHPSync(m.scheduleNoticeScroll())
}

func (m *Model) nextThinkingTickCmd() tea.Cmd {
	elapsed := turnThinkingElapsed(m)
	if elapsed < chat.ThinkingFineDuration {
		return thinkingTickAfter(thinkingFineTick)
	}
	return thinkingTickAfter(thinkingCoarseTick)
}

func turnThinkingElapsed(m *Model) time.Duration {
	if len(m.Chat) == 0 {
		return 0
	}
	blk := m.Chat[len(m.Chat)-1]
	if blk.ReasoningStartedAt.IsZero() {
		return 0
	}
	end := blk.ReasoningEndedAt
	if end.IsZero() {
		end = time.Now()
	}
	d := end.Sub(blk.ReasoningStartedAt)
	if d < 0 {
		return 0
	}
	return d
}
