package builtin

import "testing"

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
}
