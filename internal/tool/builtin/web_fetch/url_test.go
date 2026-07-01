package web_fetch

import (
	"strings"
	"testing"
)

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
