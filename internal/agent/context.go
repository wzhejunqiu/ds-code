package agent

import (
	"context"
	"sync/atomic"
	"time"
)

type turnCallbacksKey struct{}

type activeTurnKey struct{}

// activeTurnHolder tracks nested RunTurn scopes (parent + subagent) with a refcount.
type activeTurnHolder struct {
	count atomic.Int32
}

type toolInvocationKey struct{}

// ToolInvocation identifies the parent session and tool_call for a running tool.
type ToolInvocation struct {
	SessionID   string
	ToolCallID  string
	ParentModel string // parent session.Model for sub-agent model resolution
}

// WithActiveTurn marks ctx as inside an active RunTurn (used for notification priority).
func WithActiveTurn(ctx context.Context) context.Context {
	if h, ok := ctx.Value(activeTurnKey{}).(*activeTurnHolder); ok {
		h.count.Add(1)
		return ctx
	}
	h := &activeTurnHolder{}
	h.count.Store(1)
	return context.WithValue(ctx, activeTurnKey{}, h)
}

// WithoutActiveTurn decrements the active turn refcount on the shared holder.
func WithoutActiveTurn(ctx context.Context) context.Context {
	if h, ok := ctx.Value(activeTurnKey{}).(*activeTurnHolder); ok {
		for {
			v := h.count.Load()
			if v <= 0 {
				break
			}
			if h.count.CompareAndSwap(v, v-1) {
				break
			}
		}
	}
	return ctx
}

// InActiveTurn reports whether the parent runner is executing a user turn.
func InActiveTurn(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	h, ok := ctx.Value(activeTurnKey{}).(*activeTurnHolder)
	return ok && h.count.Load() > 0
}

// WithToolInvocation stores parent session, tool call id, and optional parent model for tools (e.g. agent).
func WithToolInvocation(ctx context.Context, sessionID, toolCallID, parentModel string) context.Context {
	if sessionID == "" && toolCallID == "" {
		return ctx
	}
	return context.WithValue(ctx, toolInvocationKey{}, ToolInvocation{
		SessionID:   sessionID,
		ToolCallID:  toolCallID,
		ParentModel: parentModel,
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
		OnToolStart: func(name, args, command string, timeoutDeadline time.Time) {
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
