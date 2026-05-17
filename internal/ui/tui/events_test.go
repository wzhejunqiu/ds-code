package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSendAgentEvent_criticalEventuallyDelivers(t *testing.T) {
	events := make(chan tea.Msg, 1)
	events <- struct{}{} // fill buffer

	go func() {
		time.Sleep(20 * time.Millisecond)
		<-events // make room for critical message
	}()

	done := make(chan struct{})
	go func() {
		sendAgentEvent(events, turnDoneMsg{}, true)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("critical send blocked too long")
	}

	msg := <-events
	if _, ok := msg.(turnDoneMsg); !ok {
		t.Fatalf("msg type %T", msg)
	}
}

func TestSendAgentEvent_nonCriticalDropsWhenFull(t *testing.T) {
	events := make(chan tea.Msg, 1)
	events <- struct{}{}

	sendAgentEvent(events, streamContentMsg{delta: "x"}, false)

	select {
	case msg := <-events:
		if _, ok := msg.(streamContentMsg); ok {
			t.Fatal("expected non-critical message to be dropped when channel is full")
		}
	default:
	}
}
