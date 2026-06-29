package config

import "time"

const defaultFetchModel = "deepseek-v4-flash"

// ResolveFetchModel returns the model for web_fetch page analysis.
func (c WebConfig) ResolveFetchModel() string {
	if c.FetchModel != "" {
		return c.FetchModel
	}
	return defaultFetchModel
}

// ResolveFetchCacheTTL returns cache TTL for web_fetch body cache.
func (c WebConfig) ResolveFetchCacheTTL() time.Duration {
	if c.FetchCacheTTL > 0 {
		return c.FetchCacheTTL
	}
	return 15 * time.Minute
}

// ResolveFetchCacheMaxBytes returns LRU total byte budget (compressed bodies).
func (c WebConfig) ResolveFetchCacheMaxBytes() int {
	if c.FetchCacheMaxBytes > 0 {
		return c.FetchCacheMaxBytes
	}
	return 52_428_800
}
