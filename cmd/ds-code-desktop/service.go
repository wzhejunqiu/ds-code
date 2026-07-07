package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wzhejunqiu/ds-code/cmd/ds-code/app"
	"github.com/wzhejunqiu/ds-code/cmd/ds-code/slashcmd"
	desktopbridge "github.com/wzhejunqiu/ds-code/desktop/bridge"
	desktopdatadir "github.com/wzhejunqiu/ds-code/desktop/datadir"
	desktopperm "github.com/wzhejunqiu/ds-code/desktop/permission"
	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permissionmode"
	"github.com/wzhejunqiu/ds-code/internal/session"
)

// DesktopService exposes Wails bindings for the M0 single-workspace PoC.
type DesktopService struct {
	app     *app.App
	runner  *agent.Runner
	store   session.Store
	session string

	mu          sync.Mutex
	turnCancel  context.CancelFunc
	turnRunning bool
	permReg     *desktopperm.Registry
	emit        func(desktopbridge.AgentEventEnvelope)
	emitSeq     uint64
}

func newDesktopService(emit func(desktopbridge.AgentEventEnvelope)) *DesktopService {
	s := &DesktopService{emit: emit}
	s.permReg = desktopperm.NewRegistry(s.emitPermission)
	return s
}

func (s *DesktopService) emitPermission(p desktopbridge.PermissionRequestPayload) {
	s.mu.Lock()
	s.emitSeq++
	seq := s.emitSeq
	emit := s.emit
	s.mu.Unlock()
	if emit == nil {
		return
	}
	emit(desktopbridge.AgentEventEnvelope{
		V:           desktopbridge.EnvelopeVersion,
		Seq:         seq,
		StreamID:    "main",
		WorkspaceID: "default",
		Kind:        desktopbridge.KindPermissionRequest,
		Ts:          time.Now().UnixMilli(),
		Critical:    true,
		Payload:     desktopbridge.MustPayload(p),
	})
}

// OpenProject binds a single workspace/project root and prepares the agent runner.
func (s *DesktopService) OpenProject(root string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return fmt.Errorf("project root not found: %s", abs)
	}

	cfg, err := config.Load(nil, config.Options{
		StartDir:           abs,
		RequireAPIKey:      true,
		SkipProjectDataDir: true,
	})
	if err != nil {
		return err
	}
	dir, err := desktopdatadir.EnsureProjectDataDir(cfg.ProjectRoot)
	if err != nil {
		return err
	}
	cfg.ProjectDataDir = dir

	if s.app != nil {
		s.app.CloseDesktop()
	}
	s.app = app.New(cfg)
	runner, store, _, err := s.app.NewDesktopRunner(io.Discard, s.permReg.Prompter())
	if err != nil {
		return err
	}
	runner.Perm.Interactive = true
	if cfg.Permission.Mode == permissionmode.Ask {
		runner.Perm.Prompter = s.permReg.Prompter()
	}

	ctx := context.Background()
	sess, err := slashcmd.CreateSession(cfg, store)
	if err != nil {
		return err
	}
	if err := slashcmd.SeedGitSnapshot(cfg, ctx, store, sess.ID); err != nil {
		return err
	}

	s.runner = runner
	s.store = store
	s.session = sess.ID
	return nil
}

// SendMessage starts a new agent turn asynchronously.
func (s *DesktopService) SendMessage(text string) error {
	s.mu.Lock()
	if s.runner == nil || s.session == "" {
		s.mu.Unlock()
		return fmt.Errorf("open a project first")
	}
	if s.turnRunning {
		s.mu.Unlock()
		return fmt.Errorf("turn already running")
	}
	runner := s.runner
	sessionID := s.session
	emit := s.emit
	s.turnRunning = true
	s.mu.Unlock()

	go s.runTurn(runner, sessionID, text, emit)
	return nil
}

func (s *DesktopService) runTurn(runner *agent.Runner, sessionID, text string, emit func(desktopbridge.AgentEventEnvelope)) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.turnCancel = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.turnRunning = false
		s.turnCancel = nil
		s.mu.Unlock()
	}()

	turnID := uuid.NewString()
	emitter := desktopbridge.NewStreamEmitter(desktopbridge.StreamEmitterOptions{
		TurnID:      turnID,
		WorkspaceID: "default",
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

// CancelTurn cancels the in-flight turn.
func (s *DesktopService) CancelTurn() error {
	s.mu.Lock()
	cancel := s.turnCancel
	s.mu.Unlock()
	if cancel == nil {
		return fmt.Errorf("no running turn")
	}
	s.permReg.DenyAll()
	cancel()
	return nil
}

// ResolvePermission completes an inline approval card.
func (s *DesktopService) ResolvePermission(requestID string, allow bool) error {
	if !s.permReg.Resolve(requestID, allow) {
		return fmt.Errorf("unknown permission request: %s", requestID)
	}
	return nil
}

// ProjectRoot returns the active project root (for UI status).
func (s *DesktopService) ProjectRoot() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.app == nil || s.app.Cfg == nil {
		return ""
	}
	return s.app.Cfg.ProjectRoot
}
