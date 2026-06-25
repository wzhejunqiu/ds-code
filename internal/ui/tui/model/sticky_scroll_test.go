package model

import (
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/scroll"
)

func chatScrollAtBottom(m *Model) bool {
	maxY := m.chatMaxY()
	return scroll.IsPinnedBottom(m.chatScrollY) || scroll.EffectiveChatY(m.chatScrollY, maxY) == maxY
}

func TestStickyScroll_followsBottomOnContentGrowth(t *testing.T) {
	m := New(testDeps(true))
	m.Width = 80
	m.Height = 24
	seedChatLines(m, 80)

	maxY := m.chatMaxY()
	m.chatScrollY = maxY

	m.Chat = append(m.Chat, chat.Block{Role: chat.RoleAssistant, Content: strings.Repeat("stream\n", 30)})
	m.syncChatView()

	if !chatScrollAtBottom(m) {
		t.Fatalf("chatScrollY = %d, want pinned at bottom after growth", m.chatScrollY)
	}
	if !strings.Contains(m.chatVP.View(), "stream") {
		t.Fatalf("viewport should show new content:\n%s", m.chatVP.View())
	}
}

func TestStickyScroll_scrolledUpDoesNotJump(t *testing.T) {
	m := New(testDeps(true))
	m.Width = 80
	m.Height = 24
	seedChatLines(m, 80)

	m.chatScrollY = 10
	before := m.chatScrollY

	m.Chat = append(m.Chat, chat.Block{Role: chat.RoleAssistant, Content: strings.Repeat("stream\n", 30)})
	m.syncChatView()

	if m.chatScrollY != before {
		t.Fatalf("chatScrollY = %d, want %d (no jump when scrolled up)", m.chatScrollY, before)
	}
}

func TestStickyScroll_scrollBackToBottomResumesFollow(t *testing.T) {
	m := New(testDeps(true))
	m.Width = 80
	m.Height = 24
	seedChatLines(m, 80)

	m.chatScrollY = 10
	m.jumpViewport(&m.chatVP, 9999)

	if !scroll.IsPinnedBottom(m.chatScrollY) {
		t.Fatalf("chatScrollY = %d, want sentinel after jumping to bottom", m.chatScrollY)
	}

	m.Chat = append(m.Chat, chat.Block{Role: chat.RoleAssistant, Content: strings.Repeat("tail\n", 20)})
	m.syncChatView()

	if !chatScrollAtBottom(m) {
		t.Fatalf("chatScrollY = %d, want pinned at bottom after re-pin", m.chatScrollY)
	}
	if !strings.Contains(m.chatVP.View(), "tail") {
		t.Fatalf("viewport should show new tail content:\n%s", m.chatVP.View())
	}
}
