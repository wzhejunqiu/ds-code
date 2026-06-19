package view

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

func TestFooterLeft_backgroundAgentsBadge(t *testing.T) {
	n := 0
	s := &state.State{
		Deps: &deps.Deps{
			BackgroundAgents: func() int {
				return n
			},
		},
	}
	if got := FooterLeft(s); strings.Contains(got, "agents running") {
		t.Fatalf("expected no badge when zero, got %q", got)
	}
	n = 2
	got := FooterLeft(s)
	if !strings.Contains(got, "2 agents running in background") {
		t.Fatalf("expected badge, got %q", got)
	}
}

func TestBuildHeaderCached(t *testing.T) {
	s := &state.State{
		Width:  80,
		Height: 24,
		Deps: &deps.Deps{
			Version: "test",
		},
	}
	var cache HeaderCache
	first := buildHeaderCached(s, 60, &cache)
	second := buildHeaderCached(s, 60, &cache)
	if first != second {
		t.Fatal("expected header cache hit")
	}
	if !strings.Contains(first, "ds-code") {
		t.Fatalf("missing title in header: %q", first)
	}
}

func TestSyncChatIncludesHeader(t *testing.T) {
	s := &state.State{
		Width:  80,
		Height: 24,
		Deps: &deps.Deps{
			Version: "test",
		},
		Chat: []chat.Block{
			{Role: chat.RoleUser, Content: "hello"},
		},
	}
	chatVP := viewport.New(60, 10)
	toolVP := viewport.New(60, 4)

	SyncChat(s, &chatVP, &toolVP, nil, nil)
	body := chatVP.View()
	if !strings.Contains(body, "ds-code") {
		t.Fatalf("viewport should contain header:\n%s", body)
	}
	if !strings.Contains(body, "hello") {
		t.Fatalf("viewport body missing chat:\n%s", body)
	}
}

func TestLayoutUsesFullContentLineCount(t *testing.T) {
	s := &state.State{
		Width:  80,
		Height: 24,
	}
	chatVP := viewport.New(60, 10)
	toolVP := viewport.New(60, 4)

	Layout(s, &chatVP, &toolVP, nil, 100)
	if chatVP.Height >= 100 {
		t.Fatalf("chat height = %d, expected capped below content lines", chatVP.Height)
	}
}
