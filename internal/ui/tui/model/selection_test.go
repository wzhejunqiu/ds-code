package model

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/chat"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/selection"
)

func testDeps(copyOnSelect bool) *deps.Deps {
	return &deps.Deps{
		Cfg: &config.Config{TUI: config.TUIConfig{CopyOnSelect: copyOnSelect}},
	}
}

func seedChatLines(m *Model, lines int) {
	m.Width = 80
	m.Chat = []chat.Block{{Role: chat.RoleUser, Content: strings.Repeat("line\n", lines)}}
	m.syncChatView()
}

func TestSelection_overlayDisablesChatSelect(t *testing.T) {
	m := New(testDeps(true))
	m.Width = 80
	m.chatVP.SetWidth(80)
	m.chatVP.SetHeight(10)
	m.Overlay = state.OverlayHelp

	_, cmd := m.handleMouse(tea.MouseClickMsg{X: 5, Y: 2, Button: tea.MouseLeft})
	if cmd != nil {
		t.Fatal("overlay open should ignore mouse")
	}
	if m.selDragging {
		t.Fatal("should not start selection while overlay is open")
	}
}

func TestSelection_copyOnSelect(t *testing.T) {
	m := New(testDeps(true))
	m.Width = 80
	m.chatVP.SetWidth(80)
	m.chatVP.SetHeight(10)
	m.plainLines = []string{"hello world"}
	m.selRange = selection.Range{
		Start: selection.Point{Line: 0, Col: 0},
		End:   selection.Point{Line: 0, Col: 5},
	}
	m.selDragging = true

	_, cmd := m.handleMouse(tea.MouseReleaseMsg{X: 5, Y: 0, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatal("expected copy command on mouse release")
	}
}

func TestSelection_copyOnSelectDisabled(t *testing.T) {
	m := New(testDeps(false))
	m.plainLines = []string{"hello"}
	m.selRange = selection.Range{
		Start: selection.Point{Line: 0, Col: 0},
		End:   selection.Point{Line: 0, Col: 3},
	}
	m.selDragging = true

	_, cmd := m.handleMouse(tea.MouseReleaseMsg{X: 3, Y: 0, Button: tea.MouseLeft})
	if cmd != nil {
		t.Fatal("copy on select disabled should not auto-copy on release")
	}
}

func TestSelection_mouseWheelScrollsChat(t *testing.T) {
	m := New(testDeps(true))
	seedChatLines(m, 50)
	m.chatVP.SetWidth(80)
	m.chatVP.SetHeight(10)

	before := m.chatScrollY
	_, cmd := m.handleMouse(tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 5, Y: 2})
	if cmd == nil {
		t.Fatal("expected wheel scroll tick command")
	}
	for i := 0; i < 8 && m.chatScrollY <= before; i++ {
		m.handleWheelScrollTick()
	}
	if m.chatScrollY <= before {
		t.Fatalf("wheel down: chatScrollY = %d, want > %d", m.chatScrollY, before)
	}

	before = m.chatScrollY
	_, cmd = m.handleMouse(tea.MouseWheelMsg{Button: tea.MouseWheelUp, X: 5, Y: 2})
	if cmd == nil && !m.scroll.ScrollActive() {
		t.Fatal("expected wheel scroll tick command")
	}
	for i := 0; i < 8 && m.chatScrollY >= before; i++ {
		m.handleWheelScrollTick()
	}
	if m.chatScrollY >= before {
		t.Fatalf("wheel up: chatScrollY = %d, want < %d", m.chatScrollY, before)
	}
}

func TestSelection_viewportHitTest(t *testing.T) {
	m := New(testDeps(true))
	m.Width = 80
	m.chatVP.SetWidth(78)
	m.chatVP.SetHeight(10)
	m.chatScrollY = 2

	pt, ok := m.mapMousePoint(tea.Mouse{X: 10, Y: 5})
	if !ok {
		t.Fatal("expected hit inside chat viewport")
	}
	if pt.Line != 7 {
		t.Fatalf("line = %d want 7 (chatScrollY 2 + y 5)", pt.Line)
	}
	if pt.Col != 10 {
		t.Fatalf("col = %d want 10", pt.Col)
	}
}

func TestSelection_copiesVisibleMCPArgs(t *testing.T) {
	m := New(testDeps(true))
	m.Chat = []chat.Block{{
		Role:     chat.RoleTool,
		ToolName: "semantic_search_nodes",
		ToolArgs: `{"query":"permission"}`,
	}}
	m.Width = 120
	m.updatePlainLines()
	joined := strings.Join(m.plainLines, "\n")
	if !strings.Contains(joined, "query") || !strings.Contains(joined, "permission") {
		t.Fatalf("plain lines missing visible MCP args: %q", joined)
	}
}

func TestSelection_runningTurnAllowsHistory(t *testing.T) {
	m := New(testDeps(true))
	seedChatLines(m, 50)
	m.chatVP.SetWidth(80)
	m.chatVP.SetHeight(10)
	m.Running = true
	m.chatScrollY = 10

	_, cmd := m.handleMouse(tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 5, Y: 2})
	if cmd == nil {
		t.Fatal("wheel should scroll history while turn is running")
	}

	_, cmd = m.handleMouse(tea.MouseClickMsg{X: 5, Y: 2, Button: tea.MouseLeft})
	if cmd != nil {
		t.Fatal("press should not copy yet")
	}
	if !m.selDragging {
		t.Fatal("should allow selection while running")
	}
}

func TestSelection_wheelScrollAfterCopy(t *testing.T) {
	m := New(testDeps(true))
	seedChatLines(m, 50)
	m.chatVP.SetWidth(80)
	m.chatVP.SetHeight(10)
	m.plainLines = []string{"hello world"}
	m.selDragging = true
	m.selRange = selection.Range{
		Start: selection.Point{Line: 0, Col: 0},
		End:   selection.Point{Line: 0, Col: 5},
	}

	_, cmd := m.handleMouse(tea.MouseReleaseMsg{X: 5, Y: 0, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatal("expected copy command on mouse release")
	}
	if m.selDragging {
		t.Fatal("selDragging should be cleared after release")
	}
	if !m.selRange.Active() {
		t.Fatal("selection highlight should remain after copy")
	}

	before := m.chatScrollY
	_, cmd = m.handleMouse(tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 5, Y: 2})
	if cmd == nil {
		t.Fatal("wheel should scroll after copy-on-select")
	}
	for i := 0; i < 8 && m.chatScrollY <= before; i++ {
		m.handleWheelScrollTick()
	}
	if m.chatScrollY <= before {
		t.Fatalf("wheel down after copy: chatScrollY = %d, want > %d", m.chatScrollY, before)
	}
}

func TestSelection_plainTextFromStyled(t *testing.T) {
	styled := "\x1b[1mhello\x1b[0m world"
	plain := selection.StripANSI(styled)
	if plain != "hello world" {
		t.Fatalf("got %q", plain)
	}
}
