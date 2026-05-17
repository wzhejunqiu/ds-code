package tui

import (
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/session"
	"github.com/hejunqiu/ds-code/internal/ui/slash"
)

func TestDismissOverlayPromptRejectsPermission(t *testing.T) {
	reply := make(chan bool, 1)
	m := model{
		overlay: overlayPrompt,
		prompt: &permission.PromptRequest{
			Tool:  "Bash",
			Reply: reply,
		},
		overlayText: "prompt",
	}

	m.dismissOverlay()

	if m.overlay != overlayNone {
		t.Fatalf("overlay = %v, want overlayNone", m.overlay)
	}
	if m.prompt != nil {
		t.Fatal("expected prompt cleared")
	}
	select {
	case allow := <-reply:
		if allow {
			t.Fatal("expected dismiss to reject permission")
		}
	default:
		t.Fatal("expected reply on prompt channel")
	}
}

func TestDismissOverlayCompleteClearsPicker(t *testing.T) {
	m := model{input: textinput.New()}
	m.overlay = overlayComplete
	m.complete = []slash.Command{{Name: "help", Description: "help"}}
	m.syncCompleteOverlay()

	m.dismissOverlay()

	if m.overlay != overlayNone || len(m.complete) != 0 || m.overlayText != "" {
		t.Fatalf("overlay=%v complete=%v text=%q", m.overlay, m.complete, m.overlayText)
	}
}

func TestUpdateResumeListErrorClearsPicker(t *testing.T) {
	m := model{}
	m.overlay = overlayResume
	m.resumeSessions = []session.Summary{{ID: "x"}}
	m.resumeFilter = "q"
	m.resumeFilterSeq = 1

	m.updateResumeList(resumeListMsg{filter: "q", seq: 1, err: errors.New("list failed")})

	if m.overlay != overlayNone || len(m.resumeSessions) != 0 {
		t.Fatalf("overlay=%v sessions=%d", m.overlay, len(m.resumeSessions))
	}
}

func TestHandleCompleteKeyEnterDefersWhenReadyToSubmit(t *testing.T) {
	m := model{input: textinput.New()}
	m.input.SetValue("/context")
	m.complete = []slash.Command{{Name: "context", Description: "panel"}}
	m.overlay = overlayComplete

	if m.handleCompleteKey(tea.KeyMsg{Type: tea.KeyEnter}) {
		t.Fatal("enter should submit command, not pick from list")
	}
}

func TestUpdateCompletionPreservesCursorOnSameFilter(t *testing.T) {
	m := model{input: textinput.New()}
	m.input.SetValue("/c")
	m.completeFilterKey = "/c"
	m.complete = slash.FilterCommands("/c")
	m.completePicker.Cursor = 1
	m.overlay = overlayComplete

	if cmd := m.updateCompletion(); cmd != nil {
		t.Fatal("expected no cmd when filter unchanged")
	}
	if m.completePicker.Cursor != 1 {
		t.Fatalf("cursor = %d, want 1", m.completePicker.Cursor)
	}
}
