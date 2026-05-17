package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
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
	// Must not match unrelated domains that merely contain the suffix as substring.
	if hostAllowed("notgithub.io", list) {
		t.Fatal("notgithub.io should not match *.github.io")
	}
	if hostAllowed("evil.github.io.attacker.com", list) {
		t.Fatal("suffix trick host should be denied")
	}
}

func TestHostAllowed_wildcardCo(t *testing.T) {
	list := []string{"*.co"}
	if !hostAllowed("foo.co", list) {
		t.Fatal("expected foo.co")
	}
	if hostAllowed("foo.co.uk", list) {
		t.Fatal("foo.co.uk should not match *.co")
	}
}

func TestWebFetch_redirectRevalidatesAllowlist(t *testing.T) {
	allowed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = w.Write([]byte("ok"))
		case "/redirect":
			http.Redirect(w, r, "http://evil.example.net/secret", http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer allowed.Close()

	parsed, err := url.Parse(allowed.URL)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Web: config.WebConfig{
			FetchEnabled: true,
			Allowlist:    []string{parsed.Hostname()},
		},
	}
	tool := &WebFetchTool{Cfg: cfg, Strict: false}

	_, err = tool.Execute(context.Background(), json.RawMessage(`{"url":"`+allowed.URL+`/redirect"}`))
	if err == nil {
		t.Fatal("expected redirect allowlist error")
	}
	if !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("err = %v, want allowlist denial", err)
	}

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"`+allowed.URL+`/ok"}`))
	if err != nil {
		t.Fatalf("allowed fetch: %v", err)
	}
	if !strings.Contains(out, "ok") {
		t.Fatalf("body = %q", out)
	}
}

func TestNewWebFetchClient_rejectsDisallowedRedirectHost(t *testing.T) {
	client := newWebFetchClient([]string{"allowed.test"})
	first, err := http.NewRequest(http.MethodGet, "http://allowed.test/start", nil)
	if err != nil {
		t.Fatal(err)
	}
	next, err := http.NewRequest(http.MethodGet, "http://evil.example.net/next", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = client.CheckRedirect(next, []*http.Request{first})
	if err == nil || !strings.Contains(err.Error(), "evil.example.net") {
		t.Fatalf("CheckRedirect err = %v", err)
	}
}
