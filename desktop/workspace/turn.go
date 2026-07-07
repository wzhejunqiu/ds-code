package workspace

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	desktopbridge "github.com/wzhejunqiu/ds-code/desktop/bridge"
	desktopsys "github.com/wzhejunqiu/ds-code/desktop/sys"
	"github.com/wzhejunqiu/ds-code/internal/agent"
	uislash "github.com/wzhejunqiu/ds-code/internal/ui/slash"
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
	desktopsys.TurnStarted()
	defer func() {
		state.clear()
		desktopsys.TurnFinished("", "", false)
	}()

	turnID := uuid.NewString()
	hub := desktopbridge.NewEmitterHub(turnID, wsID, func(env desktopbridge.AgentEventEnvelope) bool {
		emit(env)
		return true
	})
	hub.EmitTurnStarted(sessionID)
	cb := hub.TurnCallbacks(sessionID)
	prevComplete := cb.OnBackgroundAgentComplete
	cb.OnBackgroundAgentComplete = func(agentID string) {
		if prevComplete != nil {
			prevComplete(agentID)
		}
		desktopsys.Hooks.Notify("ds-code", "Background agent finished")
		if desktopsys.Hooks.BackgroundAgentComplete != nil {
			desktopsys.Hooks.BackgroundAgentComplete(wsID, sessionID, agentID)
		}
	}
	result, err := runner.RunTurn(ctx, sessionID, text, cb)
	hub.EmitTurnDone(result, err)
}

// SendMessage starts an agent turn or handles a slash command for the given workspace and session.
func (m *Manager) SendMessage(wsID, sessionID, text string) (SlashResult, error) {
	ws, err := m.Ensure(wsID)
	if err != nil {
		return SlashResult{}, err
	}
	if ws.runner == nil {
		return SlashResult{}, fmt.Errorf("workspace not ready")
	}
	if _, _, ok := uislash.Parse(text); ok {
		res, err := m.executeSlash(ws, sessionID, text)
		if err != nil {
			return SlashResult{}, err
		}
		if res.Handled && res.Output != "" {
			m.emitSystemNotice(wsID, res.Output)
		}
		return res, nil
	}
	state := m.turnState(wsID)
	if state.isRunning() {
		return SlashResult{}, fmt.Errorf("turn already running")
	}
	emit := m.emitFor(wsID)
	go m.runTurn(wsID, sessionID, text, ws.runner, emit)
	return SlashResult{Handled: false}, nil
}

func (m *Manager) emitSystemNotice(wsID, text string) {
	m.emitFor(wsID)(desktopbridge.AgentEventEnvelope{
		V:           desktopbridge.EnvelopeVersion,
		StreamID:    "main",
		WorkspaceID: wsID,
		Kind:        desktopbridge.KindSystemNotice,
		Critical:    true,
		Payload:     desktopbridge.MustPayload(desktopbridge.SystemNoticePayload{Text: text}),
	})
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
