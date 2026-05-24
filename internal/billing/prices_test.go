package billing_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/billing"
	"github.com/wzhejunqiu/ds-code/internal/session"
)

func TestConfigure_overridesDefaults(t *testing.T) {
	billing.ResetPricesForTest()
	t.Cleanup(billing.ResetPricesForTest)

	billing.Configure(billing.ModelPriceOverrides{
		ModelPrices: map[string]billing.PerMillion{
			"deepseek-v4-pro": {Input: 9, Output: 9, CacheHit: 9},
		},
	})
	snap := billing.SnapshotForModel("deepseek-v4-pro")
	if snap.InputPerMillion != 9 {
		t.Fatalf("input = %v", snap.InputPerMillion)
	}
	cost := billing.EstimateCNY("deepseek-v4-pro", session.UsageSnapshot{PromptTokensTotal: 1_000_000})
	if cost != 9.0 {
		t.Fatalf("cost = %v", cost)
	}
}

func TestSnapshotForModel_unknownModelUsesFlashRates(t *testing.T) {
	billing.ResetPricesForTest()
	t.Cleanup(billing.ResetPricesForTest)

	flash := billing.SnapshotForModel("deepseek-v4-flash")
	unknown := billing.SnapshotForModel("unknown-model-xyz")
	if unknown.ModelID != billing.DefaultSubagentModel {
		t.Fatalf("model id = %q, want %q", unknown.ModelID, billing.DefaultSubagentModel)
	}
	if unknown.InputPerMillion != flash.InputPerMillion {
		t.Fatalf("unknown input = %v, flash = %v", unknown.InputPerMillion, flash.InputPerMillion)
	}
	if unknown.OutputPerMillion != flash.OutputPerMillion {
		t.Fatalf("unknown output = %v, flash = %v", unknown.OutputPerMillion, flash.OutputPerMillion)
	}
}
