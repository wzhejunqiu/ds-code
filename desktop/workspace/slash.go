package workspace

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/wzhejunqiu/ds-code/cmd/ds-code/slashcmd"
	"github.com/wzhejunqiu/ds-code/internal/runmode"
	"github.com/wzhejunqiu/ds-code/internal/session"
	uislash "github.com/wzhejunqiu/ds-code/internal/ui/slash"
)

// SlashResult is returned when a slash command is handled.
type SlashResult struct {
	Output       string `json:"output"`
	NewSessionID string `json:"newSessionId,omitempty"`
	Handled      bool   `json:"handled"`
}

// ExecuteSlash runs a slash command line for a workspace session.
func (m *Manager) ExecuteSlash(wsID, sessionID, line string) (SlashResult, error) {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return SlashResult{}, err
	}
	if _, _, ok := uislash.Parse(line); !ok {
		return SlashResult{Handled: false}, nil
	}
	return m.executeSlash(rt, sessionID, line)
}

func (m *Manager) executeSlash(rt *Runtime, sessionID, line string) (SlashResult, error) {
	if rt.runner == nil || rt.store == nil || rt.ctxSvc == nil {
		return SlashResult{}, fmt.Errorf("workspace not ready")
	}
	var buf bytes.Buffer
	sid := sessionID
	env := &slashcmd.Env{
		Ctx:       context.Background(),
		Out:       &buf,
		Cfg:       rt.app.Cfg,
		Runner:    rt.runner,
		Store:     rt.store,
		CtxSvc:    rt.ctxSvc,
		SessionID: &sid,
		Spawn:     rt.app.SpawnRunnerForDesktop(rt.runner),
	}
	handled, err := slashcmd.Handle(env, rt.app, line)
	if err != nil {
		return SlashResult{}, err
	}
	if !handled {
		return SlashResult{Handled: false}, nil
	}
	out := strings.TrimSpace(buf.String())
	res := SlashResult{Output: out, Handled: true}
	if sid != sessionID {
		res.NewSessionID = sid
	}
	return res, nil
}

// SetRunMode switches agent/plan mode for a session and rebuilds tools.
func (m *Manager) SetRunMode(wsID, sessionID, mode string) error {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return err
	}
	rm, err := runmode.Parse(mode)
	if err != nil {
		return err
	}
	if !rm.Configured() {
		return fmt.Errorf("invalid run_mode %q", mode)
	}
	ctx := context.Background()
	if err := rt.store.UpdateSession(ctx, sessionID, func(s *session.Session) error {
		s.RunMode = rm
		return nil
	}); err != nil {
		return err
	}
	rt.app.Cfg.RunMode = rm
	env := &slashcmd.Env{
		Ctx:       ctx,
		Out:       &bytes.Buffer{},
		Cfg:       rt.app.Cfg,
		Runner:    rt.runner,
		Store:     rt.store,
		CtxSvc:    rt.ctxSvc,
		SessionID: &sessionID,
	}
	return rt.app.SetRunMode(ctx, env, mode)
}

// SessionRunMode returns the run mode for a session.
func (m *Manager) SessionRunMode(wsID, sessionID string) (string, error) {
	rt, err := m.Ensure(wsID)
	if err != nil {
		return "", err
	}
	sess, err := rt.store.Get(context.Background(), sessionID)
	if err != nil {
		return "", err
	}
	return sess.RunMode.String(), nil
}
