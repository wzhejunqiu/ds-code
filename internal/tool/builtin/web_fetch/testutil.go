package web_fetch

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"
)

// TestFetchClient returns an HTTP client that dials a test server (test-only; skips SSRF dial checks).
func TestFetchClient(testServerURL string, allowlist []string) *http.Client {
	_ = allowlist
	target, err := url.Parse(testServerURL)
	if err != nil {
		panic(err)
	}
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, network, target.Host)
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// FetchURLWithClient exposes fetchURLWithClient for tests.
func FetchURLWithClient(ctx context.Context, start *url.URL, allowlist []string, client *http.Client) (*FetchOutcome, error) {
	return fetchURLWithClient(ctx, start, allowlist, client)
}
