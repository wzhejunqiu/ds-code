package billing_test

import (
	"testing"

	"github.com/hejunqiu/ds-code/internal/billing"
	"github.com/hejunqiu/ds-code/internal/session"
)

func TestEstimateUSD_positive(t *testing.T) {
	snap := session.UsageSnapshot{
		PromptTokensTotal:         1_000_000,
		CompletionTokensTotal:     100_000,
		PromptCacheHitTokensTotal: 0,
	}
	cost := billing.EstimateUSD("deepseek-v4-pro", snap)
	if cost <= 0 {
		t.Fatalf("cost = %v", cost)
	}
}
