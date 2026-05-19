package agent

import (
	"context"
)

type turnCallbacksKey struct{}

type toolInvocationKey struct{}

// ToolInvocation identifies the parent session and tool_call for a running tool.
type ToolInvocation struct {
	SessionID  string
	ToolCallID string
}

// WithToolInvocation stores parent session and tool call id for tools (e.g. task).
func WithToolInvocation(ctx context.Context, sessionID, toolCallID string) context.Context {
	if sessionID == "" && toolCallID == "" {
		return ctx
	}
	return context.WithValue(ctx, toolInvocationKey{}, ToolInvocation{
		SessionID:  sessionID,
		ToolCallID: toolCallID,
	})
}

// ToolInvocationFromContext returns invocation metadata from executeTool.
func ToolInvocationFromContext(ctx context.Context) (ToolInvocation, bool) {
	if ctx == nil {
		return ToolInvocation{}, false
	}
	inv, ok := ctx.Value(toolInvocationKey{}).(ToolInvocation)
	return inv, ok && (inv.SessionID != "" || inv.ToolCallID != "")
}

// WithTurnCallbacks stores turn UI callbacks on ctx for nested tools (e.g. task subagent).
func WithTurnCallbacks(ctx context.Context, cb *TurnCallbacks) context.Context {
	if cb == nil {
		return ctx
	}
	return context.WithValue(ctx, turnCallbacksKey{}, cb)
}

// TurnCallbacksFromContext returns callbacks attached by WithTurnCallbacks.
func TurnCallbacksFromContext(ctx context.Context) *TurnCallbacks {
	if ctx == nil {
		return nil
	}
	cb, _ := ctx.Value(turnCallbacksKey{}).(*TurnCallbacks)
	return cb
}

// SubagentToolCallbacks forwards nested tool events for a task subagent scope.
func SubagentToolCallbacks(parent *TurnCallbacks, subagentID string) *TurnCallbacks {
	if parent == nil || subagentID == "" {
		return nil
	}
	return &TurnCallbacks{
		OnToolStart: func(name, args, command string) {
			if parent.OnSubagentToolStart != nil {
				parent.OnSubagentToolStart(subagentID, name, args, command)
			}
		},
		OnToolEnd: func(name, args, command, result string, isError bool) {
			if parent.OnSubagentToolEnd != nil {
				parent.OnSubagentToolEnd(subagentID, name, args, command, result, isError)
			}
		},
	}
}
