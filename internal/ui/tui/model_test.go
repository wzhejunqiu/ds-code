package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/ui/slash"
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
		m := model{input: textinput.New()}
		m.input.SetValue(tt.input)
		if got := m.completionReadyToSubmit(); got != tt.want {
			t.Errorf("completionReadyToSubmit(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestHandleCompleteKeyTabSelectsFirst(t *testing.T) {
	m := model{input: textinput.New()}
	m.complete = []slash.Command{
		{Name: "clear", Description: "new session"},
		{Name: "compact", Description: "compact context"},
	}
	m.completeIdx = 1
	m.overlay = overlayComplete

	if !m.handleCompleteKey(tea.KeyMsg{Type: tea.KeyTab}) {
		t.Fatal("expected tab to be handled")
	}
	if got := m.input.Value(); got != "/clear " {
		t.Errorf("input = %q, want /clear ", got)
	}
	if m.overlay != overlayNone || m.complete != nil {
		t.Error("expected completion overlay to close after tab")
	}
}
