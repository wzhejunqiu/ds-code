package bridge

import (
	"sync"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/llm"
)

// EmitterHub routes turn callbacks to main and per-subagent stream emitters.
type EmitterHub struct {
	main *StreamEmitter
	subs map[string]*StreamEmitter
	mu   sync.Mutex

	turnID      string
	workspaceID string
	emit        EmitFunc
}

// NewEmitterHub creates the main stream emitter and lazily allocates subagent streams.
func NewEmitterHub(turnID, workspaceID string, emit EmitFunc) *EmitterHub {
	main := NewStreamEmitter(StreamEmitterOptions{
		TurnID:      turnID,
		StreamID:    "main",
		WorkspaceID: workspaceID,
		Emit:        emit,
	})
	return &EmitterHub{
		main:        main,
		subs:        make(map[string]*StreamEmitter),
		turnID:      turnID,
		workspaceID: workspaceID,
		emit:        emit,
	}
}

func (h *EmitterHub) Main() *StreamEmitter {
	return h.main
}

func (h *EmitterHub) subagent(id string) *StreamEmitter {
	sid := "subagent:" + id
	h.mu.Lock()
	defer h.mu.Unlock()
	if em, ok := h.subs[sid]; ok {
		return em
	}
	em := NewStreamEmitter(StreamEmitterOptions{
		TurnID:      h.turnID,
		StreamID:    sid,
		WorkspaceID: h.workspaceID,
		Emit:        h.emit,
	})
	h.subs[sid] = em
	return em
}

// TurnCallbacks builds callbacks with subagent tool events routed by streamId.
func (h *EmitterHub) TurnCallbacks(sessionID string) *agent.TurnCallbacks {
	em := h.main
	return &agent.TurnCallbacks{
		OnContentDelta: func(s string) {
			em.OnDelta(KindContentDelta, s)
		},
		OnReasoningDelta: func(s string) {
			em.OnDelta(KindReasoningDelta, s)
		},
		OnToolStart: func(name, args, command string, timeoutDeadline time.Time) {
			em.Flush(true)
			var deadline int64
			if !timeoutDeadline.IsZero() {
				deadline = timeoutDeadline.UnixMilli()
			}
			em.EmitNonCritical(KindToolStart, ToolStartPayload{
				Name: name, Args: args, Command: command, TimeoutDeadline: deadline,
			})
		},
		OnToolEnd: func(name, args, command, result string, isError bool) {
			em.EmitCritical(KindToolEnd, ToolEndPayload{
				Name: name, Args: args, Command: command, Result: result, IsError: isError,
			})
		},
		OnAssistantSegmentEnd: func() {
			em.Flush(true)
			em.EmitNonCritical(KindAssistantSegmentEnd, struct{}{})
		},
		OnPlanningStart: func() {
			em.EmitCritical(KindPlanningStart, struct{}{})
		},
		OnPlanningEnd: func() {
			em.EmitCritical(KindPlanningEnd, struct{}{})
		},
		OnSubagentStart: func(id, label, prompt, agentType string, background bool) {
			em.EmitCritical(KindSubagentStart, SubagentStartPayload{
				ID: id, Label: label, Prompt: prompt, AgentType: agentType, Background: background,
			})
		},
		OnSubagentEnd: func(id, summary string, err error) {
			payload := SubagentEndPayload{ID: id, Summary: summary}
			if err != nil {
				payload.Error = err.Error()
			}
			em.EmitCritical(KindSubagentEnd, payload)
		},
		OnSubagentToolStart: func(id, name, args, command string) {
			sub := h.subagent(id)
			sub.Flush(true)
			sub.EmitNonCritical(KindSubagentToolStart, SubagentToolStartPayload{
				SubagentID: id, Name: name, Args: args, Command: command,
			})
		},
		OnSubagentToolEnd: func(id, name, args, command, result string, isError bool) {
			sub := h.subagent(id)
			sub.EmitCritical(KindSubagentToolEnd, SubagentToolEndPayload{
				SubagentID: id, Name: name, Args: args, Command: command, Result: result, IsError: isError,
			})
		},
		OnBackgroundAgentComplete: func(agentID string) {
			em.EmitCritical(KindSubagentEnd, SubagentEndPayload{ID: agentID, Summary: "background complete"})
		},
		OnUsageUpdate: func(u llm.Usage) {
			em.EmitCritical(KindUsageUpdate, u)
		},
	}
}

// EmitTurnStarted sends turn.started on the main stream.
func (h *EmitterHub) EmitTurnStarted(sessionID, contentFormat string) {
	EmitTurnStarted(h.main, sessionID, contentFormat)
}

// EmitTurnDone flushes all streams and sends turn.done on main.
func (h *EmitterHub) EmitTurnDone(result *agent.TurnResult, err error) {
	h.mu.Lock()
	subs := make([]*StreamEmitter, 0, len(h.subs))
	for _, em := range h.subs {
		subs = append(subs, em)
	}
	h.mu.Unlock()
	for _, em := range subs {
		em.Flush(true)
	}
	EmitTurnDone(h.main, result, err)
}

// EmitSystemNotice sends a non-turn system message on the main stream.
func (h *EmitterHub) EmitSystemNotice(text string) {
	h.main.EmitCritical(KindSystemNotice, SystemNoticePayload{Text: text})
}
