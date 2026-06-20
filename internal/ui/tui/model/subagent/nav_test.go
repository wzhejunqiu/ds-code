package subagent_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/component"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
	subagentui "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/subagent"
)

func TestHandleNavKey_downOpensList(t *testing.T) {
	var s state.State
	s.Subagents.Start("sa-1", "probe", "do work", "Explore", false)
	var picker component.Picker
	_, handled := subagentui.HandleNavKey(&s, tea.KeyPressMsg{Code: tea.KeyDown}, &picker, func() {})
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
	_, handled := subagentui.HandleNavKey(&s, tea.KeyPressMsg{Code: tea.KeyLeft}, &picker, func() {})
	if !handled || s.SubagentNav != state.SubagentNavList {
		t.Fatalf("handled=%v nav=%v", handled, s.SubagentNav)
	}
}
