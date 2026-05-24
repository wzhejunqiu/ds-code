// Package spawn orchestrates sub-agent lifecycle: routing, tool pool filtering,
// execution, and background management. It is the single entry point for all
// agent spawns, called from the "agent" built-in tool.
package spawn

import (
	"context"
)

type querySourceKey struct{}

// QuerySource labels who initiated a spawn for diagnostics and recursive-fork guards.
type QuerySource string

const (
	QuerySourceAgent       QuerySource = "agent:builtin:general-purpose"
	QuerySourceExplore     QuerySource = "agent:builtin:explore"
	QuerySourcePlan        QuerySource = "agent:builtin:plan"
	QuerySourceVerify      QuerySource = "agent:builtin:verification"
	QuerySourceFork        QuerySource = "agent:builtin:fork"
	QuerySourceSkill       QuerySource = "skill:fork"
)

// WithQuerySource attaches a spawn source label to ctx.
func WithQuerySource(ctx context.Context, qs QuerySource) context.Context {
	return context.WithValue(ctx, querySourceKey{}, qs)
}

// QuerySourceFromContext returns the spawn source label, defaulting to Agent.
func QuerySourceFromContext(ctx context.Context) QuerySource {
	if v, ok := ctx.Value(querySourceKey{}).(QuerySource); ok {
		return v
	}
	return QuerySourceAgent
}
