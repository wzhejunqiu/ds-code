package web_fetch_test

import (
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/web_fetch"
)

func TestLRUCache_getPut(t *testing.T) {
	c := web_fetch.NewLRUCache(config.WebConfig{
		FetchCacheTTL:      time.Hour,
		FetchCacheMaxBytes: 1 << 20,
	})
	key := "https://example.com/"
	page := web_fetch.PageBody{Body: []byte("hello"), ContentType: "text/plain", StatusCode: 200}
	c.Put(key, page)
	got := c.Get(key)
	if got == nil || string(got.Body) != "hello" {
		t.Fatal("cache miss")
	}
}

func TestLRUCache_evictsByCompressedSize(t *testing.T) {
	c := web_fetch.NewLRUCache(config.WebConfig{
		FetchCacheTTL:      time.Hour,
		FetchCacheMaxBytes: 300,
	})
	body := []byte(strings.Repeat("a", 200))
	c.Put("https://a.test/", web_fetch.PageBody{Body: body, StatusCode: 200})
	c.Put("https://b.test/", web_fetch.PageBody{Body: body, StatusCode: 200})
	if c.Get("https://a.test/") != nil {
		t.Fatal("expected first entry evicted")
	}
	if c.Get("https://b.test/") == nil {
		t.Fatal("expected second entry present")
	}
}

func TestLRUCache_gzipRoundTrip(t *testing.T) {
	c := web_fetch.NewLRUCache(config.WebConfig{FetchCacheMaxBytes: 1 << 20, FetchCacheTTL: time.Hour})
	raw := []byte("<html>compress me</html>")
	c.Put("https://z.test/", web_fetch.PageBody{Body: raw, ContentType: "text/html", StatusCode: 200})
	got := c.Get("https://z.test/")
	if got == nil || string(got.Body) != string(raw) {
		t.Fatal("gzip round trip failed")
	}
}
