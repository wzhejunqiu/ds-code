package main

import (
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	uipkg "github.com/wzhejunqiu/ds-code/internal/ui"
)

const (
	exitConfirmHint      = "Press ⌘Q again to exit"
	runningTurnQuitHint  = "Press Esc to cancel the current turn"
	exitWaitTurnsTimeout = 5 * time.Second
)

type exitConfirmer struct {
	mu      sync.Mutex
	pending bool
	armedAt time.Time

	svc *DesktopService
}

func newExitConfirmer(svc *DesktopService) *exitConfirmer {
	return &exitConfirmer{svc: svc}
}

func (e *exitConfirmer) shouldQuit() bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.svc.mgr.HasRunningTurn() {
		e.clearPendingLocked()
		e.emitHintLocked(runningTurnQuitHint)
		return false
	}

	if e.pending && time.Since(e.armedAt) < uipkg.ExitConfirmTimeout {
		e.pending = false
		e.prepareQuitLocked()
		return true
	}

	e.pending = true
	e.armedAt = time.Now()
	e.emitHintLocked(exitConfirmHint)
	go e.armTimeout()
	return false
}

func (e *exitConfirmer) prepareQuitLocked() {
	e.svc.mgr.CancelAllTurns()
	e.svc.mgr.WaitTurns(exitWaitTurnsTimeout)
}

func (e *exitConfirmer) armTimeout() {
	time.Sleep(uipkg.ExitConfirmTimeout)
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.pending {
		return
	}
	if time.Since(e.armedAt) >= uipkg.ExitConfirmTimeout {
		e.clearPendingLocked()
	}
}

func (e *exitConfirmer) clearPendingLocked() {
	if !e.pending {
		return
	}
	e.pending = false
	e.emitHintLocked("")
}

func (e *exitConfirmer) emitHintLocked(text string) {
	app := application.Get()
	if app == nil {
		return
	}
	app.Event.Emit("desktop:hint", map[string]string{"text": text})
}
