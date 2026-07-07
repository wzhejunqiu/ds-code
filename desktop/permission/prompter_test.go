package permission_test

import (
	"testing"

	desktopbridge "github.com/wzhejunqiu/ds-code/desktop/bridge"
	desktopperm "github.com/wzhejunqiu/ds-code/desktop/permission"
)

func TestRegistryPrompterRoundTrip(t *testing.T) {
	reqCh := make(chan desktopbridge.PermissionRequestPayload, 1)
	reg := desktopperm.NewRegistry(func(p desktopbridge.PermissionRequestPayload) {
		reqCh <- p
	})
	prompter := reg.Prompter()

	done := make(chan bool, 1)
	go func() {
		allowed, err := prompter("apply_patch", "edit foo.go")
		if err != nil {
			t.Errorf("prompter: %v", err)
		}
		done <- allowed
	}()

	got := <-reqCh
	if got.Tool != "apply_patch" {
		t.Fatalf("tool = %q", got.Tool)
	}
	if !reg.Resolve(got.ID, true) {
		t.Fatal("resolve failed")
	}
	if !<-done {
		t.Fatal("expected allow=true")
	}
}
