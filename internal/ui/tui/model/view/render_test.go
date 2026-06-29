package view

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/scroll"
	"github.com/wzhejunqiu/ds-code/internal/version"
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
	if !strings.Contains(first, version.Name) {
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
	chatVP := viewport.New(viewport.WithWidth(60), viewport.WithHeight(10))
	toolVP := viewport.New(viewport.WithWidth(60), viewport.WithHeight(4))

	SyncChat(s, &chatVP, &toolVP, nil, nil)
	body := chatVP.View()
	if !strings.Contains(body, version.Name) {
		t.Fatalf("viewport should contain header:\n%s", body)
	}
	if !strings.Contains(body, "hello") {
		t.Fatalf("viewport body missing chat:\n%s", body)
	}
}

func TestSyncChat_sentinelScrollsToBottom(t *testing.T) {
	s := &state.State{
		Width:  80,
		Height: 24,
		Deps: &deps.Deps{
			Version: "test",
		},
		Chat: []chat.Block{
			{Role: chat.RoleUser, Content: strings.Repeat("line\n", 60)},
		},
	}
	var catalog chat.LineCatalog
	scrollY := scroll.ChatBottomSentinel
	caches := &SyncCaches{Catalog: &catalog, ChatScrollY: &scrollY}
	chatVP := viewport.New(viewport.WithWidth(60), viewport.WithHeight(10))
	toolVP := viewport.New(viewport.WithWidth(60), viewport.WithHeight(4))

	SyncChat(s, &chatVP, &toolVP, nil, caches)

	if !scroll.IsPinnedBottom(scrollY) {
		t.Fatalf("scrollY = %d, want pinned bottom sentinel", scrollY)
	}
	total := catalog.TotalLines()
	maxY := total - chatVP.Height()
	if maxY < 0 {
		maxY = 0
	}
	if scroll.EffectiveChatY(scrollY, maxY) != maxY {
		t.Fatalf("effective scrollY = %d, want bottom %d (total %d)", scroll.EffectiveChatY(scrollY, maxY), maxY, total)
	}
}

func TestSyncChat_followsBottomOnContentGrowth(t *testing.T) {
	s := &state.State{
		Width:  80,
		Height: 24,
		Deps: &deps.Deps{
			Version: "test",
		},
		Chat: []chat.Block{
			{Role: chat.RoleUser, Content: strings.Repeat("line\n", 60)},
		},
	}
	var catalog chat.LineCatalog
	scrollY := 0
	caches := &SyncCaches{Catalog: &catalog, ChatScrollY: &scrollY}
	chatVP := viewport.New(viewport.WithWidth(60), viewport.WithHeight(10))
	toolVP := viewport.New(viewport.WithWidth(60), viewport.WithHeight(4))

	SyncChat(s, &chatVP, &toolVP, nil, caches)
	maxY := catalog.TotalLines() - chatVP.Height()
	if maxY < 0 {
		maxY = 0
	}
	scrollY = maxY

	s.Chat = append(s.Chat, chat.Block{Role: chat.RoleAssistant, Content: strings.Repeat("new\n", 40)})
	SyncChat(s, &chatVP, &toolVP, nil, caches)

	newMaxY := catalog.TotalLines() - chatVP.Height()
	if newMaxY < 0 {
		newMaxY = 0
	}
	if !scroll.IsPinnedBottom(scrollY) {
		t.Fatalf("scrollY = %d, want pinned bottom after content growth", scrollY)
	}
	if scroll.EffectiveChatY(scrollY, newMaxY) != newMaxY {
		t.Fatalf("effective scrollY = %d, want %d", scroll.EffectiveChatY(scrollY, newMaxY), newMaxY)
	}
	if !strings.Contains(chatVP.View(), "new") {
		t.Fatalf("viewport should show new content:\n%s", chatVP.View())
	}
}

