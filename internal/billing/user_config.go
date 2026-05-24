package billing

import "github.com/wzhejunqiu/ds-code/internal/config"

// SetupFromUserConfig loads ~/.ds-code/config/config.yaml billing.model_prices and applies overrides.
func SetupFromUserConfig() error {
	ub, err := config.LoadUserBilling()
	if err != nil {
		return err
	}
	overrides := ModelPriceOverrides{PriceTableVersion: ub.PriceTableVersion}
	if len(ub.ModelPrices) > 0 {
		overrides.ModelPrices = make(map[string]PerMillion, len(ub.ModelPrices))
		for model, p := range ub.ModelPrices {
			overrides.ModelPrices[model] = PerMillion{
				Input:    p.InputPerMillion,
				Output:   p.OutputPerMillion,
				CacheHit: p.CacheHitPerMillion,
			}
		}
	}
	Configure(overrides)
	return nil
}
