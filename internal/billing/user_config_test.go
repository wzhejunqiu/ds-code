package billing_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/billing"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
)

func TestSetupFromUserConfig(t *testing.T) {
	dir := testutil.IsolatedHome(t)
	userDir := filepath.Join(dir, ".ds-code", "config")
	if err := os.MkdirAll(userDir, 0o700); err != nil {
		t.Fatal(err)
	}
	yaml := `billing:
  price_table_version: test-v1
  model_prices:
    deepseek-v4-pro:
      input_per_million: 7
      output_per_million: 14
      cache_hit_per_million: 1
`
	if err := os.WriteFile(filepath.Join(userDir, "config.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	billing.ResetPricesForTest()
	t.Cleanup(billing.ResetPricesForTest)

	if err := billing.SetupFromUserConfig(); err != nil {
		t.Fatal(err)
	}
	snap := billing.SnapshotForModel("deepseek-v4-pro")
	if snap.InputPerMillion != 7 || snap.PriceTableVersion != "test-v1" {
		t.Fatalf("snap = %+v", snap)
	}
}
