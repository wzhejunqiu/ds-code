package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// LoadUserBilling reads billing.model_prices from the user-level config file only.
func LoadUserBilling() (UserBillingConfig, error) {
	path, err := UserConfigPath()
	if err != nil {
		return UserBillingConfig{}, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return UserBillingConfig{}, nil
	}
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return UserBillingConfig{}, fmt.Errorf("config: read user billing %s: %w", path, err)
	}
	var out UserBillingConfig
	if err := v.UnmarshalKey("billing", &out); err != nil {
		return UserBillingConfig{}, fmt.Errorf("config: unmarshal user billing: %w", err)
	}
	if err := validateUserBilling(out); err != nil {
		return UserBillingConfig{}, err
	}
	return out, nil
}

func validateUserBilling(b UserBillingConfig) error {
	for model, p := range b.ModelPrices {
		if model == "" {
			return fmt.Errorf("config: billing.model_prices: empty model id")
		}
		if p.InputPerMillion <= 0 || p.OutputPerMillion <= 0 || p.CacheHitPerMillion <= 0 {
			return fmt.Errorf(
				"config: billing.model_prices.%s: input_per_million, output_per_million, cache_hit_per_million must be > 0",
				model,
			)
		}
	}
	return nil
}

// rejectProjectBilling rejects billing.* in project-level .ds-code/config.yaml.
func rejectProjectBilling(projectRoot string) error {
	path := ProjectConfigPath(projectRoot)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("config: read %s: %w", path, err)
	}
	if v.IsSet("billing") {
		return fmt.Errorf("config: billing.* is only allowed in user config (%s), not project config.yaml", path)
	}
	return nil
}
