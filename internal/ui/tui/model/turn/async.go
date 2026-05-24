package turn

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/msg"
	"go.uber.org/zap"
)

const (
	agentEventRetryInterval = 5 * time.Millisecond
	agentEventMaxRetries    = 200
)

type streamBuffer struct {
	mu               sync.Mutex
	pendingContent   string
	pendingReasoning string
}

func (b *streamBuffer) appendContent(delta string) {
	b.mu.Lock()
	b.pendingContent += delta
	b.mu.Unlock()
}

func (b *streamBuffer) appendReasoning(delta string) {
	b.mu.Lock()
	b.pendingReasoning += delta
	b.mu.Unlock()
}

func (b *streamBuffer) trySendContent(events chan<- tea.Msg) {
	b.mu.Lock()
	if b.pendingContent == "" {
		b.mu.Unlock()
		return
	}
	payload := b.pendingContent
	b.mu.Unlock()
	select {
	case events <- msg.StreamContentMsg{Delta: payload}:
		b.mu.Lock()
		if len(b.pendingContent) >= len(payload) {
			b.pendingContent = b.pendingContent[len(payload):]
		}
		b.mu.Unlock()
	default:
		logging.L().Debug("stream content dropped", zap.Int("pending_chars", len(payload)))
	}
}

func (b *streamBuffer) trySendReasoning(events chan<- tea.Msg) {
	b.mu.Lock()
	if b.pendingReasoning == "" {
		b.mu.Unlock()
		return
	}
	payload := b.pendingReasoning
	b.mu.Unlock()
	select {
	case events <- msg.StreamReasoningMsg{Delta: payload}:
		b.mu.Lock()
		if len(b.pendingReasoning) >= len(payload) {
			b.pendingReasoning = b.pendingReasoning[len(payload):]
		}
		b.mu.Unlock()
	default:
		logging.L().Debug("stream reasoning dropped", zap.Int("pending_chars", len(payload)))
	}
}

func (b *streamBuffer) flush(events chan<- tea.Msg) {
	b.mu.Lock()
	c := b.pendingContent
	r := b.pendingReasoning
	b.pendingContent = ""
	b.pendingReasoning = ""
	b.mu.Unlock()
	if c != "" {
		sendAgentEvent(events, msg.StreamContentMsg{Delta: c}, false)
	}
	if r != "" {
		sendAgentEvent(events, msg.StreamReasoningMsg{Delta: r}, false)
	}
}

// RunAsync runs agent.RunTurn on a goroutine and forwards TurnCallbacks to the UI.
func RunAsync(d deps.Deps, line string, events chan<- tea.Msg, wg *sync.WaitGroup) {
	if wg != nil {
		wg.Add(1)
		defer wg.Done()
	}
	start := time.Now()
	logging.L().Debug("tui turn async start",
		zap.String("session_id", d.SessionID),
		zap.Int("prompt_chars", len(line)),
	)
	ctx, cancel := context.WithCancel(context.Background())
	sendAgentEvent(events, msg.TurnStartedMsg{Cancel: cancel}, true)

	var buf streamBuffer

	cb := &agent.TurnCallbacks{
		OnContentDelta: func(s string) {
			buf.appendContent(s)
			buf.trySendContent(events)
		},
		OnReasoningDelta: func(s string) {
			buf.appendReasoning(s)
			buf.trySendReasoning(events)
		},
		OnToolStart: func(name, args, command string) {
			buf.flush(events)
			sendAgentEvent(events, msg.ToolStartMsg{Name: name, Args: args, Command: command}, false)
		},
		OnToolEnd: func(name, args, command, result string, isError bool) {
			sendAgentEvent(events, msg.ToolEndMsg{Name: name, Args: args, Command: command, Result: result, IsError: isError}, true)
		},
		OnAssistantSegmentEnd: func() {
			buf.flush(events)
			sendAgentEvent(events, msg.AssistantSegmentEndMsg{}, false)
		},
		OnPlanningStart: func() {
			sendAgentEvent(events, msg.PlanningStartMsg{}, true)
		},
		OnPlanningEnd: func() {
			sendAgentEvent(events, msg.PlanningEndMsg{}, true)
		},
		OnSubagentStart: func(id, label, prompt, agentType string, background bool) {
			sendAgentEvent(events, msg.SubagentStartMsg{
				ID: id, Label: label, Prompt: prompt, AgentType: agentType, Background: background,
			}, true)
		},
		OnSubagentEnd: func(id, summary string, err error) {
			sendAgentEvent(events, msg.SubagentEndMsg{ID: id, Summary: summary, Err: err}, true)
		},
		OnSubagentToolStart: func(id, name, args, command string) {
			sendAgentEvent(events, msg.SubagentToolStartMsg{SubagentID: id, Name: name, Args: args, Command: command}, false)
		},
		OnSubagentToolEnd: func(id, name, args, command, result string, isError bool) {
			sendAgentEvent(events, msg.SubagentToolEndMsg{
				SubagentID: id, Name: name, Args: args, Command: command, Result: result, IsError: isError,
			}, true)
		},
	}

	result, err := d.Runner.RunTurn(ctx, d.SessionID, line, cb)
	buf.flush(events)
	sendAgentEvent(events, msg.TurnDoneMsg{Result: result, Err: err}, true)
	subRounds := 0
	if result != nil {
		subRounds = result.SubRounds
	}
	logging.L().Debug("tui turn async done",
		zap.String("session_id", d.SessionID),
		zap.Int64("duration_ms", time.Since(start).Milliseconds()),
		zap.Int("sub_rounds", subRounds),
		zap.Bool("ok", err == nil),
	)
}

func sendAgentEvent(events chan<- tea.Msg, m tea.Msg, critical bool) {
	if !critical {
		select {
		case events <- m:
		default:
			logging.L().Debug("agent event dropped",
				zap.String("type", fmt.Sprintf("%T", m)),
			)
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
	logging.L().Error("agent event dropped after retries",
		zap.String("type", fmt.Sprintf("%T", m)),
		zap.Int("retries", agentEventMaxRetries),
	)
}
