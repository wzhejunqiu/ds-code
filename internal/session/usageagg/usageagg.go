// Package usageagg combines main-session and subagent-run token totals.
package usageagg

import (
	"context"

	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

// TotalForSession returns cumulative usage for the main agent plus all subagent runs
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

// SubagentOnly returns usage summed from all subagent_runs for a parent session.
func SubagentOnly(ctx context.Context, sub subagentstore.Store, parentSessionID string) (session.UsageSnapshot, error) {
	return subagentUsage(ctx, sub, parentSessionID, true)
}

// SubagentTaskOnly returns usage from task subagent runs (excludes session-title runs).
func SubagentTaskOnly(ctx context.Context, sub subagentstore.Store, parentSessionID string) (session.UsageSnapshot, error) {
	return subagentUsage(ctx, sub, parentSessionID, false)
}

func subagentUsage(ctx context.Context, sub subagentstore.Store, parentSessionID string, includeTitle bool) (session.UsageSnapshot, error) {
	if sub == nil {
		return session.UsageSnapshot{}, nil
	}
	if includeTitle {
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
	runs, err := sub.ListRuns(ctx, parentSessionID)
	if err != nil {
		return session.UsageSnapshot{}, err
	}
	var snap session.UsageSnapshot
	for _, r := range runs {
		if r.RunKind == subagentstore.RunKindTitle {
			continue
		}
		snap.PromptTokensTotal += r.PromptTokensTotal
		snap.CompletionTokensTotal += r.CompletionTokensTotal
		snap.PromptCacheHitTokensTotal += r.PromptCacheHitTokensTotal
	}
	snap.Billed = int(snap.PromptTokensTotal + snap.CompletionTokensTotal)
	return snap, nil
}
