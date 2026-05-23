package agent

import "context"

type sessionTitleGenKey struct{}

// WithSessionTitleGen controls whether the main agent RunTurn spawns async title generation.
// Default is enabled when unset.
func WithSessionTitleGen(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, sessionTitleGenKey{}, enabled)
}

// SessionTitleGenEnabled reports whether async title generation should run on the main agent.
func SessionTitleGenEnabled(ctx context.Context) bool {
	v, ok := ctx.Value(sessionTitleGenKey{}).(bool)
	if !ok {
		return true
	}
	return v
}
