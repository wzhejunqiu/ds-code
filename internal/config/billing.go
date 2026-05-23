package config

// UserBillingConfig is loaded only from ~/.ds-code/config/config.yaml (not project YAML).
type UserBillingConfig struct {
	PriceTableVersion string                       `mapstructure:"price_table_version"`
	ModelPrices       map[string]ModelPriceConfig   `mapstructure:"model_prices"`
}

// ModelPriceConfig is CNY per 1M tokens for one model.
type ModelPriceConfig struct {
	InputPerMillion    float64 `mapstructure:"input_per_million"`
	OutputPerMillion   float64 `mapstructure:"output_per_million"`
	CacheHitPerMillion float64 `mapstructure:"cache_hit_per_million"`
}
