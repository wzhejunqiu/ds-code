package billing

import (
	"fmt"

	"github.com/hejunqiu/ds-code/internal/session"
)

// PerMillion holds USD price per 1M tokens (static table; update when vendor pricing changes).
type PerMillion struct {
	Input    float64
	Output   float64
	CacheHit float64
}

// ModelPrices is illustrative DeepSeek V4 pricing for UI estimates only.
var ModelPrices = map[string]PerMillion{
	"deepseek-v4-pro":   {Input: 0.55, Output: 2.19, CacheHit: 0.07},
	"deepseek-v4-flash": {Input: 0.27, Output: 1.10, CacheHit: 0.07},
}

// EstimateUSD returns cumulative session cost in USD from usage totals.
func EstimateUSD(model string, snap session.UsageSnapshot) float64 {
	p, ok := ModelPrices[model]
	if !ok {
		p = ModelPrices["deepseek-v4-pro"]
	}
	inBill := float64(snap.PromptTokensTotal - snap.PromptCacheHitTokensTotal)
	if inBill < 0 {
		inBill = 0
	}
	cache := float64(snap.PromptCacheHitTokensTotal)
	out := float64(snap.CompletionTokensTotal)
	return inBill/1e6*p.Input + cache/1e6*p.CacheHit + out/1e6*p.Output
}

// FormatUSD formats a dollar amount for the status bar.
func FormatUSD(usd float64) string {
	if usd < 0.01 {
		return fmt.Sprintf("$%.4f", usd)
	}
	return fmt.Sprintf("$%.3f", usd)
}
