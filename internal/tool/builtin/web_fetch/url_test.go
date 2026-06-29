package web_fetch

import (
	"strings"
	"testing"
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

func TestNormalizeURL_upgradesHTTPDefaultPort(t *testing.T) {
	u, err := normalizeURL("http://Example.com/path#frag")
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "https" || u.Host != "example.com" || u.Fragment != "" {
		t.Fatalf("u = %s", u)
	}
}

func TestNormalizeURL_preservesNonStandardHTTPPort(t *testing.T) {
	u, err := normalizeURL("http://127.0.0.1:12345/path")
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "http" || !strings.Contains(u.Host, "12345") {
		t.Fatalf("u = %s", u)
	}
}
