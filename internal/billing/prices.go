package billing

import (
	"sync"
)

var (
	pricesMu          sync.RWMutex
	activePrices      = copyDefaultPrices()
	activeTableVersion = DefaultPriceTableVersion
)

// DefaultPriceTableVersion is the built-in price table id when config does not override it.
// Rates from https://api-docs.deepseek.com/zh-cn/quick_start/pricing (CNY per 1M tokens).
const DefaultPriceTableVersion = "deepseek-official-cny-2026-04-26"

// DefaultSubagentModel is the built-in model for subagent runs (task, session title).
const DefaultSubagentModel = "deepseek-v4-flash"

func defaultPrices() map[string]PerMillion {
	// deepseek-v4-pro: 2.5× promo until 2026-05-31 23:59 Beijing; then input 12 / output 24 / cache hit 0.1.
	return map[string]PerMillion{
		"deepseek-v4-pro":   {Input: 3.0, Output: 6.0, CacheHit: 0.025},
		"deepseek-v4-flash": {Input: 1.0, Output: 2.0, CacheHit: 0.02},
	}
}

func copyDefaultPrices() map[string]PerMillion {
	src := defaultPrices()
	out := make(map[string]PerMillion, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// ModelPriceOverrides is user-level billing overrides (CNY per 1M tokens).
type ModelPriceOverrides struct {
	PriceTableVersion string
	ModelPrices       map[string]PerMillion
}

// Configure applies user-level price overrides on top of built-in defaults.
func Configure(overrides ModelPriceOverrides) {
	pricesMu.Lock()
	defer pricesMu.Unlock()
	activePrices = copyDefaultPrices()
	if overrides.PriceTableVersion != "" {
		activeTableVersion = overrides.PriceTableVersion
	} else {
		activeTableVersion = DefaultPriceTableVersion
	}
	for model, p := range overrides.ModelPrices {
		activePrices[model] = p
	}
}

// ResetPricesForTest restores built-in defaults (tests only).
func ResetPricesForTest() {
	Configure(ModelPriceOverrides{})
}

func lookupPrices(modelID string) (PerMillion, string) {
	pricesMu.RLock()
	defer pricesMu.RUnlock()
	p, ok := activePrices[modelID]
	if !ok {
		p = activePrices[DefaultSubagentModel]
	}
	return p, modelID
}

func currentPriceTableVersion() string {
	pricesMu.RLock()
	defer pricesMu.RUnlock()
	return activeTableVersion
}

// ModelPrices returns the active price table (for tests and introspection).
func ModelPrices() map[string]PerMillion {
	pricesMu.RLock()
	defer pricesMu.RUnlock()
	out := make(map[string]PerMillion, len(activePrices))
	for k, v := range activePrices {
		out[k] = v
	}
	return out
}
