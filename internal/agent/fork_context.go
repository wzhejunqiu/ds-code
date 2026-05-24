package agent

import (
	"context"

	"github.com/wzhejunqiu/ds-code/internal/llm"
)

type renderedSystemKey struct{}

// WithRenderedSystem stores the parent's pre-rendered system prompt for fork children.
func WithRenderedSystem(ctx context.Context, s string) context.Context {
	return context.WithValue(ctx, renderedSystemKey{}, s)
}

// RenderedSystemFromContext returns the parent's rendered system prompt, or "".
func RenderedSystemFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(renderedSystemKey{}).(string); ok {
		return v
	}
	return ""
}

type forkContextKey struct{}

// ForkContext carries parent-session data needed to build fork child messages.
type ForkContext struct {
	ParentMessages  []llm.Message
	ParentToolCalls []llm.ToolCall
}

// WithForkContext stores fork context for use by spawn.Handle.
func WithForkContext(ctx context.Context, fc ForkContext) context.Context {
	return context.WithValue(ctx, forkContextKey{}, fc)
}

// ForkContextFromContext returns previously stored fork context.
func ForkContextFromContext(ctx context.Context) (ForkContext, bool) {
	fc, ok := ctx.Value(forkContextKey{}).(ForkContext)
	return fc, ok
}
