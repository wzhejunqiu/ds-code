package permission_test

import (
	"testing"

	desktopbridge "github.com/wzhejunqiu/ds-code/desktop/bridge"
	desktopperm "github.com/wzhejunqiu/ds-code/desktop/permission"
	"github.com/wzhejunqiu/ds-code/internal/permission"
)

func TestRegistryWebFetchRoundTrip(t *testing.T) {
	reqCh := make(chan desktopbridge.PermissionRequestPayload, 1)
	reg := desktopperm.NewRegistry(func(p desktopbridge.PermissionRequestPayload) {
		reqCh <- p
	})
	prompter := reg.WebFetchPrompter()

	done := make(chan permission.WebFetchChoice, 1)
	go func() {
		choice, err := prompter("example.com", "https://example.com/")
		if err != nil {
			t.Errorf("prompter: %v", err)
		}
		done <- choice
	}()

	got := <-reqCh
	if got.Kind != "web_fetch" || got.Host != "example.com" {
		t.Fatalf("payload = %+v", got)
	}
	if !reg.ResolveChoice(got.ID, "allow_always") {
		t.Fatal("resolve failed")
	}
	if <-done != permission.WebFetchAllowAlways {
		t.Fatal("expected allow_always")
	}
}

func TestRegistryResolveChoiceWriteShell(t *testing.T) {
	reg := desktopperm.NewRegistry(func(desktopbridge.PermissionRequestPayload) {})
	prompter := reg.Prompter()
	done := make(chan bool, 1)
	go func() {
		allowed, _ := prompter("bash", "ls")
		done <- allowed
	}()
	// need to capture id from emit - use second registry
	reqCh := make(chan string, 1)
	reg2 := desktopperm.NewRegistry(func(p desktopbridge.PermissionRequestPayload) {
		reqCh <- p.ID
	})
	p2 := reg2.Prompter()
	go func() {
		reg2.ResolveChoice(<-reqCh, "allow")
	}()
	allowed, err := p2("bash", "ls")
	if err != nil || !allowed {
		t.Fatalf("allowed=%v err=%v", allowed, err)
	}
	_ = done
}
