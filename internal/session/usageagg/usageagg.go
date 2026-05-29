// Package usageagg combines main-session and agent-run token totals.
package usageagg

import (
	"context"

	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

// TotalForSession returns cumulative usage for the main agent plus all agent runs
// under parentSessionID (for status bar, /context billing, compact thresholds).
func TotalForSession(ctx context.Context, main session.Store, sub subagentstore.Store, parentSessionID string) (session.UsageSnapshot, error) {
	sess, err := main.Get(ctx, parentSessionID)
	if err != nil {
		return session.UsageSnapshot{}, err
	}
	snap := session.UsageSnapshotFromSession(sess)
	if sub == nil {
		return snap, nil
	}
	su, err := sub.SumUsage(ctx, parentSessionID)
	if err != nil {
		return snap, err
	}
	snap.PromptTokensTotal += int64(su.PromptTokens)
	snap.CompletionTokensTotal += int64(su.CompletionTokens)
	snap.PromptCacheHitTokensTotal += int64(su.PromptCacheHitTokens)
	snap.Billed = int(snap.PromptTokensTotal + snap.CompletionTokensTotal)
	return snap, nil
}

// SubagentOnly returns usage summed from all agent_runs for a parent session.
func SubagentOnly(ctx context.Context, sub subagentstore.Store, parentSessionID string) (session.UsageSnapshot, error) {
	if sub == nil {
		return session.UsageSnapshot{}, nil
	}
	u, err := sub.SumUsage(ctx, parentSessionID)
	if err != nil {
		return session.UsageSnapshot{}, err
	}
	snap := session.UsageSnapshot{
		PromptTokensTotal:         int64(u.PromptTokens),
		CompletionTokensTotal:     int64(u.CompletionTokens),
		PromptCacheHitTokensTotal: int64(u.PromptCacheHitTokens),
	}
	snap.Billed = int(snap.PromptTokensTotal + snap.CompletionTokensTotal)
	return snap, nil
}

// AgentTypeUsage is token usage grouped by agent type for one parent session.
type AgentTypeUsage struct {
	AgentType string
	Snapshot  session.UsageSnapshot
}

// UsageByAgentType aggregates agent_runs usage grouped by AgentType.
func UsageByAgentType(ctx context.Context, sub subagentstore.Store, parentSessionID string) ([]AgentTypeUsage, error) {
	if sub == nil {
		return nil, nil
	}
	runs, err := sub.ListRuns(ctx, parentSessionID)
	if err != nil {
		return nil, err
	}
	byType := make(map[string]*session.UsageSnapshot)
	order := make([]string, 0)
	for _, r := range runs {
		if _, ok := byType[r.AgentType]; !ok {
			byType[r.AgentType] = &session.UsageSnapshot{}
			order = append(order, r.AgentType)
		}
		s := byType[r.AgentType]
		s.PromptTokensTotal += r.PromptTokensTotal
		s.CompletionTokensTotal += r.CompletionTokensTotal
		s.PromptCacheHitTokensTotal += r.PromptCacheHitTokensTotal
	}
	out := make([]AgentTypeUsage, 0, len(order))
	for _, typ := range order {
		snap := *byType[typ]
		snap.Billed = int(snap.PromptTokensTotal + snap.CompletionTokensTotal)
		out = append(out, AgentTypeUsage{AgentType: typ, Snapshot: snap})
	}
	return out, nil
}
