package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/testutil"
)

func TestLoadUserBilling_parsesUserFile(t *testing.T) {
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

	ub, err := LoadUserBilling()
	if err != nil {
		t.Fatal(err)
	}
	if ub.PriceTableVersion != "test-v1" {
		t.Fatalf("version = %q", ub.PriceTableVersion)
	}
	p := ub.ModelPrices["deepseek-v4-pro"]
	if p.InputPerMillion != 7 || p.OutputPerMillion != 14 {
		t.Fatalf("pro = %+v", p)
	}
}

func TestRejectProjectBilling(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	dsCode := filepath.Join(dir, ".ds-code")
	if err := os.MkdirAll(dsCode, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `billing:
  model_prices:
    deepseek-v4-pro:
      input_per_million: 1
      output_per_million: 1
      cache_hit_per_million: 1
`
	if err := os.WriteFile(filepath.Join(dsCode, "config.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := rejectProjectBilling(dir); err == nil {
		t.Fatal("expected error for project billing")
	}
}
