package subagent_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/component"
	subagentui "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/subagent"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

func TestHandleNavKey_downOpensList(t *testing.T) {
	var s state.State
	s.Subagents.Start("sa-1", "probe", "do work", "Explore", false)
	var picker component.Picker
	_, handled := subagentui.HandleNavKey(&s, tea.KeyMsg{Type: tea.KeyDown}, &picker, func() {})
	if !handled || s.SubagentNav != state.SubagentNavList {
		t.Fatalf("handled=%v nav=%v", handled, s.SubagentNav)
	}
}

func TestHandleNavKey_leftFromDetailReturnsToList(t *testing.T) {
	var s state.State
	s.Subagents.Start("sa-1", "probe", "do work", "Explore", false)
	var picker component.Picker
	subagentui.OpenDetail(&s, "sa-1", &picker, func() {})
	if s.SubagentNav != state.SubagentNavDetail {
		t.Fatal("expected detail nav")
	}
	_, handled := subagentui.HandleNavKey(&s, tea.KeyMsg{Type: tea.KeyLeft}, &picker, func() {})
	if !handled || s.SubagentNav != state.SubagentNavList {
		t.Fatalf("handled=%v nav=%v", handled, s.SubagentNav)
	}
}
