package permission

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
)

func skipIfHostBlockedBySSRF(t *testing.T, host string) {
	t.Helper()
	if isBlockedFetchHost(host) {
		t.Skipf("host %q blocked by SSRF/DNS in this environment", host)
	}
}

func TestHostAllowed(t *testing.T) {
	skipIfHostBlockedBySSRF(t, "example.com")
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
	skipIfHostBlockedBySSRF(t, "example.com")
	e := NewEngine(permissionmode.Auto, t.TempDir(), false)
	ctx := WithWebFetchApproval(context.Background())
	if err := e.CheckFetchHost(ctx, "example.com"); err != nil {
		t.Fatalf("auto should skip allowlist: %v", err)
	}
}

func TestEngine_readonly_allowlisted(t *testing.T) {
	skipIfHostBlockedBySSRF(t, "example.com")
	e := NewEngine(permissionmode.Readonly, t.TempDir(), false)
	e.WebAllowlist = []string{"example.com"}
	ctx := WithWebFetchApproval(context.Background())
	if err := e.CheckFetchHost(ctx, "example.com"); err != nil {
		t.Fatalf("expected allow: %v", err)
	}
}

func TestEngine_readonly_unlistedNeedsPrompt(t *testing.T) {
	skipIfHostBlockedBySSRF(t, "example.com")
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
	skipIfHostBlockedBySSRF(t, "example.com")
	e := NewEngine(permissionmode.Readonly, t.TempDir(), false)
	_, err := e.PrepareWebFetch(context.Background(), map[string]any{"url": "https://example.com/"})
	if err != ErrNeedTTY {
		t.Fatalf("err = %v, want ErrNeedTTY", err)
	}
}

func TestEngine_ask_sameAsReadonly(t *testing.T) {
	skipIfHostBlockedBySSRF(t, "example.net")
	for _, mode := range []permissionmode.Mode{permissionmode.Readonly, permissionmode.Ask} {
		e := NewEngine(mode, t.TempDir(), true)
		e.WebFetchPrompter = func(host, rawURL string) (WebFetchChoice, error) {
			return WebFetchDeny, nil
		}
		_, err := e.PrepareWebFetch(context.Background(), map[string]any{"url": "https://example.net/"})
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

func TestEngine_CheckWebFetch_routesFromCheck(t *testing.T) {
	skipIfHostBlockedBySSRF(t, "example.com")
	auto := NewEngine(permissionmode.Auto, t.TempDir(), false)
	if err := auto.Check("web_fetch", map[string]any{"url": "https://example.com/"}); err != nil {
		t.Fatalf("auto Check: %v", err)
	}
	if err := auto.CheckWebFetch("https://example.com/"); err != nil {
		t.Fatalf("auto CheckWebFetch: %v", err)
	}

	readonly := NewEngine(permissionmode.Readonly, t.TempDir(), false)
	if err := readonly.Check("web_fetch", map[string]any{"url": "https://example.com/"}); err != ErrNeedTTY {
		t.Fatalf("readonly Check: err = %v, want ErrNeedTTY", err)
	}
	if err := readonly.CheckWebFetch("https://example.com/"); err != ErrNeedTTY {
		t.Fatalf("readonly CheckWebFetch: err = %v, want ErrNeedTTY", err)
	}
}

func TestEngine_allowAlways_updatesMemoryAndSkipsReprompt(t *testing.T) {
	skipIfHostBlockedBySSRF(t, "example.net")
	root := t.TempDir()
	e := NewEngine(permissionmode.Readonly, root, true)
	e.ProjectRoot = root
	prompts := 0
	e.WebFetchPrompter = func(host, rawURL string) (WebFetchChoice, error) {
		prompts++
		return WebFetchAllowAlways, nil
	}

	ctx, err := e.PrepareWebFetch(context.Background(), map[string]any{"url": "https://example.net/"})
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if !hostAllowed("example.net", e.WebAllowlist) {
		t.Fatalf("WebAllowlist = %v, want example.net", e.WebAllowlist)
	}
	b, err := os.ReadFile(config.ProjectConfigPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "example.net") {
		t.Fatalf("config missing example.net: %s", b)
	}

	_, err = e.PrepareWebFetch(ctx, map[string]any{"url": "https://example.net/page"})
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if prompts != 1 {
		t.Fatalf("prompts = %d, want 1", prompts)
	}
}

func TestEngine_allowOnce_sameHostRedirect(t *testing.T) {
	skipIfHostBlockedBySSRF(t, "example.com")
	e := NewEngine(permissionmode.Readonly, t.TempDir(), true)
	e.WebFetchPrompter = func(host, rawURL string) (WebFetchChoice, error) {
		return WebFetchAllowOnce, nil
	}
	ctx, err := e.PrepareWebFetch(context.Background(), map[string]any{"url": "https://example.com/start"})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := e.CheckFetchHost(ctx, "example.com"); err != nil {
		t.Fatalf("first hop: %v", err)
	}
	if err := e.CheckFetchHost(ctx, "example.com"); err != nil {
		t.Fatalf("redirect hop: %v", err)
	}
}
