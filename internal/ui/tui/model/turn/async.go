package turn

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hejunqiu/ds-code/internal/agent"
	"github.com/hejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/hejunqiu/ds-code/internal/ui/tui/model/msg"
)

const (
	agentEventRetryInterval = 5 * time.Millisecond
	agentEventMaxRetries    = 200
)

// RunAsync runs agent.RunTurn on a goroutine and forwards TurnCallbacks to the UI.
func RunAsync(d deps.Deps, line string, events chan<- tea.Msg) {
	ctx, cancel := context.WithCancel(context.Background())
	sendAgentEvent(events, msg.TurnStartedMsg{Cancel: cancel}, true)

	cb := &agent.TurnCallbacks{
		OnContentDelta: func(s string) {
			sendAgentEvent(events, msg.StreamContentMsg{Delta: s}, false)
		},
		OnReasoningDelta: func(s string) {
			sendAgentEvent(events, msg.StreamReasoningMsg{Delta: s}, false)
		},
		OnToolStart: func(name, args, command string) {
			sendAgentEvent(events, msg.ToolStartMsg{Name: name, Args: args, Command: command}, false)
		},
		OnToolEnd: func(name, args, command, result string, isError bool) {
			sendAgentEvent(events, msg.ToolEndMsg{Name: name, Args: args, Command: command, Result: result, IsError: isError}, false)
		},
		OnAssistantSegmentEnd: func() {
			sendAgentEvent(events, msg.AssistantSegmentEndMsg{}, false)
		},
		OnPlanningStart: func() {
			sendAgentEvent(events, msg.PlanningStartMsg{}, false)
		},
		OnPlanningEnd: func() {
			sendAgentEvent(events, msg.PlanningEndMsg{}, false)
		},
	}

	result, err := d.Runner.RunTurn(ctx, d.SessionID, line, cb)
	sendAgentEvent(events, msg.TurnDoneMsg{Result: result, Err: err}, true)
}

func sendAgentEvent(events chan<- tea.Msg, m tea.Msg, critical bool) {
	if !critical {
		select {
		case events <- m:
		default:
		}
		return
	}
	for i := 0; i < agentEventMaxRetries; i++ {
		select {
		case events <- m:
			return
		default:
			time.Sleep(agentEventRetryInterval)
		}
	}
	events <- m
}
