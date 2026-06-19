package model

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	tuimsg "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
)

func TestHistoryLoaded_returnsViewportSync(t *testing.T) {
	m := New(testDeps(true))
	m.Width = 80
	m.Height = 24
	m.chatVP.Height = 10
	m.chatVP.SetContent("placeholder\n")

	_, cmd := m.Update(tuimsg.HistoryLoadedMsg{
		Chat: []chat.Block{{Role: chat.RoleUser, Content: "hello from history"}},
	})
	if cmd == nil {
		t.Fatal("HistoryLoaded should return HP viewport sync command")
	}
	if !m.chatVP.HighPerformanceRendering {
		t.Fatal("HP rendering should be enabled after history load")
	}
}

func TestSessionResumed_returnsViewportSync(t *testing.T) {
	m := New(testDeps(true))
	m.Width = 80
	m.Height = 24
	m.chatVP.Height = 10

	_, cmd := m.Update(tuimsg.SessionResumedMsg{
		SessionID: "sess-resumed",
		Chat:      []chat.Block{{Role: chat.RoleAssistant, Content: "resumed transcript"}},
	})
	if cmd == nil {
		t.Fatal("SessionResumed should return HP viewport sync command")
	}
}
