package bridge

import (
	"context"
	"errors"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/llm"
)

// TurnCallbacksOptions configures adapter construction.
type TurnCallbacksOptions struct {
	Emitter   *StreamEmitter
	SessionID string
}

// TurnCallbacks builds agent.TurnCallbacks that forward to the StreamEmitter.
func TurnCallbacks(opts TurnCallbacksOptions) *agent.TurnCallbacks {
	em := opts.Emitter
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
			em.EmitNonCritical(KindSubagentToolStart, SubagentToolStartPayload{
				SubagentID: id, Name: name, Args: args, Command: command,
			})
		},
		OnSubagentToolEnd: func(id, name, args, command, result string, isError bool) {
			em.EmitCritical(KindSubagentToolEnd, SubagentToolEndPayload{
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

// EmitTurnStarted sends turn.started (critical).
func EmitTurnStarted(em *StreamEmitter, sessionID, contentFormat string) {
	em.EmitCritical(KindTurnStarted, TurnStartedPayload{SessionID: sessionID, ContentFormat: contentFormat})
}

// EmitTurnDone sends turn.done after RunTurn returns.
func EmitTurnDone(em *StreamEmitter, result *agent.TurnResult, err error) {
	em.Flush(true)
	payload := TurnDonePayload{}
	if result != nil {
		payload.SubRounds = result.SubRounds
		payload.FinalChars = len(result.FinalContent)
	}
	if err != nil {
		payload.Error = err.Error()
		if errors.Is(err, context.Canceled) {
			payload.Cancelled = true
		}
	}
	em.EmitCritical(KindTurnDone, payload)
}
