package main

import (
	"testing"
	"time"

	desktopworkspace "github.com/wzhejunqiu/ds-code/desktop/workspace"
	uipkg "github.com/wzhejunqiu/ds-code/internal/ui"
)

func newTestManager(t *testing.T) *desktopworkspace.Manager {
	t.Helper()
	m, err := desktopworkspace.NewManagerWithRegistry(&desktopworkspace.Registry{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestExitConfirmerRequiresDoublePress(t *testing.T) {
	svc := &DesktopService{mgr: newTestManager(t)}
	confirmer := newExitConfirmer(svc)

	if confirmer.shouldQuit() {
		t.Fatal("first Cmd+Q should not quit")
	}
	if !confirmer.pending {
		t.Fatal("expected exit confirm armed after first Cmd+Q")
	}
	if !confirmer.shouldQuit() {
		t.Fatal("second Cmd+Q within timeout should quit")
	}
}

func TestExitConfirmerTimesOut(t *testing.T) {
	svc := &DesktopService{mgr: newTestManager(t)}
	confirmer := newExitConfirmer(svc)

	confirmer.shouldQuit()
	confirmer.armedAt = time.Now().Add(-uipkg.ExitConfirmTimeout - time.Millisecond)
	if confirmer.shouldQuit() {
		t.Fatal("Cmd+Q after timeout should re-arm instead of quitting")
	}
	if !confirmer.pending {
		t.Fatal("expected exit confirm armed again after timeout window")
	}
}
