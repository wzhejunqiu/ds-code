package model

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	uipkg "github.com/hejunqiu/ds-code/internal/ui"
	"github.com/hejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/msg"
)

const (
	thinkingFineTick   = 100 * time.Millisecond
	thinkingCoarseTick = time.Second
)

func statusTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return msg.StatusRefreshMsg{} })
}

func thinkingTickAfter(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg { return msg.ThinkingTickMsg{} })
}

func exitConfirmTimeoutTick() tea.Cmd {
	return tea.Tick(uipkg.ExitConfirmTimeout, func(time.Time) tea.Msg { return msg.ExitConfirmTimeoutMsg{} })
}

func (m *Model) nextThinkingTickCmd() tea.Cmd {
	elapsed := turnThinkingElapsed(m)
	if elapsed < chat.ThinkingFineDuration {
		return thinkingTickAfter(thinkingFineTick)
	}
	return thinkingTickAfter(thinkingCoarseTick)
}

func turnThinkingElapsed(m *Model) time.Duration {
	if len(m.State.Chat) == 0 {
		return 0
	}
	blk := m.State.Chat[len(m.State.Chat)-1]
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
