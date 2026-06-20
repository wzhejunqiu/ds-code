package model

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/ui/slash"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/input"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

func TestCompletionReadyToSubmit(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"/context", true},
		{"/context ", true},
		{"/context --json", true},
		{"/c", false},
		{"/cont", false},
		{"/help", true},
		{"hello", false},
	}
	for _, tt := range tests {
		if got := input.CompletionReadyToSubmit(tt.input); got != tt.want {
			t.Errorf("CompletionReadyToSubmit(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestSyncCompleteOverlayHighlightsSelection(t *testing.T) {
	m := New(&deps.Deps{})
	m.Complete = []slash.Command{
		{Name: "clear", Description: "new session"},
		{Name: "context", Description: "context panel"},
	}
	m.completePicker.Cursor = 1
	input.SyncCompleteOverlay(&m.State, &m.completePicker)

	lines := strings.Split(m.OverlayText, "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2:\n%s", len(lines), m.OverlayText)
	}
	if lines[0] == lines[1] {
		t.Fatal("expected selected and unselected lines to differ visually")
	}
	if !strings.Contains(lines[1], "context") {
		t.Fatalf("selected line missing command: %q", lines[1])
	}
}

func TestRunningEscDismissesCompleteOverlay(t *testing.T) {
	m := New(&deps.Deps{})
	m.Running = true
	m.Overlay = state.OverlayComplete
	m.Complete = []slash.Command{{Name: "clear", Description: "new session"}}
	input.SyncCompleteOverlay(&m.State, &m.completePicker)

	_, handled := m.updateKey(tea.KeyPressMsg{Code: tea.KeyEsc})
	if !handled {
		t.Fatal("expected esc to be handled")
	}
	if m.Overlay != state.OverlayNone {
		t.Fatalf("overlay = %v, want overlayNone", m.Overlay)
	}
	if m.Complete != nil {
		t.Fatal("expected complete list to be cleared")
	}
	if m.OverlayText != "" {
		t.Fatalf("overlayText = %q, want empty", m.OverlayText)
	}
}

func TestHandleCompleteKeyTabSelectsFirst(t *testing.T) {
	m := New(&deps.Deps{})
	m.Complete = []slash.Command{
		{Name: "clear", Description: "new session"},
		{Name: "compact", Description: "compact context"},
	}
	m.completePicker.Cursor = 1
	m.Overlay = state.OverlayComplete

	if !input.HandleCompleteKey(&m.State, tea.KeyPressMsg{Code: tea.KeyTab}, &m.completePicker, m.input.Value(), m.input.SetValue, m.input.CursorEnd) {
		t.Fatal("expected tab to be handled")
	}
	if got := m.input.Value(); got != "/clear " {
		t.Errorf("input = %q, want /clear ", got)
	}
	if m.Overlay != state.OverlayNone || m.Complete != nil {
		t.Error("expected completion overlay to close after tab")
	}
}

func TestHandleCompleteKeyEnterDefersWhenReadyToSubmit(t *testing.T) {
	m := New(&deps.Deps{})
	m.input.SetValue("/context")
	m.Complete = []slash.Command{{Name: "context", Description: "panel"}}
	m.Overlay = state.OverlayComplete

	if input.HandleCompleteKey(&m.State, tea.KeyPressMsg{Code: tea.KeyEnter}, &m.completePicker, m.input.Value(), m.input.SetValue, m.input.CursorEnd) {
		t.Fatal("enter should submit command, not pick from list")
	}
}

func TestUpdateCompletionResumeNoRefetchWhenLoadedEmpty(t *testing.T) {
	m := New(&deps.Deps{Store: session.NewMemoryStore()})
	m.input.SetValue("/resume nomatch")
	m.Overlay = state.OverlayResume
	m.ResumeFilter = "nomatch"
	m.ResumeSessions = []session.Summary{}
	seqBefore := m.ResumeFilterSeq
	m.TestSyncResumePicker()
	overlayBefore := m.OverlayText

	for i := 0; i < 5; i++ {
		if cmd := input.UpdateCompletion(&m.State, m.input.Value(), &m.completePicker, &m.resumePicker); cmd != nil {
			t.Fatalf("iteration %d: expected no refetch cmd on cursor blink, got cmd", i)
		}
	}
	if m.ResumeFilterSeq != seqBefore {
		t.Fatalf("ResumeFilterSeq = %d, want %d (no new debounce tick)", m.ResumeFilterSeq, seqBefore)
	}
	if m.OverlayText != overlayBefore {
		t.Fatalf("overlay changed on repeat UpdateCompletion:\nbefore: %q\nafter:  %q", overlayBefore, m.OverlayText)
	}
	if !strings.Contains(m.OverlayText, "No matching sessions") {
		t.Fatalf("overlay = %q, want stable empty-list message", m.OverlayText)
	}
}

func TestUpdateCompletionPreservesCursorOnSameFilter(t *testing.T) {
	m := New(&deps.Deps{})
	m.input.SetValue("/c")
	m.CompleteFilterKey = "/c"
	m.Complete = slash.FilterCommands("/c")
	m.completePicker.Cursor = 1
	m.Overlay = state.OverlayComplete

	if cmd := input.UpdateCompletion(&m.State, m.input.Value(), &m.completePicker, &m.resumePicker); cmd != nil {
		t.Fatal("expected no cmd when filter unchanged")
	}
	if m.completePicker.Cursor != 1 {
		t.Fatalf("cursor = %d, want 1", m.completePicker.Cursor)
	}
}

// silence import
var _ = textinput.New

func TestUpdate_recoversLeakedMouseWheelWithoutInputGarbage(t *testing.T) {
	m := New(&deps.Deps{})
	seedChatLines(m, 50)
	m.chatVP.SetHeight(10)

	before := m.chatScrollY
	updated, cmd := m.Update(tea.KeyPressMsg{Text: "[<65;87;6M", Code: tea.KeyExtended})
	m = updated.(*Model)
	if m.input.Value() != "" {
		t.Fatalf("input = %q, want empty", m.input.Value())
	}
	if cmd == nil {
		t.Fatal("expected wheel scroll tick command")
	}
	for i := 0; i < 8 && m.chatScrollY <= before; i++ {
		updated, _ = m.Update(msg.WheelScrollTickMsg{})
		m = updated.(*Model)
	}
	if m.chatScrollY <= before {
		t.Fatalf("wheel down: chatScrollY = %d, want > %d", m.chatScrollY, before)
	}
}

func TestUpdate_recoversLeakedMouseWheelCharByChar(t *testing.T) {
	m := New(&deps.Deps{})
	seedChatLines(m, 50)
	m.chatVP.SetHeight(10)

	seq := "[<65;87;6M"
	var updated tea.Model = m
	var cmd tea.Cmd
	for _, r := range seq {
		updated, cmd = updated.Update(tea.KeyPressMsg{Text: string(r), Code: tea.KeyExtended})
	}
	m = updated.(*Model)
	if m.input.Value() != "" {
		t.Fatalf("input = %q, want empty", m.input.Value())
	}
	if m.mouseLeakBuf != "" {
		t.Fatalf("mouseLeakBuf = %q, want empty", m.mouseLeakBuf)
	}
	if cmd == nil {
		t.Fatal("expected wheel scroll tick command")
	}
	before := 0
	for i := 0; i < 8 && m.chatScrollY <= before; i++ {
		updated, _ = m.Update(msg.WheelScrollTickMsg{})
		m = updated.(*Model)
	}
	if m.chatScrollY <= before {
		t.Fatalf("wheel down: chatScrollY = %d, want > %d", m.chatScrollY, before)
	}
}

func TestUpdate_bracketThenNormalText(t *testing.T) {
	m := New(&deps.Deps{})
	updated, _ := m.Update(tea.KeyPressMsg{Text: "[", Code: tea.KeyExtended})
	m = updated.(*Model)
	if m.input.Value() != "" {
		t.Fatal("expected [ to be buffered, not inserted")
	}
	updated, _ = m.Update(tea.KeyPressMsg{Text: "foo", Code: tea.KeyExtended})
	m = updated.(*Model)
	if got := m.input.Value(); got != "[foo" {
		t.Fatalf("input = %q, want [foo", got)
	}
}
