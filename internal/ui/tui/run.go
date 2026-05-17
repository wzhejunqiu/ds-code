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

	trySend := func(msg tea.Msg) {
		select {
		case events <- msg:
		default:
		}
	}

	cb := &agent.TurnCallbacks{
		OnContentDelta: func(s string) {
			trySend(streamContentMsg{delta: s})
		},
		OnReasoningDelta: func(s string) {
			trySend(streamReasoningMsg{delta: s})
		},
		OnToolStart: func(name, args, command string) {
			trySend(toolStartMsg{name: name, args: args, command: command})
		},
		OnToolEnd: func(name, args, command, result string, isError bool) {
			trySend(toolEndMsg{name: name, args: args, command: command, result: result, isError: isError})
		},
		OnAssistantSegmentEnd: func() {
			trySend(assistantSegmentEndMsg{})
		},
		OnPlanningStart: func() {
			trySend(planningStartMsg{})
		},
		OnPlanningEnd: func() {
			trySend(planningEndMsg{})
		},
	}

	result, err := d.Runner.RunTurn(ctx, d.SessionID, line, cb)
	events <- turnDoneMsg{result: result, err: err}
}
