package config_test

import (
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
)

func TestWebConfig_resolve(t *testing.T) {
	w := config.WebConfig{
		FetchModel:         "custom-model",
		FetchCacheTTL:      5 * time.Minute,
		FetchCacheMaxBytes: 1000,
	}
	if w.ResolveFetchModel() != "custom-model" {
		t.Fatal("model")
	}
	if w.ResolveFetchCacheTTL() != 5*time.Minute {
		t.Fatal("ttl")
	}
	if w.ResolveFetchCacheMaxBytes() != 1000 {
		t.Fatal("max bytes")
	}

	empty := config.WebConfig{}
	if empty.ResolveFetchModel() != "deepseek-v4-flash" {
		t.Fatal("default model")
	}
	if empty.ResolveFetchCacheTTL() != 15*time.Minute {
		t.Fatal("default ttl")
	}
	if empty.ResolveFetchCacheMaxBytes() != 52_428_800 {
		t.Fatal("default max bytes")
	}
}
