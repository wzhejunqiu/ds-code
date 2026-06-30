package spawn

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
)

func TestCopyWebPermFields_fromParentRuntimeAllowlist(t *testing.T) {
	parent := permission.NewEngine("auto", t.TempDir(), true)
	parent.WebAllowlist = []string{"a.com", "b.com"}
	dst := permission.NewEngine("auto", t.TempDir(), false)
	cfg := &config.Config{Web: config.WebConfig{Allowlist: []string{"cfg-only.test"}}}

	copyWebPermFields(dst, parent, cfg)

	if len(dst.WebAllowlist) != 2 {
		t.Fatalf("WebAllowlist = %v, want 2 entries from parent", dst.WebAllowlist)
	}
	if dst.WebAllowlist[0] != "a.com" || dst.WebAllowlist[1] != "b.com" {
		t.Fatalf("WebAllowlist = %v", dst.WebAllowlist)
	}
}

func TestCopyWebPermFields_fallbackToCfg(t *testing.T) {
	parent := permission.NewEngine("auto", t.TempDir(), true)
	dst := permission.NewEngine("auto", t.TempDir(), false)
	cfg := &config.Config{Web: config.WebConfig{Allowlist: []string{"pkg.go.dev", "example.com"}}}

	copyWebPermFields(dst, parent, cfg)

	if len(dst.WebAllowlist) != 2 {
		t.Fatalf("WebAllowlist = %v, want cfg fallback", dst.WebAllowlist)
	}
	if dst.WebAllowlist[0] != "pkg.go.dev" {
		t.Fatalf("WebAllowlist = %v", dst.WebAllowlist)
	}
}

func TestCopyWebPermFields_copiesPrompter(t *testing.T) {
	parent := permission.NewEngine("auto", t.TempDir(), true)
	parent.WebFetchPrompter = func(host, rawURL string) (permission.WebFetchChoice, error) {
		return permission.WebFetchDeny, nil
	}
	dst := permission.NewEngine("auto", t.TempDir(), false)
	cfg := &config.Config{}

	copyWebPermFields(dst, parent, cfg)

	if dst.WebFetchPrompter == nil {
		t.Fatal("expected WebFetchPrompter copied from parent")
	}
	choice, err := dst.WebFetchPrompter("x.test", "https://x.test/")
	if err != nil {
		t.Fatal(err)
	}
	if choice != permission.WebFetchDeny {
		t.Fatalf("choice = %v, want Deny", choice)
	}
}
