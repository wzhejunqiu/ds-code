package workspace

import (
	"context"

	"github.com/wzhejunqiu/ds-code/internal/billing"
	"github.com/wzhejunqiu/ds-code/internal/session/usageagg"
)

// SessionUsageView is cumulative token usage and cost for the status bar.
type SessionUsageView struct {
	Model              string  `json:"model"`
	PromptTokens       int64   `json:"promptTokens"`
	CompletionTokens   int64   `json:"completionTokens"`
	EstimatedCostCNY   float64 `json:"estimatedCostCNY"`
	EstimatedCostLabel string  `json:"estimatedCostLabel"`
}

// SessionUsage returns cumulative usage and estimated cost for a session.
func (m *Manager) SessionUsage(wsID, sessionID string) (SessionUsageView, error) {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return SessionUsageView{}, err
	}
	ctx := context.Background()
	sess, err := rt.store.Get(ctx, sessionID)
	if err != nil {
		return SessionUsageView{}, err
	}
	subStore, err := rt.app.OpenSubagentStoreForDesktop()
	if err != nil {
		return SessionUsageView{}, err
	}
	snap, err := usageagg.TotalForSession(ctx, rt.store, subStore, sessionID)
	if err != nil {
		return SessionUsageView{}, err
	}
	cost, err := usageagg.EstimateCostForSession(ctx, rt.store, subStore, sessionID)
	if err != nil {
		return SessionUsageView{}, err
	}
	return SessionUsageView{
		Model:              sess.Model,
		PromptTokens:       snap.PromptTokensTotal,
		CompletionTokens:   snap.CompletionTokensTotal,
		EstimatedCostCNY:   cost.TotalCNY,
		EstimatedCostLabel: billing.FormatCNY(cost.TotalCNY),
	}, nil
}