func TestSyncChat_scrolledUpDoesNotJump(t *testing.T) {
	s := &state.State{
		Width:  80,
		Height: 24,
		Deps: &deps.Deps{
			Version: "test",
		},
		Chat: []chat.Block{
			{Role: chat.RoleUser, Content: strings.Repeat("line\n", 60)},
		},
	}
	var catalog chat.LineCatalog
	scrollY := 0
	caches := &SyncCaches{Catalog: &catalog, ChatScrollY: &scrollY}
	chatVP := viewport.New(viewport.WithWidth(60), viewport.WithHeight(10))
	toolVP := viewport.New(viewport.WithWidth(60), viewport.WithHeight(4))

	SyncChat(s, &chatVP, &toolVP, nil, caches)
	scrollY = 10

	s.Chat = append(s.Chat, chat.Block{Role: chat.RoleAssistant, Content: strings.Repeat("new\n", 40)})
	SyncChat(s, &chatVP, &toolVP, nil, caches)

	if scrollY != 10 {
		t.Fatalf("scrollY = %d, want 10 (no jump when scrolled up)", scrollY)
	}
	if strings.Contains(chatVP.View(), "new") {
		t.Fatalf("viewport should not show new content when scrolled up:\n%s", chatVP.View())
	}
}

func TestLayoutUsesFullContentLineCount(t *testing.T) {
	s := &state.State{
		Width:  80,
		Height: 24,
	}
	chatVP := viewport.New(viewport.WithWidth(60), viewport.WithHeight(10))
	toolVP := viewport.New(viewport.WithWidth(60), viewport.WithHeight(4))

	Layout(s, &chatVP, &toolVP, nil, 100)
	if chatVP.Height() >= 100 {
		t.Fatalf("chat height = %d, expected capped below content lines", chatVP.Height())
	}
}

func TestLayout_reservesOverlaySpace(t *testing.T) {
	s := &state.State{
		Width:  80,
		Height: 24,
	}
	chatVP := viewport.New(viewport.WithWidth(60), viewport.WithHeight(10))
	toolVP := viewport.New(viewport.WithWidth(60), viewport.WithHeight(4))

	Layout(s, &chatVP, &toolVP, nil, 200)
	withoutOverlay := chatVP.Height()

	s.Overlay = state.OverlayComplete
	s.OverlayText = "/help — show help\n/resume — resume session\n/clear — new session"
	Layout(s, &chatVP, &toolVP, nil, 200)
	withOverlay := chatVP.Height()

	wantDelta := overlayChromeLines(s)
	if withoutOverlay-withOverlay != wantDelta {
		t.Fatalf("chat height delta = %d, want %d (without=%d with=%d)",
			withoutOverlay-withOverlay, wantDelta, withoutOverlay, withOverlay)
	}
}

func TestSyncChat_visibleWindowOnly(t *testing.T) {
	s := &state.State{
		Width:  80,
		Height: 24,
		Deps: &deps.Deps{
			Version: "test",
		},
		Chat: []chat.Block{
			{Role: chat.RoleUser, Content: strings.Repeat("line\n", 200)},
		},
	}
	var catalog chat.LineCatalog
	var cache chat.RenderCache
	scrollY := 0
	caches := &SyncCaches{Catalog: &catalog, Chat: &cache, ChatScrollY: &scrollY}
	chatVP := viewport.New(viewport.WithWidth(60), viewport.WithHeight(10))
	toolVP := viewport.New(viewport.WithWidth(60), viewport.WithHeight(4))

	SyncChat(s, &chatVP, &toolVP, nil, caches)

	visibleLines := ContentLineCount(chatVP.View())
	if visibleLines > chatVP.Height()+2 {
		t.Fatalf("viewport content lines = %d, want about viewport height %d", visibleLines, chatVP.Height())
	}
	if catalog.TotalLines() <= chatVP.Height() {
		t.Fatalf("catalog total %d should exceed viewport height %d", catalog.TotalLines(), chatVP.Height())
	}
}
