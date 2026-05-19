package state

import (
	"time"

	"github.com/hejunqiu/ds-code/internal/logging"
)

// WaitTurnsOnExit cancels an active turn and waits for agent goroutines to finish.
func (s *State) WaitTurnsOnExit() {
	if s.TurnCancel != nil {
		s.TurnCancel()
	}
	done := make(chan struct{})
	go func() {
		s.TurnWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		logging.L().Error("timed out waiting for active turn on exit")
	}
}
