//go:build tuitest

package tuitest

import (
	"io"
	"os"
	"testing"

	"github.com/wzhejunqiu/ds-code/cmd/ds-code/app"
	"github.com/wzhejunqiu/ds-code/internal/agent"
	"github.com/wzhejunqiu/ds-code/internal/config"
	ctxpkg "github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/session"
	"github.com/wzhejunqiu/ds-code/internal/tuitest/mockserver"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/deps"
)

// Stack wires production app/runner/TUI deps against a mock LLM server.
type Stack struct {
	App      *app.App
	Runner   *agent.Runner
	Store    session.Store
	CtxSvc   *ctxpkg.Service
	Cfg      *config.Config
	Mock     *mockserver.Server
	Registry *mockserver.Registry
	Project  string
	cleanup  []func()
}

// NewHarness creates a stack for the interactive ds-code-tui-test binary.
func NewHarness() (*Stack, error) {
	return newStackCore(true)
}

// NewStack creates an isolated harness stack for tests.
func NewStack(t testing.TB) (*Stack, error) {
	t.Helper()
	s, err := newStackCore(false)
	if err != nil {
		return nil, err
	}
	t.Cleanup(s.Close)
	t.Cleanup(func() { _ = os.RemoveAll(s.Project) })
	return s, nil
}

func newStackCore(keepProjectOnClose bool) (*Stack, error) {
	dir, err := os.MkdirTemp("", "ds-code-tui-test-*")
	if err != nil {
		return nil, err
	}

	if err := PrepareProjectRoot(dir); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}

	reg := mockserver.NewRegistry()
	mock := mockserver.New(reg)

	cfg, err := HarnessConfig(mock.BaseURL(), dir)
	if err != nil {
		mock.Close()
		_ = os.RemoveAll(dir)
		return nil, err
	}

	application := app.New(cfg)
	runner, store, ctxSvc, err := application.NewRunner(io.Discard)
	if err != nil {
		mock.Close()
		application.Close()
		_ = os.RemoveAll(dir)
		return nil, err
	}

	s := &Stack{
		App:      application,
		Runner:   runner,
		Store:    store,
		CtxSvc:   ctxSvc,
		Cfg:      cfg,
		Mock:     mock,
		Registry: reg,
		Project:  dir,
	}
	s.cleanup = []func(){
		func() { mock.Close() },
		func() { application.Close() },
	}
	if !keepProjectOnClose {
		s.cleanup = append(s.cleanup, func() { _ = os.RemoveAll(dir) })
	}
	return s, nil
}

// Close releases resources.
func (s *Stack) Close() {
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		s.cleanup[i]()
	}
}

// Deps builds tui.Deps; sessionID must already exist.
func (s *Stack) Deps(sessionID string, handleSlash deps.SlashFunc) deps.Deps {
	subStore, _ := s.App.OpenSubagentStore()
	return deps.Deps{
		Cfg:         s.Cfg,
		Runner:      s.Runner,
		Store:       s.Store,
		Subagent:    subStore,
		Context:     s.CtxSvc,
		SessionID:   sessionID,
		Version:     "tui-test",
		HandleSlash: handleSlash,
	}
}
