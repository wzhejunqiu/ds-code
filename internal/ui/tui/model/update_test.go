package model

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	tuimsg "github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/state"
)

func TestKeyRelease_ignored(t *testing.T) {
	m := New(testDeps(true))
	m.TestInputSetValue("draft line")

	before := m.input.Value()
	updated, cmd := m.Update(tea.KeyReleaseMsg{Code: tea.KeyEnter})
	m = updated.(*Model)
	if cmd != nil {
		t.Fatal("KeyRelease should not enqueue commands")
	}
	if m.input.Value() != before {
		t.Fatalf("input changed on KeyRelease: %q", m.input.Value())
	}
	if m.Running {
		t.Fatal("KeyRelease should not submit")
	}
}

func TestView_noSideEffects(t *testing.T) {
	m := New(testDeps(true))
	seedChatLines(m, 30)
	m.chatScrollY = 5

	beforeScroll := m.chatScrollY
	beforePlain := len(m.plainLines)
	beforeTotal := m.lineCatalog.TotalLines()

	_ = m.View()

	if m.chatScrollY != beforeScroll {
		t.Fatalf("chatScrollY changed: %d -> %d", beforeScroll, m.chatScrollY)
	}
	if len(m.plainLines) != beforePlain {
		t.Fatalf("plainLines length changed: %d -> %d", beforePlain, len(m.plainLines))
	}
	if m.lineCatalog.TotalLines() != beforeTotal {
		t.Fatalf("catalog total changed: %d -> %d", beforeTotal, m.lineCatalog.TotalLines())
	}
}

func TestListenPrompt_permissionAsk(t *testing.T) {
	ch := make(chan permission.PromptRequest, 1)
	m := New(testDeps(true))
	m.Deps.PromptCh = ch

	initCmd := m.Init()
	_ = initCmd

	ch <- permission.PromptRequest{Tool: "shell", Summary: "rm -rf /"}
	if listenCmd := m.listenPrompt(); listenCmd != nil {
		if msg := listenCmd(); msg != nil {
			updated, _ := m.Update(msg)
			m = updated.(*Model)
		}
	}

	if m.Overlay != state.OverlayPrompt {
		t.Fatalf("overlay = %v, want OverlayPrompt", m.Overlay)
	}
	if m.Prompt == nil {
		t.Fatal("expected prompt request stored")
	}
	if m.Prompt.Tool != "shell" {
		t.Fatalf("prompt tool = %q", m.Prompt.Tool)
	}

	// Direct message path should also open the overlay.
	m.Overlay = state.OverlayNone
	m.Prompt = nil
	req := permission.PromptRequest{Tool: "apply_patch", Summary: "edit foo.go"}
	updated, nextCmd := m.Update(tuimsg.PromptRequestMsg{Req: req})
	m = updated.(*Model)
	if m.Overlay != state.OverlayPrompt {
		t.Fatalf("overlay = %v, want OverlayPrompt after PromptRequestMsg", m.Overlay)
	}
	if nextCmd == nil {
		t.Fatal("expected follow-up listenPrompt cmd")
	}
}
