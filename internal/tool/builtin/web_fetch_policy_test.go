package builtin

import (
	"strings"
	"testing"
)

func TestIsBlockedFetchHost_dnsFailureBlocks(t *testing.T) {
	// Non-existent TLD; lookup should fail and we fail closed.
	if !isBlockedFetchHost("definitely-not-a-real-host.ds-code-ssrf-test.invalid") {
		t.Fatal("DNS failure should block host")
	}
}

func TestValidateFetchURLHost_dnsFailure(t *testing.T) {
	err := validateFetchURLHost("definitely-not-a-real-host.ds-code-ssrf-test.invalid", []string{"definitely-not-a-real-host.ds-code-ssrf-test.invalid"})
	if err == nil {
		t.Fatal("expected error for unresolvable allowlisted host")
	}
	if !strings.Contains(err.Error(), "not in web.allowlist") && !strings.Contains(err.Error(), "blocked") {
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
