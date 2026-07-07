package workspace

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	desktopbridge "github.com/wzhejunqiu/ds-code/desktop/bridge"
	"github.com/wzhejunqiu/ds-code/internal/agent"
)

type turnState struct {
	mu          sync.Mutex
	running     bool
	cancel      context.CancelFunc
	waitingPerm bool
}

func (t *turnState) setRunning(cancel context.CancelFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.running = true
	t.cancel = cancel
}

func (t *turnState) clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.running = false
	t.cancel = nil
	t.waitingPerm = false
}

func (t *turnState) cancelTurn() bool {
	t.mu.Lock()
	cancel := t.cancel
	t.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (t *turnState) isRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

func (t *turnState) setWaitingPerm(v bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.waitingPerm = v
}

func (t *turnState) status() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.waitingPerm {
		return "waiting_permission"
	}
	if t.running {
		return "running"
	}
	return "idle"
}

func (m *Manager) runTurn(wsID, sessionID, text string, runner *agent.Runner, emit func(desktopbridge.AgentEventEnvelope)) {
	ctx, cancel := context.WithCancel(context.Background())
	state := m.turnState(wsID)
	state.setRunning(cancel)
	defer state.clear()

	turnID := uuid.NewString()
	emitter := desktopbridge.NewStreamEmitter(desktopbridge.StreamEmitterOptions{
		TurnID:      turnID,
		WorkspaceID: wsID,
		Emit: func(env desktopbridge.AgentEventEnvelope) bool {
			emit(env)
			return true
		},
	})
	desktopbridge.EmitTurnStarted(emitter, sessionID)
	cb := desktopbridge.TurnCallbacks(desktopbridge.TurnCallbacksOptions{
		Emitter:   emitter,
		SessionID: sessionID,
	})
	result, err := runner.RunTurn(ctx, sessionID, text, cb)
	desktopbridge.EmitTurnDone(emitter, result, err)
}

// SendMessage starts an agent turn for the given workspace and session.
func (m *Manager) SendMessage(wsID, sessionID, text string) error {
	ws, err := m.Ensure(wsID)
	if err != nil {
		return err
	}
	if ws.runner == nil {
		return fmt.Errorf("workspace not ready")
	}
	state := m.turnState(wsID)
	if state.isRunning() {
		return fmt.Errorf("turn already running")
	}
	emit := m.emitFor(wsID)
	go m.runTurn(wsID, sessionID, text, ws.runner, emit)
	return nil
}

// CancelTurn cancels the in-flight turn for a workspace.
func (m *Manager) CancelTurn(wsID string) error {
	state := m.turnState(wsID)
	if !state.cancelTurn() {
		return fmt.Errorf("no running turn")
	}
	if reg := m.permReg(wsID); reg != nil {
		reg.DenyAll()
	}
	return nil
}

// TurnStatus returns running / waiting_permission / idle for a workspace.
func (m *Manager) TurnStatus(wsID string) string {
	return m.turnState(wsID).status()
}
