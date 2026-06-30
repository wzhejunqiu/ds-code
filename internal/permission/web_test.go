package permission

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
)

func TestHostAllowed(t *testing.T) {
	list := []string{"example.com", "*.github.io"}
	if !hostAllowed("example.com", list) {
		t.Fatal("expected example.com")
	}
	if !hostAllowed("docs.github.io", list) {
		t.Fatal("expected subdomain github.io")
	}
	if hostAllowed("evil.com", list) {
		t.Fatal("expected deny")
	}
	if hostAllowed("example.com", nil) {
		t.Fatal("empty allowlist should deny")
	}
	if hostAllowed("notgithub.io", list) {
		t.Fatal("notgithub.io should not match *.github.io")
	}
	if hostAllowed("evil.github.io.attacker.com", list) {
		t.Fatal("suffix trick host should be denied")
	}
}

func TestHostAllowed_wildcardCo(t *testing.T) {
	list := []string{"*.co"}
	if hostAllowed("foo.co", list) {
		t.Fatal("short TLD wildcard *.co should be rejected")
	}
	if hostAllowed("foo.co.uk", list) {
		t.Fatal("foo.co.uk should not match *.co")
	}
}

func TestCheckFetchSSRF_loopback(t *testing.T) {
	if err := CheckFetchSSRF("127.0.0.1"); err == nil {
		t.Fatal("expected loopback block")
	}
}

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

func TestEngine_auto_skipsAllowlist(t *testing.T) {
	e := NewEngine(permissionmode.Auto, t.TempDir(), false)
	ctx := WithWebFetchApproval(context.Background())
	if err := e.CheckFetchHost(ctx, "example.com"); err != nil {
		t.Fatalf("auto should skip allowlist: %v", err)
	}
}

func TestEngine_readonly_allowlisted(t *testing.T) {
	e := NewEngine(permissionmode.Readonly, t.TempDir(), false)
	e.WebAllowlist = []string{"example.com"}
	ctx := WithWebFetchApproval(context.Background())
	if err := e.CheckFetchHost(ctx, "example.com"); err != nil {
		t.Fatalf("expected allow: %v", err)
	}
}

func TestEngine_readonly_unlistedNeedsPrompt(t *testing.T) {
	e := NewEngine(permissionmode.Readonly, t.TempDir(), true)
	e.WebFetchPrompter = func(host, rawURL string) (WebFetchChoice, error) {
		return WebFetchAllowOnce, nil
	}
	ctx, err := e.PrepareWebFetch(context.Background(), map[string]any{"url": "https://example.com/"})
	if err != nil {
		t.Fatalf("prompt allow once: %v", err)
	}
	if !isOnceApproved(ctx, "example.com") {
		t.Fatal("expected once approval on ctx")
	}
}

func TestEngine_readonly_nonInteractiveNeedTTY(t *testing.T) {
	e := NewEngine(permissionmode.Readonly, t.TempDir(), false)
	_, err := e.PrepareWebFetch(context.Background(), map[string]any{"url": "https://example.com/"})
	if err != ErrNeedTTY {
		t.Fatalf("err = %v, want ErrNeedTTY", err)
	}
}

func TestEngine_ask_sameAsReadonly(t *testing.T) {
	for _, mode := range []permissionmode.Mode{permissionmode.Readonly, permissionmode.Ask} {
		e := NewEngine(mode, t.TempDir(), true)
		e.WebFetchPrompter = func(host, rawURL string) (WebFetchChoice, error) {
			return WebFetchDeny, nil
		}
		_, err := e.PrepareWebFetch(context.Background(), map[string]any{"url": "https://new.test/"})
		if err != ErrRejected {
			t.Fatalf("mode %s: err = %v, want ErrRejected", mode, err)
		}
	}
}

func TestAppendUniqueAllowlist(t *testing.T) {
	out := appendUniqueAllowlist([]string{"a.test"}, "b.test")
	if len(out) != 2 {
		t.Fatalf("len = %d", len(out))
	}
	out = appendUniqueAllowlist(out, "b.test")
	if len(out) != 2 {
		t.Fatalf("dedup failed: %v", out)
	}
}

func TestNormalizeFetchHost(t *testing.T) {
	if normalizeFetchHost(" Example.COM ") != "example.com" {
		t.Fatal("normalize failed")
	}
	if strings.Contains(normalizeFetchHost("x"), "/") {
		t.Fatal("unexpected")
	}
}
