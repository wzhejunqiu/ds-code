package turn

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/msg"
)

func TestSendAgentEvent_criticalDropsWhenChannelStaysFull(t *testing.T) {
	events := make(chan tea.Msg, 1)
	events <- struct{}{} // keep channel full

	done := make(chan struct{})
	go func() {
		sendAgentEvent(events, msg.TurnDoneMsg{}, true)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("critical send should not block when channel stays full")
	}

	select {
	case m := <-events:
		if _, ok := m.(msg.TurnDoneMsg); ok {
			t.Fatal("TurnDoneMsg should be dropped after retries, not delivered")
		}
	default:
	}
}
