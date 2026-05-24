package billing_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/billing"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/session"
)

func TestEstimateCNY_positive(t *testing.T) {
	snap := session.UsageSnapshot{
		PromptTokensTotal:     1_000_000,
		CompletionTokensTotal: 500_000,
	}
	cost := billing.EstimateCNY("deepseek-v4-pro", snap)
	if cost <= 0 {
		t.Fatalf("cost = %v", cost)
	}
}

func TestEstimateCNYFromSnapshot_usesFrozenRates(t *testing.T) {
	billing.ResetPricesForTest()
	t.Cleanup(billing.ResetPricesForTest)
	billing.Configure(billing.ModelPriceOverrides{
		ModelPrices: map[string]billing.PerMillion{
			"deepseek-v4-pro": {Input: 99, Output: 99, CacheHit: 99},
		},
	})

	frozen := billing.PriceSnapshot{
		Currency:           billing.CurrencyCNY,
		ModelID:            "deepseek-v4-pro",
		PriceTableVersion:  "test",
		InputPerMillion:    1,
		OutputPerMillion:   1,
		CacheHitPerMillion: 1,
	}
	u := llm.Usage{PromptTokens: 1_000_000, CompletionTokens: 0}
	if got := billing.EstimateCNYFromSnapshot(frozen, u); got != 1.0 {
		t.Fatalf("frozen cost = %v, want 1", got)
	}
	if got := billing.EstimateCNY("deepseek-v4-pro", session.UsageSnapshot{PromptTokensTotal: 1_000_000}); got == 1.0 {
		t.Fatal("current price table should not use frozen snapshot rates")
	}
}

func TestFormatCNY(t *testing.T) {
	if billing.FormatCNY(0.001) != "¥0.0010" {
		t.Fatalf("small: %s", billing.FormatCNY(0.001))
	}
	if billing.FormatCNY(1.2) != "¥1.200" {
		t.Fatalf("large: %s", billing.FormatCNY(1.2))
	}
}
