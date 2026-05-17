//go:build debug

package tui

import "sync"

var (
	testPanicPhase string
	testPanicMu   sync.Mutex
)

// ArmTestPanic schedules a single panic on the next Update (/debug-panic, debug builds only).
func ArmTestPanic(phase string) {
	testPanicMu.Lock()
	testPanicPhase = phase
	testPanicMu.Unlock()
}

func debugBeforeUpdate() {
	testPanicMu.Lock()
	want := testPanicPhase
	testPanicMu.Unlock()
	if want == "update" {
		testPanicMu.Lock()
		testPanicPhase = ""
		testPanicMu.Unlock()
		panic("intentional test panic in update")
	}
}
