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
	close(events)
	return err
}

// runTurnAsync runs agent.RunTurn on a goroutine and forwards TurnCallbacks to the UI.
// Stream deltas may be dropped when the channel is full; lifecycle messages are retried.
func runTurnAsync(d Deps, line string, events chan<- tea.Msg) {
	ctx, cancel := context.WithCancel(context.Background())
	sendAgentEvent(events, turnStartedMsg{cancel: cancel}, true)

	cb := &agent.TurnCallbacks{
		OnContentDelta: func(s string) {
			sendAgentEvent(events, streamContentMsg{delta: s}, false)
		},
		OnReasoningDelta: func(s string) {
			sendAgentEvent(events, streamReasoningMsg{delta: s}, false)
		},
		OnToolStart: func(name, args, command string) {
			sendAgentEvent(events, toolStartMsg{name: name, args: args, command: command}, false)
		},
		OnToolEnd: func(name, args, command, result string, isError bool) {
			sendAgentEvent(events, toolEndMsg{name: name, args: args, command: command, result: result, isError: isError}, false)
		},
		OnAssistantSegmentEnd: func() {
			sendAgentEvent(events, assistantSegmentEndMsg{}, false)
		},
		OnPlanningStart: func() {
			sendAgentEvent(events, planningStartMsg{}, false)
		},
		OnPlanningEnd: func() {
			sendAgentEvent(events, planningEndMsg{}, false)
		},
	}

	result, err := d.Runner.RunTurn(ctx, d.SessionID, line, cb)
	sendAgentEvent(events, turnDoneMsg{result: result, err: err}, true)
}
