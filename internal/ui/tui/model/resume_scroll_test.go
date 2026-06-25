package model

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/session"
)

func TestSessionResumed_scrollsToBottom(t *testing.T) {
	m := New(testDeps(true))
	m.Width = 80
	m.Height = 24
	seedChatLines(m, 10)
	m.chatScrollY = 0

	longChat := []chat.Block{{Role: chat.RoleUser, Content: strings.Repeat("resumed\n", 80)}}
	session.UpdateSessionResumed(&m.State, msg.SessionResumedMsg{
		SessionID: "sess-1",
		Chat:      longChat,
	}, &m.resumePicker, m.syncChatAfterLoad, m.syncToolView, nil)

	if !chatScrollAtBottom(m) {
		t.Fatalf("chatScrollY = %d, want scrolled to bottom", m.chatScrollY)
	}
}

func TestHistoryLoaded_scrollsToBottom(t *testing.T) {
	m := New(testDeps(true))
	m.Width = 80
	m.Height = 24
	m.chatScrollY = 0

	longChat := []chat.Block{{Role: chat.RoleUser, Content: strings.Repeat("history\n", 80)}}
	session.UpdateHistoryLoaded(&m.State, msg.HistoryLoadedMsg{Chat: longChat}, m.syncChatAfterLoad, nil)

	if !chatScrollAtBottom(m) {
		t.Fatalf("chatScrollY = %d, want scrolled to bottom", m.chatScrollY)
	}
}
