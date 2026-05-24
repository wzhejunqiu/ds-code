package billing_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/billing"
	"github.com/wzhejunqiu/ds-code/internal/llm"
)

func TestParseSnapshot_corruptJSONReturnsZeroRates(t *testing.T) {
	billing.ResetPricesForTest()
	t.Cleanup(billing.ResetPricesForTest)
	billing.Configure(billing.ModelPriceOverrides{
		ModelPrices: map[string]billing.PerMillion{
			"deepseek-v4-pro": {Input: 99, Output: 99, CacheHit: 99},
		},
	})

	s := billing.ParseSnapshot("deepseek-v4-pro", "{not-json")
	if s.InputPerMillion != 0 || s.OutputPerMillion != 0 {
		t.Fatalf("corrupt snapshot rates = %+v, want zero", s)
	}
	u := llm.Usage{PromptTokens: 1_000_000}
	if got := billing.EstimateCNYFromSnapshot(s, u); got != 0 {
		t.Fatalf("corrupt snapshot cost = %v, want 0", got)
	}
}

func TestParseSnapshot_emptyUsesCurrentPrices(t *testing.T) {
	billing.ResetPricesForTest()
	t.Cleanup(billing.ResetPricesForTest)
	billing.Configure(billing.ModelPriceOverrides{
		ModelPrices: map[string]billing.PerMillion{
			"deepseek-v4-pro": {Input: 5, Output: 5, CacheHit: 5},
		},
	})

	s := billing.ParseSnapshot("deepseek-v4-pro", "")
	if s.InputPerMillion != 5 {
		t.Fatalf("empty snapshot input = %v, want 5", s.InputPerMillion)
	}
}
