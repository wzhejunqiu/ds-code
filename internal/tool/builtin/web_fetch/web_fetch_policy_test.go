package web_fetch

import (
	"net"
	"strings"
	"testing"
)

func TestIsBlockedFetchHost_dnsFailureBlocks(t *testing.T) {
	host := "definitely-not-a-real-host.ds-code-ssrf-test.invalid"
	ips, err := net.LookupIP(host)
	if err == nil && len(ips) > 0 {
		t.Skip("environment resolves .invalid names; skipping DNS-failure assertion")
	}
	if !isBlockedFetchHost(host) {
		t.Fatal("DNS failure or empty resolution should block host")
	}
}

func TestValidateFetchURLHost_dnsFailure(t *testing.T) {
	host := "definitely-not-a-real-host.ds-code-ssrf-test.invalid"
	ips, err := net.LookupIP(host)
	if err == nil && len(ips) > 0 {
		t.Skip("environment resolves .invalid names; skipping DNS-failure assertion")
	}
	err = validateFetchURLHost(host, []string{host})
	if err == nil {
		t.Fatal("expected error for unresolvable allowlisted host")
	}
	if !strings.Contains(err.Error(), "不在 web.allowlist") && !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsBlockedFetchHost_loopbackIP(t *testing.T) {
	if !isBlockedFetchHost("127.0.0.1") {
		t.Fatal("127.0.0.1 should be blocked")
	}
	if !isBlockedFetchHost("10.0.0.1") {
		t.Fatal("private IP should be blocked")
	}
	if !isBlockedFetchHost("169.254.169.254") {
		t.Fatal("metadata IP should be blocked")
	}
}
