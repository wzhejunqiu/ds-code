package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/agent"
)

// Run starts the interactive Bubble Tea UI.
func Run(deps Deps) error {
	events := make(chan tea.Msg, 64)
	deps.Events = events
	d := deps
	m := newModel(&d)
	p := tea.NewProgram(&m, tea.WithAltScreen())
	go func() {
		for msg := range events {
			p.Send(msg)
		}
	}()
	_, err := p.Run()
	return err
}

func runTurnAsync(d Deps, line string, events chan<- tea.Msg) {
	ctx, cancel := context.WithCancel(context.Background())
	// cancel is stored on model via turnStartedMsg - send cancel func
	events <- turnStartedMsg{cancel: cancel}

	cb := &agent.TurnCallbacks{
		OnContentDelta: func(s string) {
			select {
			case events <- streamContentMsg{delta: s}:
			default:
			}
		},
		OnReasoningDelta: func(s string) {
			select {
			case events <- streamReasoningMsg{delta: s}:
			default:
			}
		},
		OnToolStart: func(name string) {
			events <- toolStartMsg{name: name}
		},
		OnToolEnd: func(name, preview string) {
			events <- toolEndMsg{name: name, preview: preview}
		},
	}

	result, err := d.Runner.RunTurn(ctx, d.SessionID, line, cb)
	events <- turnDoneMsg{result: result, err: err}
}
