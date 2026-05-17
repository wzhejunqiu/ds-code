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

// runTurnAsync runs agent.RunTurn on a goroutine and forwards TurnCallbacks to the UI.
// Must not block on a full events channel — streaming uses trySend (drops if UI lags).
func runTurnAsync(d Deps, line string, events chan<- tea.Msg) {
	ctx, cancel := context.WithCancel(context.Background())
	events <- turnStartedMsg{cancel: cancel} // model stores cancel for Esc

	trySend := func(msg tea.Msg) {
		select {
		case events <- msg:
		default: // backpressure: skip delta rather than stall the agent
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
