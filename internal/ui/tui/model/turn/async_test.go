package turn

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
)

func TestSendAgentEvent_criticalEventuallyDelivers(t *testing.T) {
	events := make(chan tea.Msg, 1)
	events <- struct{}{}

	go func() {
		time.Sleep(20 * time.Millisecond)
		<-events
	}()

	done := make(chan struct{})
	go func() {
		sendAgentEvent(events, msg.TurnDoneMsg{}, true)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("critical send blocked too long")
	}

	got := <-events
	if _, ok := got.(msg.TurnDoneMsg); !ok {
		t.Fatalf("msg type %T", got)
	}
}

func TestSendAgentEvent_nonCriticalDropsWhenFull(t *testing.T) {
	events := make(chan tea.Msg, 1)
	events <- struct{}{}

	sendAgentEvent(events, msg.StreamContentMsg{Delta: "x"}, false)

	select {
	case m := <-events:
		if _, ok := m.(msg.StreamContentMsg); ok {
			t.Fatal("expected non-critical message to be dropped when channel is full")
		}
	default:
	}
}
