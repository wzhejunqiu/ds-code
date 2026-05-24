package usageagg

import (
	"context"

	"github.com/wzhejunqiu/ds-code/internal/billing"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/session/subagentstore"
)

// CostBreakdown holds token snapshots and estimated CNY costs for a parent session.
type CostBreakdown struct {
	MainCNY     float64
	SubagentCNY float64
	TotalCNY    float64
	MainSnap    session.UsageSnapshot
	SubSnap     session.UsageSnapshot
	TotalSnap   session.UsageSnapshot
}

// EstimateCostForSession aggregates tokens and costs for main + subagent runs.
func EstimateCostForSession(ctx context.Context, main session.Store, sub subagentstore.Store, parentSessionID string) (CostBreakdown, error) {
	var out CostBreakdown
	total, err := TotalForSession(ctx, main, sub, parentSessionID)
	if err != nil {
		return out, err
	}
	out.TotalSnap = total

	sess, err := main.Get(ctx, parentSessionID)
	if err != nil {
		return out, err
	}
	out.MainSnap = session.UsageSnapshotFromSession(sess)
	out.SubSnap, err = SubagentOnly(ctx, sub, parentSessionID)
	if err != nil {
		return out, err
	}

	out.MainCNY = sumMainMessageCost(ctx, main, parentSessionID, sess)
	out.SubagentCNY = sumSubagentCost(ctx, sub, parentSessionID)
	out.TotalCNY = out.MainCNY + out.SubagentCNY
	if out.TotalCNY == 0 {
		out.TotalCNY = estimateFromSession(sess, out.TotalSnap)
	}
	return out, nil
}

func estimateFromSession(sess session.Session, snap session.UsageSnapshot) float64 {
	if sess.PricingSnapshotJSON != "" {
		ps := billing.ParseSnapshot(sess.Model, sess.PricingSnapshotJSON)
		return billing.EstimateCNYFromSnapshotTotals(ps, snap)
	}
	return billing.EstimateCNY(sess.Model, snap)
}

func sumMainMessageCost(ctx context.Context, main session.Store, sessionID string, sess session.Session) float64 {
	msgs, err := main.ListMessages(ctx, sessionID)
	if err != nil {
		return estimateFromSession(sess, session.UsageSnapshotFromSession(sess))
	}
	var sum float64
	var has bool
	for _, m := range msgs {
		if m.EstimatedCostCNY <= 0 {
			continue
		}
		sum += m.EstimatedCostCNY
		has = true
	}
	if has {
		return sum
	}
	return estimateFromSession(sess, session.UsageSnapshotFromSession(sess))
}

func sumSubagentCost(ctx context.Context, sub subagentstore.Store, parentSessionID string) float64 {
	if sub == nil {
		return 0
	}
	if sum, err := sub.SumEstimatedCostCNY(ctx, parentSessionID); err == nil && sum > 0 {
		return sum
	}
	runs, err := sub.ListRuns(ctx, parentSessionID)
	if err != nil || len(runs) == 0 {
		snap, err := SubagentOnly(ctx, sub, parentSessionID)
		if err != nil || snap.Billed == 0 {
			return 0
		}
		return billing.EstimateCNY(billing.DefaultSubagentModel, snap)
	}
	var total float64
	var hasRunCost bool
	for _, r := range runs {
		if r.EstimatedCostCNY > 0 {
			total += r.EstimatedCostCNY
			hasRunCost = true
			continue
		}
		rs := session.UsageSnapshot{
			PromptTokensTotal:         r.PromptTokensTotal,
			CompletionTokensTotal:     r.CompletionTokensTotal,
			PromptCacheHitTokensTotal: r.PromptCacheHitTokensTotal,
		}
		if r.PricingSnapshotJSON != "" {
			s := billing.ParseSnapshot(r.Model, r.PricingSnapshotJSON)
			total += billing.EstimateCNYFromSnapshotTotals(s, rs)
		} else {
			total += billing.EstimateCNY(r.Model, rs)
		}
	}
	if hasRunCost || total > 0 {
		return total
	}
	snap, err := SubagentOnly(ctx, sub, parentSessionID)
	if err != nil || snap.Billed == 0 {
		return 0
	}
	return billing.EstimateCNY(billing.DefaultSubagentModel, snap)
}
