package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const agentEventRetryInterval = 5 * time.Millisecond
const agentEventMaxRetries = 200

// sendAgentEvent enqueues a message for the TUI event pump.
// Non-critical messages (stream deltas) may be dropped when the channel is full.
// Critical messages (turn lifecycle) retry briefly, then block as a last resort.
func sendAgentEvent(events chan<- tea.Msg, msg tea.Msg, critical bool) {
	if !critical {
		select {
		case events <- msg:
		default:
		}
		return
	}
	for i := 0; i < agentEventMaxRetries; i++ {
		select {
		case events <- msg:
			return
		default:
			time.Sleep(agentEventRetryInterval)
		}
	}
	events <- msg
}
