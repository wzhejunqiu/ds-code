package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hejunqiu/ds-code/internal/billing"
	"github.com/hejunqiu/ds-code/internal/config"
	ctxpkg "github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/session"
)

// ContextPanelData holds inputs for /context rendering.
type ContextPanelData struct {
	Session   session.Session
	Snapshot  session.UsageSnapshot
	Breakdown ctxpkg.ContextBreakdown
	Threshold int
	Estimated bool
}

// BuildContextPanelData assembles panel data for the current session.
func BuildContextPanelData(cfg *config.Config, sess session.Session, view *ctxpkg.APIContextView) (ContextPanelData, error) {
	bd, err := ctxpkg.CountBreakdown(view)
	if err != nil {
		return ContextPanelData{}, err
	}
	ratio := cfg.Context.CompactThresholdRatio
	if ratio <= 0 {
		ratio = 0.80
	}
	threshold := int(float64(bd.Window) * ratio)
	return ContextPanelData{
		Session:   sess,
		Snapshot:  session.UsageSnapshotFromSession(sess),
		Breakdown: bd,
		Threshold: threshold,
		Estimated: bd.Estimated,
	}, nil
}

// FormatContextPanel renders the two-layer /context view for TUI or REPL.
func FormatContextPanel(d ContextPanelData) string {
	var b strings.Builder
	cost := billing.FormatUSD(billing.EstimateUSD(d.Session.Model, d.Snapshot))
	fmt.Fprintf(&b, "Session %s\n", d.Session.ID)
	fmt.Fprintf(&b, "Billed (cumulative)  %d tokens   est. %s\n", d.Snapshot.Billed, cost)
	fmt.Fprintf(&b, "  prompt in      %d\n", d.Snapshot.PromptTokensTotal)
	fmt.Fprintf(&b, "  completion out %d\n", d.Snapshot.CompletionTokensTotal)
	fmt.Fprintf(&b, "  cache hit      %d\n", d.Snapshot.PromptCacheHitTokensTotal)
	fmt.Fprintf(&b, "Compact: prompt_total >= %d → B; next est >= %d → A\n\n",
		d.Threshold, d.Threshold)

	est := ""
	if d.Estimated {
		est = " (estimated)"
	}
	bd := d.Breakdown
	fmt.Fprintf(&b, "── Next request%s — total %d / window %d ──\n", est, bd.Total(), bd.Window)
	fmt.Fprintf(&b, "%-16s %8s %8s %8s\n", "Component", "Tokens", "% window", "% total")
	writeRow(&b, "System prompt", bd.SystemPrompt, bd)
	writeRow(&b, "Tools", bd.Tools, bd)
	if bd.Rules > 0 {
		fmt.Fprintf(&b, "%-16s %8d  (in system)\n", "Rules", bd.Rules)
	}
	if bd.Skills > 0 {
		fmt.Fprintf(&b, "%-16s %8d\n", "Skills", bd.Skills)
	}
	writeRow(&b, "Subagents", bd.Subagents, bd)
	writeRow(&b, "Conversation", bd.Conversation, bd)
	fmt.Fprintf(&b, "%-16s %8d\n", "Total", bd.Total())
	if d.Session.CompactSummary != "" {
		fmt.Fprintf(&b, "\nCompact summary active (watermark id %d)\n", d.Session.CompactUpToMessageID)
	}
	return b.String()
}

func writeRow(b *strings.Builder, name string, tokens int, bd ctxpkg.ContextBreakdown) {
	fmt.Fprintf(b, "%-16s %8d %7.1f%% %7.1f%%\n",
		name, tokens, bd.PercentOfWindow(tokens), bd.PercentOfTotal(tokens))
}

// ContextPanelJSON is the /context --json export shape.
type ContextPanelJSON struct {
	Session   session.Session        `json:"session"`
	Snapshot  session.UsageSnapshot  `json:"snapshot"`
	Breakdown ctxpkg.ContextBreakdown `json:"breakdown"`
	Threshold int                    `json:"compact_threshold"`
	CostUSD   float64                `json:"estimated_cost_usd"`
}

// FormatContextJSON encodes panel data as indented JSON.
func FormatContextJSON(d ContextPanelData) (string, error) {
	payload := ContextPanelJSON{
		Session:   d.Session,
		Snapshot:  d.Snapshot,
		Breakdown: d.Breakdown,
		Threshold: d.Threshold,
		CostUSD:   billing.EstimateUSD(d.Session.Model, d.Snapshot),
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
